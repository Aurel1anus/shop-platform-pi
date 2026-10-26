package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

func (s *MySQLStore) GetAccountByUsername(ctx context.Context, username string) (domain.Account, string, bool) {
	var account domain.Account
	var role string
	var passwordHash string
	err := s.db.QueryRowContext(ctx, `
		SELECT account_id, username, password_hash, display_name, avatar_url, phone, email, role, merchant_id, status, created_at
		FROM accounts
		WHERE username = ? AND status = 'active'
	`, username).Scan(
		&account.AccountID,
		&account.Username,
		&passwordHash,
		&account.DisplayName,
		&account.AvatarURL,
		&account.Phone,
		&account.Email,
		&role,
		&account.MerchantID,
		&account.Status,
		&account.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Account{}, "", false
	}
	account.Role = domain.AccountRole(role)
	return account, passwordHash, err == nil
}

func (s *MySQLStore) GetAccount(ctx context.Context, accountID string) (domain.Account, bool) {
	return s.getAccountByID(ctx, accountID)
}

func (s *MySQLStore) GetAccountByToken(ctx context.Context, token string) (domain.Account, bool) {
	var account domain.Account
	var role string
	tokenHash := authTokenHash(token)
	err := s.db.QueryRowContext(ctx, `
		SELECT a.account_id, a.username, a.display_name, a.avatar_url, a.phone, a.email, a.role, a.merchant_id, a.status, a.created_at
		FROM auth_tokens t
		JOIN accounts a ON a.account_id = t.account_id
		WHERE t.token IN (?, ?) AND t.expires_at > ? AND a.status IN ('active', 'risk')
	`, tokenHash, strings.TrimSpace(token), time.Now()).Scan(
		&account.AccountID,
		&account.Username,
		&account.DisplayName,
		&account.AvatarURL,
		&account.Phone,
		&account.Email,
		&role,
		&account.MerchantID,
		&account.Status,
		&account.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Account{}, false
	}
	account.Role = domain.AccountRole(role)
	return account, err == nil
}

func (s *MySQLStore) ListAccounts(ctx context.Context) []domain.Account {
	rows, err := s.db.QueryContext(ctx, `
		SELECT account_id, username, display_name, role, merchant_id, status, created_at
		FROM accounts
		ORDER BY created_at, account_id
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	items := make([]domain.Account, 0)
	for rows.Next() {
		var item domain.Account
		var role string
		if err := rows.Scan(&item.AccountID, &item.Username, &item.DisplayName, &role, &item.MerchantID, &item.Status, &item.CreatedAt); err != nil {
			return nil
		}
		item.Role = domain.AccountRole(role)
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) ListAccountsPage(ctx context.Context, page int, pageSize int) ([]domain.Account, int) {
	page, pageSize = normalizePage(page, pageSize)
	total := s.countRows(ctx, "accounts")
	rows, err := s.db.QueryContext(ctx, `
		SELECT account_id, username, display_name, role, merchant_id, status, created_at
		FROM accounts
		ORDER BY created_at, account_id
		LIMIT ? OFFSET ?
	`, pageSize, pageOffset(page, pageSize))
	if err != nil {
		return nil, 0
	}
	defer rows.Close()

	items := make([]domain.Account, 0)
	for rows.Next() {
		var item domain.Account
		var role string
		if err := rows.Scan(&item.AccountID, &item.Username, &item.DisplayName, &role, &item.MerchantID, &item.Status, &item.CreatedAt); err != nil {
			return nil, 0
		}
		item.Role = domain.AccountRole(role)
		items = append(items, item)
	}
	return items, total
}

func (s *MySQLStore) CreateAccount(ctx context.Context, input domain.AccountCreateInput) (domain.Account, error) {
	now := time.Now()
	role := strings.TrimSpace(string(input.Role))
	if role == "" {
		role = string(domain.AccountRoleUser)
	}
	account := domain.Account{
		AccountID:   nextID("acct"),
		Username:    strings.TrimSpace(input.Username),
		DisplayName: strings.TrimSpace(input.DisplayName),
		AvatarURL:   strings.TrimSpace(input.AvatarURL),
		Phone:       strings.TrimSpace(input.Phone),
		Email:       strings.TrimSpace(input.Email),
		Role:        domain.AccountRole(role),
		MerchantID:  strings.TrimSpace(input.MerchantID),
		Status:      domain.AccountStatusActive,
		CreatedAt:   now,
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (account_id, username, password_hash, display_name, avatar_url, phone, email, role, merchant_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)
	`, account.AccountID, account.Username, input.PasswordHash, account.DisplayName, account.AvatarURL, account.Phone, account.Email, role, account.MerchantID, now, now)
	if err != nil {
		if isDuplicateKeyError(err) {
			return domain.Account{}, domain.ErrAlreadyExists
		}
		return domain.Account{}, fmt.Errorf("insert account: %w", err)
	}
	return account, nil
}

func (s *MySQLStore) UpdateAccountPasswordHash(ctx context.Context, accountID string, passwordHash string) bool {
	if strings.TrimSpace(accountID) == "" || strings.TrimSpace(passwordHash) == "" {
		return false
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE accounts
		SET password_hash = ?, updated_at = ?
		WHERE account_id = ? AND status IN ('active', 'risk')
	`, passwordHash, time.Now(), accountID)
	if err != nil {
		return false
	}
	affected, err := result.RowsAffected()
	return err == nil && affected > 0
}

func (s *MySQLStore) UpdateAccountStatus(ctx context.Context, accountID string, status string) (domain.Account, bool) {
	result, err := s.db.ExecContext(ctx, `UPDATE accounts SET status = ?, updated_at = ? WHERE account_id = ?`, status, time.Now(), accountID)
	if err != nil {
		return domain.Account{}, false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.Account{}, false
	}
	var account domain.Account
	var role string
	err = s.db.QueryRowContext(ctx, `
		SELECT account_id, username, display_name, avatar_url, phone, email, role, merchant_id, status, created_at
		FROM accounts
		WHERE account_id = ?
	`, accountID).Scan(&account.AccountID, &account.Username, &account.DisplayName, &account.AvatarURL, &account.Phone, &account.Email, &role, &account.MerchantID, &account.Status, &account.CreatedAt)
	if err != nil {
		return domain.Account{}, false
	}
	account.Role = domain.AccountRole(role)
	return account, true
}

func (s *MySQLStore) UpdateAccountProfile(ctx context.Context, accountID string, displayName string, avatarURL string) (domain.Account, bool) {
	sets := []string{"updated_at = ?"}
	args := []any{time.Now()}
	if strings.TrimSpace(displayName) != "" {
		sets = append(sets, "display_name = ?")
		args = append(args, truncateRunes(strings.TrimSpace(displayName), 24))
	}
	if strings.TrimSpace(avatarURL) != "" {
		sets = append(sets, "avatar_url = ?")
		args = append(args, truncateRunes(strings.TrimSpace(avatarURL), 512))
	}
	args = append(args, accountID)
	result, err := s.db.ExecContext(ctx, `UPDATE accounts SET `+strings.Join(sets, ", ")+` WHERE account_id = ? AND status = 'active'`, args...)
	if err != nil {
		return domain.Account{}, false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.Account{}, false
	}
	return s.getAccountByID(ctx, accountID)
}

func (s *MySQLStore) UpdateAccountContact(ctx context.Context, accountID string, phone string, email string) (domain.Account, bool) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE accounts
		SET phone = ?, email = ?, updated_at = ?
		WHERE account_id = ? AND status = 'active'
	`, truncateRunes(strings.TrimSpace(phone), 32), truncateRunes(strings.TrimSpace(email), 128), time.Now(), accountID)
	if err != nil {
		return domain.Account{}, false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.Account{}, false
	}
	return s.getAccountByID(ctx, accountID)
}

func (s *MySQLStore) DeleteAuthToken(ctx context.Context, token string) bool {
	if strings.TrimSpace(token) == "" {
		return false
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_tokens WHERE token IN (?, ?)`, authTokenHash(token), strings.TrimSpace(token))
	return err == nil
}

func (s *MySQLStore) DeleteAuthTokensByAccount(ctx context.Context, accountID string) bool {
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_tokens WHERE account_id = ?`, accountID)
	return err == nil
}

func (s *MySQLStore) DeleteAccount(ctx context.Context, accountID string) bool {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
		UPDATE accounts
		SET status = 'deleted', deleted_at = ?, updated_at = ?
		WHERE account_id = ? AND status = 'active'
	`, time.Now(), time.Now(), accountID)
	if err != nil {
		return false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return false
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM auth_tokens WHERE account_id = ?`, accountID); err != nil {
		return false
	}
	return tx.Commit() == nil
}

func (s *MySQLStore) getAccountByID(ctx context.Context, accountID string) (domain.Account, bool) {
	var account domain.Account
	var role string
	err := s.db.QueryRowContext(ctx, `
		SELECT account_id, username, display_name, avatar_url, phone, email, role, merchant_id, status, created_at
		FROM accounts
		WHERE account_id = ?
	`, accountID).Scan(&account.AccountID, &account.Username, &account.DisplayName, &account.AvatarURL, &account.Phone, &account.Email, &role, &account.MerchantID, &account.Status, &account.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Account{}, false
	}
	account.Role = domain.AccountRole(role)
	return account, err == nil
}

func (s *MySQLStore) CreateAuthToken(ctx context.Context, accountID string) (string, error) {
	token, err := newAuthToken()
	if err != nil {
		return "", err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO auth_tokens (token, account_id, created_at, expires_at)
		VALUES (?, ?, ?, ?)
	`, authTokenHash(token), accountID, time.Now(), time.Now().Add(24*time.Hour))
	if err != nil {
		return "", fmt.Errorf("insert auth token: %w", err)
	}
	return token, nil
}
