package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/auth"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/repository"
)

// defaultRole is the role new users are issued. RBAC enrichment (looking up
// the actual role from user_roles) comes in a later milestone.
const defaultRole = "user"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenReuse         = errors.New("refresh token reuse detected")
)

// TokenPair carries the credentials a successful auth flow returns to a
// client. Both tokens are plaintext strings; the refresh token's hash is
// what's persisted server-side.
type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

type AuthConfig struct {
	JWTSecret  []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type AuthService struct { // like a class's properties, so a service could have access to the repository layer pattern
	users  repository.UsersRepository
	tokens repository.RefreshTokensRepository
	cfg    AuthConfig
}

func NewAuthService(
	users repository.UsersRepository,
	tokens repository.RefreshTokensRepository,
	cfg AuthConfig,
) *AuthService {
	return &AuthService{users: users, tokens: tokens, cfg: cfg}
}

// Register hashes the password, creates the user row, and issues a fresh
// token pair. Any uniqueness violation (username/email already taken) is
// returned wrapped from the repository.
func (s *AuthService) Register(ctx context.Context, username, email, password string) (models.User, TokenPair, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return models.User{}, TokenPair{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.Create(ctx, repository.CreateUserInput{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
	})
	if err != nil {
		return models.User{}, TokenPair{}, err
	}

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return models.User{}, TokenPair{}, err
	}
	return user, pair, nil
}

// Login verifies credentials and issues a token pair. A wrong password and
// an unknown username both return ErrInvalidCredentials so the response
// doesn't leak which usernames exist.
func (s *AuthService) Login(ctx context.Context, username, password string) (models.User, TokenPair, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.User{}, TokenPair{}, ErrInvalidCredentials
		}
		return models.User{}, TokenPair{}, err
	}

	if !auth.VerifyPassword(user.PasswordHash, password) {
		return models.User{}, TokenPair{}, ErrInvalidCredentials
	}

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return models.User{}, TokenPair{}, err
	}
	return user, pair, nil
}

// Refresh implements rotation with reuse detection.
//
//   - If the presented token is unknown -> ErrInvalidCredentials.
//   - If it's already revoked, that means somebody is using a previously
//     rotated token: classic theft signal. Revoke the entire user's session
//     family and return ErrTokenReuse.
//   - If it's expired, treat as invalid.
//   - Otherwise: issue a new pair, then revoke the old token with a
//     replaced_by pointer to the new row's ID for audit.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	hash := auth.HashRefreshToken(refreshToken)

	existing, err := s.tokens.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return TokenPair{}, ErrInvalidCredentials
		}
		return TokenPair{}, err
	}

	now := time.Now()

	if existing.RevokedAt != nil {
		_ = s.tokens.RevokeAllForUser(ctx, existing.UserID)
		return TokenPair{}, ErrTokenReuse
	}
	if !existing.IsActive(now) {
		return TokenPair{}, ErrInvalidCredentials
	}

	user, err := s.users.GetByID(ctx, existing.UserID)
	if err != nil {
		return TokenPair{}, err
	}

	pair, newRow, err := s.issueTokenPairWithRow(ctx, user)
	if err != nil {
		return TokenPair{}, err
	}

	if err := s.tokens.Revoke(ctx, existing.ID, &newRow.ID); err != nil {
		return TokenPair{}, fmt.Errorf("revoke previous refresh token: %w", err)
	}

	return pair, nil
}

// Logout is idempotent: revoking an unknown or already-revoked token
// returns nil so clients don't have to special-case it.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	hash := auth.HashRefreshToken(refreshToken)

	existing, err := s.tokens.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}
	if existing.RevokedAt != nil {
		return nil
	}

	if err := s.tokens.Revoke(ctx, existing.ID, nil); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return err
	}
	return nil
}

// issueTokenPair generates an access JWT and a refresh token, persists the
// refresh hash, and returns the plaintext pair.
func (s *AuthService) issueTokenPair(ctx context.Context, user models.User) (TokenPair, error) {
	pair, _, err := s.issueTokenPairWithRow(ctx, user)
	return pair, err
}

// issueTokenPairWithRow is the internal variant used by Refresh which also
// needs the persisted refresh-token row to write the replaced_by pointer.
func (s *AuthService) issueTokenPairWithRow(ctx context.Context, user models.User) (TokenPair, models.RefreshToken, error) {
	access, err := auth.GenerateAccessToken(
		user.ID.String(),
		user.Username,
		defaultRole,
		s.cfg.JWTSecret,
		s.cfg.AccessTTL,
	)
	if err != nil {
		return TokenPair{}, models.RefreshToken{}, fmt.Errorf("generate access token: %w", err)
	}

	refreshPlain, refreshHash, err := auth.GenerateRefreshToken()
	if err != nil {
		return TokenPair{}, models.RefreshToken{}, fmt.Errorf("generate refresh token: %w", err)
	}

	expiresAt := time.Now().Add(s.cfg.RefreshTTL)

	row, err := s.tokens.Create(ctx, repository.CreateRefreshTokenInput{
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return TokenPair{}, models.RefreshToken{}, err
	}

	return TokenPair{
		AccessToken:      access,
		RefreshToken:     refreshPlain,
		RefreshExpiresAt: expiresAt,
	}, row, nil
}
