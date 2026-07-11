package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const SessionCookieName = "docker_manager_session"

type User struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type session struct {
	User      User      `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

type SessionManager struct {
	mu sync.Mutex

	sessions map[string]session

	ttl          time.Duration
	cookieSecure bool
	redis        *redis.Client
}

func NewSessionManager(
	ttl time.Duration,
	cookieSecure bool,
	redisAddress string,
) *SessionManager {
	manager := &SessionManager{
		sessions:     make(map[string]session),
		ttl:          ttl,
		cookieSecure: cookieSecure,
	}
	if redisAddress != "" {
		manager.redis = redis.NewClient(&redis.Options{Addr: redisAddress})
	}
	return manager
}

func (m *SessionManager) Create(
	w http.ResponseWriter,
	user User,
) error {
	token, err := generateSessionToken()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	expiresAt := now.Add(m.ttl)

	current := session{
		User:      user,
		ExpiresAt: expiresAt,
	}
	key := sessionKey(token)
	if m.redis != nil {
		payload, marshalErr := json.Marshal(current)
		if marshalErr != nil {
			return fmt.Errorf("encode session: %w", marshalErr)
		}
		if redisErr := m.redis.Set(context.Background(), "session:"+key, payload, m.ttl).Err(); redisErr != nil {
			return fmt.Errorf("store session in Redis: %w", redisErr)
		}
	} else {
		m.mu.Lock()
		m.deleteExpiredLocked(now)
		m.sessions[key] = current
		m.mu.Unlock()
	}

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     SessionCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   m.cookieSecure,
			SameSite: http.SameSiteStrictMode,

			Expires: expiresAt,
			MaxAge:  int(m.ttl.Seconds()),
		},
	)

	return nil
}

func (m *SessionManager) Current(
	r *http.Request,
) (User, bool) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return User{}, false
	}

	key := sessionKey(cookie.Value)
	now := time.Now().UTC()
	if m.redis != nil {
		payload, redisErr := m.redis.Get(r.Context(), "session:"+key).Bytes()
		if redisErr != nil {
			if redisErr != redis.Nil {
				log.Printf("read Redis session: %v", redisErr)
			}
			return User{}, false
		}
		var currentSession session
		if jsonErr := json.Unmarshal(payload, &currentSession); jsonErr != nil || !currentSession.ExpiresAt.After(now) {
			return User{}, false
		}
		return currentSession.User, true
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	currentSession, exists := m.sessions[key]
	if !exists {
		return User{}, false
	}

	if !currentSession.ExpiresAt.After(now) {
		delete(m.sessions, key)
		return User{}, false
	}

	return currentSession.User, true
}

func (m *SessionManager) Revoke(
	r *http.Request,
) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return
	}

	key := sessionKey(cookie.Value)
	if m.redis != nil {
		if redisErr := m.redis.Del(r.Context(), "session:"+key).Err(); redisErr != nil {
			log.Printf("revoke Redis session: %v", redisErr)
		}
		return
	}
	m.mu.Lock()
	delete(m.sessions, key)
	m.mu.Unlock()
}

func (m *SessionManager) Close() error {
	if m.redis != nil {
		return m.redis.Close()
	}
	return nil
}

func (m *SessionManager) Destroy(
	w http.ResponseWriter,
	r *http.Request,
) {
	m.Revoke(r)

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     SessionCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   m.cookieSecure,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   -1,
			Expires:  time.Unix(1, 0),
		},
	)
}

func (m *SessionManager) deleteExpiredLocked(
	now time.Time,
) {
	for key, currentSession := range m.sessions {
		if !currentSession.ExpiresAt.After(now) {
			delete(m.sessions, key)
		}
	}
}

func generateSessionToken() (string, error) {
	tokenBytes := make([]byte, 32)

	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf(
			"generate session token: %w",
			err,
		)
	}

	return base64.RawURLEncoding.EncodeToString(
		tokenBytes,
	), nil
}

func sessionKey(token string) string {
	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:])
}
