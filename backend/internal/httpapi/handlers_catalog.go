package httpapi

import (
	"net/http"
	"strconv"

	"github.com/alexisvaleev/mealswipe/backend/internal/store"
)

func (s *Server) listIngredients(w http.ResponseWriter, r *http.Request) {
	items, err := s.st.SearchIngredients(r.Context(), r.URL.Query().Get("q"), 200)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ingredients": items})
}

func (s *Server) listRecipes(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	hid, ok := s.mustHousehold(w, r, sess)
	if !ok {
		return
	}
	q := r.URL.Query()
	items, total, err := s.st.ListRecipes(r.Context(), sess.UserID, hid,
		q.Get("meal_type"), q.Get("q"),
		queryInt(r, "page", 1), queryInt(r, "page_size", 24))
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recipes": items, "total": total})
}

func (s *Server) getRecipe(w http.ResponseWriter, r *http.Request) {
	if !requireUUID(w, r, "id") {
		return
	}
	sess := sessionFrom(r)
	hid, ok := s.mustHousehold(w, r, sess)
	if !ok {
		return
	}
	d, err := s.st.GetRecipe(r.Context(), sess.UserID, hid, r.PathValue("id"))
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recipe": d})
}

// discover returns the swipe deck: unseen + hard-filtered, random order.
// The deck stays personal; the favorited flag is household-wide.
func (s *Server) discover(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	hid, ok := s.mustHousehold(w, r, sess)
	if !ok {
		return
	}
	p, _, err := s.st.GetPreferences(r.Context(), sess.UserID)
	if s.storeErr(w, err) {
		return
	}
	limit := queryInt(r, "limit", 20)
	items, err := s.st.DiscoverRecipes(r.Context(), sess.UserID, hid, store.DiscoverFilter{
		Allergens:      p.Allergens,
		ExcludedIDs:    p.ExcludedIngredientIDs,
		MealTypes:      p.MealTypes,
		DietType:       p.DietType,
		MaxCookMinutes: p.MaxCookMinutes,
		Limit:          limit,
	})
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recipes": items})
}

type swipeReq struct {
	Direction string `json:"direction"`
}

func (s *Server) swipe(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	var in swipeReq
	if !decode(w, r, &in) {
		return
	}
	if !requireUUID(w, r, "id") {
		return
	}
	if in.Direction != "like" && in.Direction != "dislike" {
		writeErr(w, http.StatusBadRequest, "validation", "direction must be like|dislike")
		return
	}
	sw, err := s.st.UpsertSwipe(r.Context(), sess.UserID, r.PathValue("id"), in.Direction)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"swipe": sw})
}

func (s *Server) undoSwipe(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	ok, err := s.st.DeleteLatestSwipe(r.Context(), sess.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "nothing to undo")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"undone": true})
}

func (s *Server) favorite(w http.ResponseWriter, r *http.Request) {
	if !requireUUID(w, r, "id") {
		return
	}
	sess := sessionFrom(r)
	hid, ok := s.mustHousehold(w, r, sess)
	if !ok {
		return
	}
	if s.storeErr(w, s.st.AddFavorite(r.Context(), hid, sess.UserID, r.PathValue("id"))) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) unfavorite(w http.ResponseWriter, r *http.Request) {
	if !requireUUID(w, r, "id") {
		return
	}
	sess := sessionFrom(r)
	hid, ok := s.mustHousehold(w, r, sess)
	if !ok {
		return
	}
	if s.storeErr(w, s.st.RemoveFavorite(r.Context(), hid, r.PathValue("id"))) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listFavorites(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	hid, ok := s.mustHousehold(w, r, sess)
	if !ok {
		return
	}
	q := r.URL.Query()
	items, err := s.st.ListFavorites(r.Context(), sess.UserID, hid,
		q.Get("category"), q.Get("q"), q.Get("sort"))
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recipes": items})
}

func (s *Server) pantryList(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	hid, ok := s.mustHousehold(w, r, sess)
	if !ok {
		return
	}
	ids, err := s.st.PantryIDs(r.Context(), hid)
	if s.storeErr(w, err) {
		return
	}
	flat := make([]int64, 0, len(ids))
	for id := range ids {
		flat = append(flat, id)
	}
	ings, err := s.st.IngredientsByIDs(r.Context(), flat)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ingredients": ings})
}

// pantryAdd marks an ingredient as owned. The "already have it" flow for a
// shopping line is: DELETE /shopping-lists/{id}/items/{item_id} then
// PUT /pantry/{ingredient_id} so future generations skip it.
func (s *Server) pantryAdd(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	hid, ok := s.mustHousehold(w, r, sess)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("ingredient_id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "validation", "bad ingredient id")
		return
	}
	if s.storeErr(w, s.st.PantryAdd(r.Context(), hid, id)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) pantryRemove(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	hid, ok := s.mustHousehold(w, r, sess)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("ingredient_id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "validation", "bad ingredient id")
		return
	}
	if s.storeErr(w, s.st.PantryRemove(r.Context(), hid, id)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
