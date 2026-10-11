package httpapi

import (
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexisvaleev/mealswipe/backend/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const sessionCookie = "mealswipe_session"

const bcryptCost = 12

// sessionInfo is the validated session stored in the request context.
type sessionInfo = store.Session

func setSessionCookie(w http.ResponseWriter, token string, ttl time.Duration, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		MaxAge:   int(ttl.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		MaxAge:   -1,
	})
}

type registerReq struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// usernameRe: 3..32 chars, starts alphanumeric, then letters/digits/_/./-.
var usernameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{2,31}$`)

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in registerReq
	if !decode(w, r, &in) {
		return
	}
	username := strings.ToLower(strings.TrimSpace(in.Username))
	if !usernameRe.MatchString(username) {
		writeErr(w, http.StatusBadRequest, "validation",
			"username must be 3..32 chars: letters, digits, '_', '.', '-'")
		return
	}
	if len(in.Password) < 8 || len(in.Password) > 72 {
		writeErr(w, http.StatusBadRequest, "validation", "password must be 8..72 characters")
		return
	}
	name := strings.TrimSpace(in.DisplayName)
	if utf8.RuneCountInString(name) > 64 {
		writeErr(w, http.StatusBadRequest, "validation", "display_name too long (max 64)")
		return
	}
	if name == "" {
		name = username
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcryptCost)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", "hash error")
		return
	}
	u, err := s.st.CreateUser(r.Context(), username, string(hash), name)
	if s.storeErr(w, err) {
		return
	}
	s.issueSession(w, r, u, http.StatusCreated)
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in loginReq
	if !decode(w, r, &in) {
		return
	}
	u, err := s.st.UserByUsername(r.Context(), strings.ToLower(strings.TrimSpace(in.Username)))
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid_credentials", "wrong username or password")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		writeErr(w, http.StatusUnauthorized, "invalid_credentials", "wrong username or password")
		return
	}
	s.issueSession(w, r, u, http.StatusOK)
}

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, u *store.User, status int) {
	token, csrf, err := s.st.CreateSession(r.Context(), u.ID, s.cfg.SessionTTL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", "session error")
		return
	}
	setSessionCookie(w, token, s.cfg.SessionTTL, s.cfg.CookieSecure)
	writeJSON(w, status, map[string]any{"user": u, "csrf": csrf})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if tok, _ := r.Context().Value(ctxToken).(string); tok != "" {
		_ = s.st.DeleteSession(r.Context(), tok)
	}
	clearSessionCookie(w, s.cfg.CookieSecure)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	u, err := s.st.UserByID(r.Context(), sess.UserID)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u, "csrf": sess.CSRF})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": Version,
		"app":     s.cfg.AppName,
	})
}
