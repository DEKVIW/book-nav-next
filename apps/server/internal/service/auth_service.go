package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/apperr"
	"github.com/booknav/book-nav/apps/server/internal/pkg/password"
	"github.com/booknav/book-nav/apps/server/internal/repository"
	"github.com/google/uuid"
)

type AuthService struct {
	users    *repository.UserRepo
	sessions *repository.SessionRepo
	invites  *repository.InvitationRepo
}

func NewAuthService(users *repository.UserRepo, sessions *repository.SessionRepo, invites *repository.InvitationRepo) *AuthService {
	return &AuthService{users: users, sessions: sessions, invites: invites}
}

type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

type RegisterInput struct {
	Username       string `json:"username"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	InvitationCode string `json:"invitation_code"`
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (*domain.User, *domain.Session, error) {
	in.Username = strings.TrimSpace(in.Username)
	if in.Username == "" || in.Password == "" {
		return nil, nil, apperr.New(apperr.Validation, "用户名和密码不能为空")
	}
	u, err := s.users.GetByUsername(ctx, in.Username)
	if err != nil {
		return nil, nil, err
	}
	if u == nil || !password.Check(u.PasswordHash, in.Password) {
		return nil, nil, apperr.New(apperr.Unauthorized, "用户名或密码错误")
	}
	ttl := 24 * time.Hour
	if in.Remember {
		ttl = 30 * 24 * time.Hour
	}
	sess, err := s.createSession(ctx, u.ID, ttl)
	if err != nil {
		return nil, nil, err
	}
	return u, sess, nil
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*domain.User, error) {
	// Personal bookmark nav: public registration disabled
	_ = in
	return nil, apperr.New(apperr.Forbidden, "注册已关闭")
}

func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.sessions.Delete(ctx, sessionID)
}

func (s *AuthService) UserFromSession(ctx context.Context, sessionID string) (*domain.User, *domain.Session, error) {
	if sessionID == "" {
		return nil, nil, nil
	}
	sess, err := s.sessions.Get(ctx, sessionID)
	if err != nil || sess == nil {
		return nil, nil, err
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		_ = s.sessions.Delete(ctx, sessionID)
		return nil, nil, nil
	}
	u, err := s.users.GetByID(ctx, sess.UserID)
	if err != nil || u == nil {
		return nil, nil, err
	}
	return u, sess, nil
}

func (s *AuthService) createSession(ctx context.Context, userID int64, ttl time.Duration) (*domain.Session, error) {
	csrf, err := randomToken(16)
	if err != nil {
		return nil, err
	}
	sess := &domain.Session{
		ID:        uuid.NewString(),
		UserID:    userID,
		CSRFToken: csrf,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}
	if err := s.sessions.Create(ctx, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
