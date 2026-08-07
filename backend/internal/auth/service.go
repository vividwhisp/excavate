package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"excavate/internal/cache"
	"excavate/internal/store"
)

var (
	ErrEmailExists   = errors.New("email already registered")
	ErrInvalidCreds  = errors.New("invalid email or password")
	ErrPasswordShort = errors.New("password must be at least 8 characters")
)

type Service struct {
	users    *store.UserRepo
	sessions *cache.SessionStore
}

func NewService(users *store.UserRepo, sessions *cache.SessionStore) *Service {
	return &Service{users: users, sessions: sessions}
}

// Register creates a user and returns a fresh session token.
func (s *Service) Register(ctx context.Context, email, password string) (user store.User, token string, err error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(password) < 8 {
		return user, "", ErrPasswordShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return user, "", fmt.Errorf("hash password: %w", err)
	}

	user, err = s.users.Create(ctx, email, string(hash))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return user, "", ErrEmailExists
		}
		return user, "", fmt.Errorf("create user: %w", err)
	}

	token, err = s.newSession(ctx, user.ID)
	return user, token, err
}

// Login validates credentials and returns a fresh session token.
func (s *Service) Login(ctx context.Context, email, password string) (user store.User, token string, err error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err = s.users.ByEmail(ctx, email)
	if err != nil {
		if store.IsNotFound(err) {
			return user, "", ErrInvalidCreds
		}
		return user, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		// Same error whether the email or password is wrong, so attackers
		// cannot tell which field they guessed.
		return user, "", ErrInvalidCreds
	}

	token, err = s.newSession(ctx, user.ID)
	return user, token, err
}

// UserIDByToken resolves a session token to a user ID.
func (s *Service) UserIDByToken(ctx context.Context, token string) (string, error) {
	return s.sessions.Get(ctx, token)
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, token)
}

func (s *Service) newSession(ctx context.Context, userID string) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}
	if err := s.sessions.Create(ctx, token, userID); err != nil {
		return "", err
	}
	return token, nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}