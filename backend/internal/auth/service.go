package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

type Repository interface {
	CreateAccount(ctx context.Context, input domain.AccountCreateInput) (domain.Account, error)
	CreateAuthToken(ctx context.Context, accountID string) (string, error)
	DeleteAccount(ctx context.Context, accountID string) bool
	DeleteAuthToken(ctx context.Context, token string) bool
	GetAccountByToken(ctx context.Context, token string) (domain.Account, bool)
	GetAccountByUsername(ctx context.Context, username string) (domain.Account, string, bool)
	ListAccountsPage(ctx context.Context, page int, pageSize int) ([]domain.Account, int)
	UpdateAccountContact(ctx context.Context, accountID string, phone string, email string) (domain.Account, bool)
	UpdateAccountPasswordHash(ctx context.Context, accountID string, passwordHash string) bool
	UpdateAccountProfile(ctx context.Context, accountID string, displayName string, avatarURL string) (domain.Account, bool)
	UpdateAccountStatus(ctx context.Context, accountID string, status string) (domain.Account, bool)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) CreateAccount(ctx context.Context, input domain.AccountCreateInput) (domain.Account, error) {
	return s.repo.CreateAccount(ctx, input)
}

func (s *Service) CreateAuthToken(ctx context.Context, accountID string) (string, error) {
	return s.repo.CreateAuthToken(ctx, accountID)
}

func (s *Service) DeleteAccount(ctx context.Context, accountID string) bool {
	return s.repo.DeleteAccount(ctx, accountID)
}

func (s *Service) DeleteAuthToken(ctx context.Context, token string) bool {
	return s.repo.DeleteAuthToken(ctx, token)
}

func (s *Service) GetAccountByToken(ctx context.Context, token string) (domain.Account, bool) {
	return s.repo.GetAccountByToken(ctx, token)
}

func (s *Service) GetAccountByUsername(ctx context.Context, username string) (domain.Account, string, bool) {
	return s.repo.GetAccountByUsername(ctx, username)
}

func (s *Service) ListAccountsPage(ctx context.Context, page int, pageSize int) ([]domain.Account, int) {
	return s.repo.ListAccountsPage(ctx, page, pageSize)
}

func (s *Service) UpdateAccountContact(ctx context.Context, accountID string, phone string, email string) (domain.Account, bool) {
	return s.repo.UpdateAccountContact(ctx, accountID, phone, email)
}

func (s *Service) UpdateAccountPasswordHash(ctx context.Context, accountID string, passwordHash string) bool {
	return s.repo.UpdateAccountPasswordHash(ctx, accountID, passwordHash)
}

func (s *Service) UpdateAccountProfile(ctx context.Context, accountID string, displayName string, avatarURL string) (domain.Account, bool) {
	return s.repo.UpdateAccountProfile(ctx, accountID, displayName, avatarURL)
}

func (s *Service) UpdateAccountStatus(ctx context.Context, accountID string, status string) (domain.Account, bool) {
	return s.repo.UpdateAccountStatus(ctx, accountID, status)
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func verifyPassword(storedHash string, password string) bool {
	storedHash = strings.TrimSpace(storedHash)
	if strings.HasPrefix(storedHash, "$2a$") || strings.HasPrefix(storedHash, "$2b$") || strings.HasPrefix(storedHash, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil
	}
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:]) == storedHash
}

func shouldUpgradePasswordHash(storedHash string) bool {
	return !strings.HasPrefix(strings.TrimSpace(storedHash), "$2")
}

func validateRegisterInput(username string, password string, displayName string) error {
	if username == "" || password == "" {
		return errText("账号和密码不能为空")
	}
	if len([]rune(username)) < 3 || len([]rune(username)) > 32 {
		return errText("账号长度需要在 3 到 32 个字符之间")
	}
	for _, item := range username {
		if (item >= 'a' && item <= 'z') || (item >= 'A' && item <= 'Z') || (item >= '0' && item <= '9') || item == '_' || item == '-' {
			continue
		}
		return errText("账号只能包含字母、数字、下划线和短横线")
	}
	if len([]rune(password)) < 8 || len([]rune(password)) > 64 {
		return errText("密码长度需要在 8 到 64 个字符之间")
	}
	if len([]rune(displayName)) > 32 {
		return errText("昵称不能超过 32 个字符")
	}
	return nil
}

type errText string

func (e errText) Error() string {
	return string(e)
}

var ErrInvalidCredentials = errors.New("账号或密码错误")
var ErrUsernameExists = errors.New("账号已存在")
var ErrInvalidRegistration = errors.New("注册参数不合法")

func (s *Service) Login(ctx context.Context, username, password string) (domain.Account, string, error) {
	account, hash, ok := s.repo.GetAccountByUsername(ctx, strings.TrimSpace(username))
	if !ok || !verifyPassword(hash, password) {
		return domain.Account{}, "", ErrInvalidCredentials
	}
	if shouldUpgradePasswordHash(hash) {
		if upgraded, err := hashPassword(password); err == nil {
			s.repo.UpdateAccountPasswordHash(ctx, account.AccountID, upgraded)
		}
	}
	token, err := s.repo.CreateAuthToken(ctx, account.AccountID)
	return account, token, err
}
func (s *Service) Register(ctx context.Context, username, password, displayName string) (domain.Account, string, error) {
	username = strings.TrimSpace(username)
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = username
	}
	if err := validateRegisterInput(username, password, displayName); err != nil {
		return domain.Account{}, "", errors.Join(ErrInvalidRegistration, err)
	}
	if _, _, exists := s.repo.GetAccountByUsername(ctx, username); exists {
		return domain.Account{}, "", ErrUsernameExists
	}
	hash, err := hashPassword(password)
	if err != nil {
		return domain.Account{}, "", err
	}
	account, err := s.repo.CreateAccount(ctx, domain.AccountCreateInput{Username: username, PasswordHash: hash, DisplayName: displayName, Role: domain.AccountRoleUser})
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return domain.Account{}, "", ErrUsernameExists
		}
		return domain.Account{}, "", err
	}
	token, err := s.repo.CreateAuthToken(ctx, account.AccountID)
	return account, token, err
}
