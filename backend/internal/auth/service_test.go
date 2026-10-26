package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

type authRepo struct {
	Repository
	hash     string
	upgraded string
	tokens   int
}

func (r *authRepo) GetAccountByUsername(context.Context, string) (domain.Account, string, bool) {
	return domain.Account{AccountID: "user-a"}, r.hash, true
}
func (r *authRepo) UpdateAccountPasswordHash(_ context.Context, _ string, hash string) bool {
	r.upgraded = hash
	return true
}
func (r *authRepo) CreateAuthToken(context.Context, string) (string, error) {
	r.tokens++
	return "test-token", nil
}

func TestLoginUpgradesLegacyHashAndRejectsWrongPassword(t *testing.T) {
	sum := sha256.Sum256([]byte("a-valid-password"))
	repo := &authRepo{hash: hex.EncodeToString(sum[:])}
	svc := New(repo)
	_, token, err := svc.Login(context.Background(), "user", "a-valid-password")
	if err != nil || token == "" || !verifyPassword(repo.upgraded, "a-valid-password") || shouldUpgradePasswordHash(repo.upgraded) {
		t.Fatalf("legacy hash migration failed: %v", err)
	}
	_, _, err = svc.Login(context.Background(), "user", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) || repo.tokens != 1 {
		t.Fatal("wrong password issued a token")
	}
}

type registrationRaceRepo struct{ Repository }

func (*registrationRaceRepo) GetAccountByUsername(context.Context, string) (domain.Account, string, bool) {
	return domain.Account{}, "", false
}

func (*registrationRaceRepo) CreateAccount(context.Context, domain.AccountCreateInput) (domain.Account, error) {
	return domain.Account{}, domain.ErrAlreadyExists
}

func TestConcurrentRegistrationStillReturnsUsernameConflict(t *testing.T) {
	_, _, err := New(&registrationRaceRepo{}).Register(context.Background(), "newuser", "valid-password", "用户")
	if !errors.Is(err, ErrUsernameExists) {
		t.Fatalf("expected username conflict, got %v", err)
	}
}
