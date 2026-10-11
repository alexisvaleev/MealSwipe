// Package httpapi implements the /api/v1 JSON API (SPEC §5).
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/alexisvaleev/mealswipe/backend/internal/config"
	"github.com/alexisvaleev/mealswipe/backend/internal/store"
)

// Version is reported by /health.
const Version = "1.0.0"

// Server bundles dependencies shared by handlers.
type Server struct {
	cfg      config.Config
	st       *store.Store
	log      *slog.Logger
	global   *ipRate
	authRate *ipRate
}

// NewServer builds the API server.
func NewServer(cfg config.Config, st *store.Store, log *slog.Logger) *Server {
	return &Server{
		cfg:      cfg,
		st:       st,
		log:      log,
		global:   newIPRate(600, 60), // 600 req/min global (SPEC §11)
		authRate: newIPRate(20, 10),  // 20 req/min, burst 10 on /auth/*
	}
}

// Routes wires the full middleware chain + router.
func (s *Server) Routes() http.Handler {
	// Public endpoints.
	api := http.NewServeMux()
	api.HandleFunc("GET /api/v1/health", s.health)
	api.Handle("POST /api/v1/auth/register", s.authRate.wrap(http.HandlerFunc(s.register)))
	api.Handle("POST /api/v1/auth/login", s.authRate.wrap(http.HandlerFunc(s.login)))

	// Session-protected endpoints.
	au := http.NewServeMux()
	au.HandleFunc("POST /api/v1/auth/logout", s.logout)
	au.HandleFunc("GET /api/v1/auth/me", s.me)
	au.HandleFunc("GET /api/v1/preferences", s.getPrefs)
	au.HandleFunc("PUT /api/v1/preferences", s.putPrefs)
	au.HandleFunc("GET /api/v1/ingredients", s.listIngredients)
	au.HandleFunc("GET /api/v1/recipes", s.listRecipes)
	au.HandleFunc("GET /api/v1/recipes/{id}", s.getRecipe)
	au.HandleFunc("GET /api/v1/discover", s.discover)
	au.HandleFunc("PUT /api/v1/recipes/{id}/swipe", s.swipe)
	au.HandleFunc("DELETE /api/v1/recipes/{id}/swipe", s.unswipe)
	au.HandleFunc("GET /api/v1/likes", s.likes)
	au.HandleFunc("POST /api/v1/swipes/undo", s.undoSwipe)
	au.HandleFunc("PUT /api/v1/recipes/{id}/favorite", s.favorite)
	au.HandleFunc("DELETE /api/v1/recipes/{id}/favorite", s.unfavorite)
	au.HandleFunc("GET /api/v1/favorites", s.listFavorites)
	au.HandleFunc("GET /api/v1/menus/current", s.currentMenu)
	au.HandleFunc("POST /api/v1/menus/generate", s.generateMenu)
	au.HandleFunc("GET /api/v1/menus/{id}", s.getMenu)
	au.HandleFunc("POST /api/v1/menus/{id}/items", s.addMenuItem)
	au.HandleFunc("PATCH /api/v1/menus/{id}/items/{item_id}", s.patchMenuItem)
	au.HandleFunc("DELETE /api/v1/menus/{id}/items/{item_id}", s.deleteMenuItem)
	au.HandleFunc("POST /api/v1/menus/{id}/items/{item_id}/swap", s.swapMenuItem)
	au.HandleFunc("GET /api/v1/menus/{id}/shopping-list", s.menuShoppingList)
	au.HandleFunc("POST /api/v1/shopping-lists/{id}/regenerate", s.regenerateList)
	au.HandleFunc("POST /api/v1/shopping-lists/{id}/items", s.addListItem)
	au.HandleFunc("PATCH /api/v1/shopping-lists/{id}/items/{item_id}", s.patchListItem)
	au.HandleFunc("DELETE /api/v1/shopping-lists/{id}/items/{item_id}", s.deleteListItem)
	au.HandleFunc("GET /api/v1/pantry", s.pantryList)
	au.HandleFunc("PUT /api/v1/pantry/{ingredient_id}", s.pantryAdd)
	au.HandleFunc("DELETE /api/v1/pantry/{ingredient_id}", s.pantryRemove)
	au.HandleFunc("GET /api/v1/household", s.getHousehold)
	au.HandleFunc("PATCH /api/v1/household", s.patchHousehold)
	au.HandleFunc("POST /api/v1/household/join", s.joinHousehold)
	au.HandleFunc("POST /api/v1/household/leave", s.leaveHousehold)
	au.HandleFunc("POST /api/v1/household/invite/rotate", s.rotateInvite)
	api.Handle("/api/v1/", s.requireAuth(au))

	return securityHeaders(
		s.originCheck(
			s.global.wrap(
				bodyCap(api))))
}
