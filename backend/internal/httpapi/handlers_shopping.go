package httpapi

import (
	"net/http"
	"unicode/utf8"

	"github.com/alexisvaleev/mealswipe/backend/internal/store"
)

// regenerateList rebuilds non-manual items from the bound menu,
// keeping manual items and checked state.
func (s *Server) regenerateList(w http.ResponseWriter, r *http.Request) {
	if !requireUUID(w, r, "id") {
		return
	}
	sess := sessionFrom(r)
	list, err := s.st.GetList(r.Context(), sess.UserID, r.PathValue("id"))
	if s.storeErr(w, err) {
		return
	}
	if list.MenuID == nil {
		writeErr(w, http.StatusBadRequest, "validation", "list is not bound to a menu")
		return
	}
	if s.storeErr(w, s.st.GenerateListItems(r.Context(), sess.UserID, list.ID)) {
		return
	}
	l2, err := s.st.GetList(r.Context(), sess.UserID, list.ID)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"list": l2})
}

type addListItemReq struct {
	Name     string   `json:"name"`
	Quantity *float64 `json:"quantity"`
	Unit     string   `json:"unit"`
	Category string   `json:"category"`
}

func (s *Server) addListItem(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	var in addListItemReq
	if !decode(w, r, &in) {
		return
	}
	list, err := s.st.GetList(r.Context(), sess.UserID, r.PathValue("id"))
	if s.storeErr(w, err) {
		return
	}
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 200 {
		writeErr(w, http.StatusBadRequest, "validation", "name required (max 200 chars)")
		return
	}
	if in.Category == "" {
		in.Category = "other"
	}
	if !validCategories[in.Category] {
		writeErr(w, http.StatusBadRequest, "validation", "unknown category")
		return
	}
	if in.Quantity != nil && *in.Quantity < 0 {
		writeErr(w, http.StatusBadRequest, "validation", "quantity must be >= 0")
		return
	}
	it, err := s.st.AddListItem(r.Context(), list.ID, in.Name, in.Quantity, in.Unit, in.Category)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": it})
}

type patchListItemReq struct {
	Checked  *bool    `json:"checked"`
	Quantity *float64 `json:"quantity"`
	Name     *string  `json:"name"`
}

func (s *Server) patchListItem(w http.ResponseWriter, r *http.Request) {
	if !requireUUID(w, r, "id", "item_id") {
		return
	}
	sess := sessionFrom(r)
	var in patchListItemReq
	if !decode(w, r, &in) {
		return
	}
	if in.Quantity != nil && *in.Quantity < 0 {
		writeErr(w, http.StatusBadRequest, "validation", "quantity must be >= 0")
		return
	}
	if in.Name != nil && (*in.Name == "" || utf8.RuneCountInString(*in.Name) > 200) {
		writeErr(w, http.StatusBadRequest, "validation", "bad name")
		return
	}
	it, err := s.st.PatchListItem(r.Context(), sess.UserID, r.PathValue("id"),
		r.PathValue("item_id"), store.ListItemPatch{
			Checked: in.Checked, Quantity: in.Quantity, Name: in.Name,
		})
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": it})
}

func (s *Server) deleteListItem(w http.ResponseWriter, r *http.Request) {
	if !requireUUID(w, r, "id", "item_id") {
		return
	}
	sess := sessionFrom(r)
	if s.storeErr(w, s.st.DeleteListItem(r.Context(), sess.UserID,
		r.PathValue("id"), r.PathValue("item_id"))) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
