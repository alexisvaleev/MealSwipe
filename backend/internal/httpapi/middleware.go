package httpapi

import (
	"context"
	"crypto/subtle"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// --- security headers (SPEC §11) ---

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' https: data:")
		next.ServeHTTP(w, r)
	})
}

// bodyCap limits request bodies to 1 MiB on mutating methods.
func bodyCap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mutating(r.Method) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		}
		next.ServeHTTP(w, r)
	})
}

func mutating(m string) bool {
	switch m {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// --- Origin check ---

// originCheck enforces: when the Origin header is present it must match the
// request Host or be in ALLOWED_ORIGINS (SPEC §5).
func (s *Server) originCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o := r.Header.Get("Origin")
		if o == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !s.originAllowed(o, r.Host) {
			writeErr(w, http.StatusForbidden, "forbidden_origin", "origin not allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) originAllowed(origin, host string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	// Compare hostnames (cookies are host-scoped, not port-scoped); a reverse
	// proxy may legitimately forward a Host without the original port.
	reqHost := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		reqHost = h
	}
	if strings.EqualFold(u.Hostname(), reqHost) {
		return true
	}
	for _, a := range s.cfg.AllowedOrigins {
		if strings.EqualFold(a, origin) || strings.EqualFold(a, u.Host) ||
			strings.EqualFold(a, u.Scheme+"://"+u.Host) {
			return true
		}
	}
	return false
}

// --- per-IP rate limiting (x/time/rate) ---

type ipEntry struct {
	lim  *rate.Limiter
	last time.Time
}

type ipRate struct {
	mu    sync.Mutex
	m     map[string]*ipEntry
	per   rate.Limit
	burst int
}

func newIPRate(perMinute, burst int) *ipRate {
	return &ipRate{
		m:     map[string]*ipEntry{},
		per:   rate.Every(time.Minute / time.Duration(perMinute)),
		burst: burst,
	}
}

// clientIP prefers X-Real-IP (set by the bundled nginx reverse proxy) and
// falls back to RemoteAddr.
func clientIP(r *http.Request) string {
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (l *ipRate) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.m[ip]
	if !ok {
		e = &ipEntry{lim: rate.NewLimiter(l.per, l.burst)}
		l.m[ip] = e
	}
	e.last = time.Now()
	// Lazy sweep: when the map grows, drop idle entries.
	if len(l.m) > 4096 {
		cutoff := time.Now().Add(-10 * time.Minute)
		for k, v := range l.m {
			if v.last.Before(cutoff) {
				delete(l.m, k)
			}
		}
	}
	return e.lim.Allow()
}

func (l *ipRate) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r)) {
			writeErr(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- session auth + CSRF ---

type ctxKey int

const (
	ctxSession ctxKey = iota
	ctxToken
)

// requireAuth validates the mealswipe_session cookie (SHA-256 lookup),
// applies sliding expiry (<half TTL left -> renew), and enforces the
// session-bound CSRF token on mutating methods.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil || c.Value == "" {
			writeErr(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		sess, err := s.st.SessionByToken(r.Context(), c.Value)
		if err != nil {
			clearSessionCookie(w, s.cfg.CookieSecure)
			writeErr(w, http.StatusUnauthorized, "unauthorized", "invalid session")
			return
		}
		if time.Now().After(sess.ExpiresAt) {
			_ = s.st.DeleteSession(r.Context(), c.Value)
			clearSessionCookie(w, s.cfg.CookieSecure)
			writeErr(w, http.StatusUnauthorized, "unauthorized", "session expired")
			return
		}
		if mutating(r.Method) {
			tok := r.Header.Get("X-CSRF-Token")
			if tok == "" {
				tok = r.Header.Get("X-CSRF-Header") // accepted alias
			}
			if tok == "" || subtle.ConstantTimeCompare([]byte(tok), []byte(sess.CSRF)) != 1 {
				writeErr(w, http.StatusForbidden, "csrf_mismatch", "invalid CSRF token")
				return
			}
		}
		// Sliding expiry: renew when less than half the TTL remains.
		if time.Until(sess.ExpiresAt) < s.cfg.SessionTTL/2 {
			exp := time.Now().Add(s.cfg.SessionTTL)
			if err := s.st.RenewSession(r.Context(), c.Value, exp); err == nil {
				setSessionCookie(w, c.Value, s.cfg.SessionTTL, s.cfg.CookieSecure)
			}
		}
		ctx := context.WithValue(r.Context(), ctxSession, sess)
		ctx = context.WithValue(ctx, ctxToken, c.Value)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func sessionFrom(r *http.Request) *sessionInfo {
	sess, _ := r.Context().Value(ctxSession).(*sessionInfo)
	return sess
}
