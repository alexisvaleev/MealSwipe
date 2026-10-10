package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexisvaleev/mealswipe/backend/internal/store"
)

const maxBody = 1 << 20 // 1 MiB JSON body cap (SPEC §5)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

type errBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": errBody{Code: code, Message: msg}})
}

// storeErr maps store sentinel errors onto the error envelope.
// Unexpected errors are logged (never echoed to the client).
func (s *Server) storeErr(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, store.ErrConflict):
		writeErr(w, http.StatusConflict, "conflict", "slot already occupied")
	case errors.Is(err, store.ErrEmailTaken):
		writeErr(w, http.StatusConflict, "email_taken", "email already registered")
	default:
		s.log.Error("internal", "err", err)
		writeErr(w, http.StatusInternalServerError, "internal", "internal error")
	}
	return true
}

// decode reads a JSON body (≤1 MiB). It writes the error response itself and
// reports whether decoding succeeded.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		var mb *http.MaxBytesError
		if errors.As(err, &mb) {
			writeErr(w, http.StatusRequestEntityTooLarge, "too_large", "body exceeds 1 MiB")
		} else {
			writeErr(w, http.StatusBadRequest, "invalid_json", "malformed JSON body")
		}
		return false
	}
	return true
}

func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// parseDate reads a YYYY-MM-DD query/body date and normalises it to the
// Monday of that week (menus are keyed by Monday).
func monday(t time.Time) time.Time {
	t = t.UTC().Truncate(24 * time.Hour)
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	return t.AddDate(0, 0, -(wd - 1))
}

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(s))
}

func uuidLike(s string) bool { return len(s) == 36 }

// requireUUID 404s when a path value can never match a uuid column.
func requireUUID(w http.ResponseWriter, r *http.Request, keys ...string) bool {
	for _, k := range keys {
		if !uuidLike(r.PathValue(k)) {
			writeErr(w, http.StatusNotFound, "not_found", "not found")
			return false
		}
	}
	return true
}
