package httpapi

import (
	"net/http"
	"strings"
	"unicode/utf8"
)

// getHousehold returns the session user's household with members + invite code.
func (s *Server) getHousehold(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	h, err := s.st.GetHousehold(r.Context(), sess.UserID)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"household": h})
}

type renameReq struct {
	Name string `json:"name"`
}

// patchHousehold renames the household (any member may rename — local app).
func (s *Server) patchHousehold(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	hid, ok := s.mustHousehold(w, r, sess)
	if !ok {
		return
	}
	var in renameReq
	if !decode(w, r, &in) {
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 60 {
		writeErr(w, http.StatusBadRequest, "validation", "name required (max 60 chars)")
		return
	}
	if s.storeErr(w, s.st.RenameHousehold(r.Context(), hid, name)) {
		return
	}
	h, err := s.st.GetHousehold(r.Context(), sess.UserID)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"household": h})
}

type joinReq struct {
	Code string `json:"code"`
}

// joinHousehold moves the user into the household behind the invite code.
// Personal data (preferences, swipes) travels along; shared data stays
// with the old household.
func (s *Server) joinHousehold(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	var in joinReq
	if !decode(w, r, &in) {
		return
	}
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	if code == "" || len(code) > 16 {
		writeErr(w, http.StatusBadRequest, "validation", "invite code required")
		return
	}
	h, err := s.st.JoinHousehold(r.Context(), sess.UserID, code)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"household": h})
}

// leaveHousehold moves the user into a fresh personal household.
func (s *Server) leaveHousehold(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	h, err := s.st.LeaveHousehold(r.Context(), sess.UserID)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"household": h})
}

// rotateInvite issues a new invite code; the old one stops working.
func (s *Server) rotateInvite(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	hid, ok := s.mustHousehold(w, r, sess)
	if !ok {
		return
	}
	if s.storeErr(w, s.st.RotateInvite(r.Context(), hid)) {
		return
	}
	h, err := s.st.GetHousehold(r.Context(), sess.UserID)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"household": h})
}
