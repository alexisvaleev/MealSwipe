package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/alexisvaleev/mealswipe/backend/internal/config"
	"github.com/alexisvaleev/mealswipe/backend/internal/db"
	"github.com/alexisvaleev/mealswipe/backend/internal/httpapi"
	"github.com/alexisvaleev/mealswipe/backend/internal/seed"
	"github.com/alexisvaleev/mealswipe/backend/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

// All tests in this file require a real Postgres: set TEST_DATABASE_URL.
// They drop and recreate the public schema, so use a throwaway database.

const testSeedJSON = `{"recipes":[
 {"slug":"pasta-milk","title":"Паста с молоком","description":"t","instructions":["вари"],
  "prep_minutes":5,"cook_minutes":10,"servings":2,
  "meal_types":["lunch"],"category":"Паста","tags":[],"diet_flags":["vegetarian"],
  "cost_per_serving":50,"kcal_per_serving":400,
  "ingredients":[{"name":"молоко","quantity":200,"unit":"ml"},{"name":"мука","quantity":100,"unit":"g"}],
  "ingredient_meta":{"молоко":{"category":"dairy","unit_family":"volume","allergen_tags":["milk"]},
                     "мука":{"category":"grains","unit_family":"mass","allergen_tags":["gluten"]}}},
 {"slug":"soup-milk","title":"Суп с молоком","description":"t","instructions":["вари"],
  "prep_minutes":5,"cook_minutes":10,"servings":2,
  "meal_types":["lunch"],"category":"Суп","tags":[],"diet_flags":["vegetarian"],
  "cost_per_serving":60,"kcal_per_serving":300,
  "ingredients":[{"name":"молоко","quantity":500,"unit":"ml"},{"name":"соль","quantity":1,"unit":"pinch"}],
  "ingredient_meta":{"соль":{"category":"spices","unit_family":"mass","allergen_tags":[]}}},
 {"slug":"third-lunch","title":"Третий обед","description":"t","instructions":["вари"],
  "prep_minutes":5,"cook_minutes":10,"servings":2,
  "meal_types":["lunch"],"category":"Салат","tags":[],"diet_flags":["vegetarian"],
  "cost_per_serving":40,"kcal_per_serving":250,
  "ingredients":[{"name":"морковь","quantity":2,"unit":"pcs"}],
  "ingredient_meta":{"морковь":{"category":"produce","unit_family":"count","allergen_tags":[]}}}
]}`

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st := store.New(pool)
	cfg := config.Config{
		AppName: "Test", SessionTTL: time.Hour, LogLevel: slog.LevelError,
	}
	srv := httpapi.NewServer(cfg, st, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(func() { ts.Close(); pool.Close() })
	return ts, st
}

type client struct {
	t      *testing.T
	base   string
	cookie *http.Cookie
	csrf   string
	hc     *http.Client
}

func newClient(t *testing.T, base string) *client {
	return &client{t: t, base: base, hc: &http.Client{Timeout: 10 * time.Second}}
}

func (c *client) do(method, path string, body any, withAuth bool) (int, map[string]any) {
	c.t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal: %v", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, rdr)
	if err != nil {
		c.t.Fatalf("request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if withAuth && c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	if withAuth && c.csrf != "" {
		req.Header.Set("X-CSRF-Token", c.csrf)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		c.t.Fatalf("do %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	var out map[string]any
	data, _ := io.ReadAll(resp.Body)
	if len(data) > 0 {
		_ = json.Unmarshal(data, &out)
	}
	if c.cookie == nil {
		for _, ck := range resp.Cookies() {
			if ck.Name == "mealswipe_session" {
				cc := *ck
				c.cookie = &cc
			}
		}
	}
	return resp.StatusCode, out
}

func (c *client) register(username string) {
	c.t.Helper()
	code, out := c.do("POST", "/api/v1/auth/register",
		map[string]any{"username": username, "password": "password123", "display_name": "Tester"}, false)
	if code != 201 {
		c.t.Fatalf("register -> %d: %v", code, out)
	}
	csrf, _ := out["csrf"].(string)
	if csrf == "" {
		c.t.Fatal("no csrf in register response")
	}
	c.csrf = csrf
	if c.cookie == nil {
		c.t.Fatal("no session cookie set")
	}
}

func seedDB(t *testing.T, st *store.Store, raw string) {
	t.Helper()
	if _, err := seed.Seed(context.Background(), st.Pool(), []byte(raw)); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestAuthFlow(t *testing.T) {
	ts, _ := newTestServer(t)
	c := newClient(t, ts.URL)

	code, out := c.do("POST", "/api/v1/auth/register",
		map[string]any{"username": "alice_1", "password": "password123", "display_name": "A"}, false)
	if code != 201 {
		t.Fatalf("register -> %d: %v", code, out)
	}
	u, _ := out["user"].(map[string]any)
	if u["username"] != "alice_1" || u["has_preferences"] != false {
		t.Fatalf("bad user payload: %v", u)
	}
	c.csrf, _ = out["csrf"].(string)

	code, out = c.do("GET", "/api/v1/auth/me", nil, true)
	if code != 200 || out["user"] == nil {
		t.Fatalf("me -> %d: %v", code, out)
	}

	code, _ = c.do("POST", "/api/v1/auth/logout", nil, true)
	if code != 204 {
		t.Fatalf("logout -> %d", code)
	}
	code, _ = c.do("GET", "/api/v1/auth/me", nil, true)
	if code != 401 {
		t.Fatalf("me after logout -> %d, want 401", code)
	}

	// new session via login
	c.cookie = nil
	code, out = c.do("POST", "/api/v1/auth/login",
		map[string]any{"username": "alice_1", "password": "password123"}, false)
	if code != 200 || out["csrf"] == nil {
		t.Fatalf("login -> %d: %v", code, out)
	}
	code, _ = c.do("POST", "/api/v1/auth/login",
		map[string]any{"username": "alice_1", "password": "wrong-password"}, false)
	if code != 401 {
		t.Fatalf("bad login -> %d, want 401", code)
	}
}

func TestRegisterValidation(t *testing.T) {
	ts, _ := newTestServer(t)
	c := newClient(t, ts.URL)
	for i, body := range []map[string]any{
		{"username": "no spaces!", "password": "password123"},
		{"username": "ab", "password": "password123"}, // too short
		{"username": "valid_user", "password": "short"},
		{"username": "valid_user", "password": "password123", "display_name": string(make([]byte, 100))},
	} {
		if code, out := c.do("POST", "/api/v1/auth/register", body, false); code != 400 {
			t.Fatalf("case %d -> %d: %v", i, code, out)
		}
	}
}

func TestCSRFRequired(t *testing.T) {
	ts, _ := newTestServer(t)
	c := newClient(t, ts.URL)
	c.register("csrf_t")
	c.csrf = ""
	code, out := c.do("PUT", "/api/v1/preferences",
		map[string]any{"household_size": 2, "meals_per_day": 3,
			"meal_types": []string{"lunch"}, "diet_type": "any"}, true)
	if code != 403 {
		t.Fatalf("mutating without csrf -> %d: %v", code, out)
	}
}

func TestUnauthenticated(t *testing.T) {
	ts, _ := newTestServer(t)
	c := newClient(t, ts.URL)
	for _, p := range []string{"/api/v1/preferences", "/api/v1/recipes", "/api/v1/favorites", "/api/v1/menus/current"} {
		if code, _ := c.do("GET", p, nil, false); code != 401 {
			t.Fatalf("GET %s -> %d, want 401", p, code)
		}
	}
	code, _ := c.do("GET", "/api/v1/health", nil, false)
	if code != 200 {
		t.Fatalf("health -> %d", code)
	}
}

func TestMenuGenerateAndUserIsolation(t *testing.T) {
	ts, st := newTestServer(t)
	seedDB(t, st, testSeedJSON)

	a := newClient(t, ts.URL)
	a.register("a_t")
	b := newClient(t, ts.URL)
	b.register("b_t")

	code, out := a.do("PUT", "/api/v1/preferences",
		map[string]any{"household_size": 2, "meals_per_day": 1,
			"meal_types": []string{"lunch"}, "diet_type": "any",
			"allergens": []string{}, "excluded_ingredient_ids": []int64{}, "goal": ""}, true)
	if code != 200 {
		t.Fatalf("put prefs -> %d: %v", code, out)
	}

	code, out = a.do("POST", "/api/v1/menus/generate",
		map[string]any{"week_start": "2026-10-12"}, true) // a Monday
	if code != 200 {
		t.Fatalf("generate -> %d: %v", code, out)
	}
	menu, _ := out["menu"].(map[string]any)
	if menu == nil {
		t.Fatalf("no menu in response: %v", out)
	}
	items, _ := menu["items"].([]any)
	if len(items) != 3 { // 3 lunch recipes -> 3 filled slots
		t.Fatalf("expected 3 items, got %v", items)
	}
	unfilled, _ := out["unfilled"].([]any)
	if len(unfilled) != 4 {
		t.Fatalf("expected 4 unfilled, got %v", unfilled)
	}
	menuID, _ := menu["id"].(string)

	// User B must not see A's menu (user isolation).
	if code, _ := b.do("GET", "/api/v1/menus/"+menuID, nil, true); code != 404 {
		t.Fatalf("user B read A's menu -> %d, want 404", code)
	}
	if code, _ := b.do("GET", "/api/v1/menus/"+menuID+"/shopping-list", nil, true); code != 404 {
		t.Fatalf("user B read A's list -> %d, want 404", code)
	}
	if code, _ := b.do("DELETE", "/api/v1/menus/"+menuID+"/items/x", nil, true); code != 404 {
		t.Fatalf("user B delete A's item -> %d, want 404", code)
	}
}

func TestSwipeUndoAndFavorites(t *testing.T) {
	ts, st := newTestServer(t)
	seedDB(t, st, testSeedJSON)
	c := newClient(t, ts.URL)
	c.register("sw_t")

	code, out := c.do("GET", "/api/v1/recipes", nil, true)
	if code != 200 {
		t.Fatalf("recipes -> %d: %v", code, out)
	}
	recipes, _ := out["recipes"].([]any)
	if len(recipes) != 3 {
		t.Fatalf("expected 3 recipes, got %v", recipes)
	}
	rid, _ := recipes[0].(map[string]any)["id"].(string)

	code, out = c.do("PUT", "/api/v1/recipes/"+rid+"/swipe",
		map[string]any{"direction": "like"}, true)
	if code != 200 {
		t.Fatalf("swipe -> %d: %v", code, out)
	}
	code, out = c.do("PUT", "/api/v1/recipes/"+rid+"/favorite", nil, true)
	if code != 204 {
		t.Fatalf("favorite -> %d: %v", code, out)
	}
	code, out = c.do("GET", "/api/v1/favorites", nil, true)
	favs, _ := out["recipes"].([]any)
	if code != 200 || len(favs) != 1 {
		t.Fatalf("favorites -> %d: %v", code, out)
	}
	if favs[0].(map[string]any)["favorited_at"] == nil {
		t.Fatal("missing favorited_at")
	}

	code, out = c.do("POST", "/api/v1/swipes/undo", nil, true)
	if code != 200 || out["undone"] != true {
		t.Fatalf("undo -> %d: %v", code, out)
	}
	code, _ = c.do("POST", "/api/v1/swipes/undo", nil, true)
	if code != 404 {
		t.Fatalf("second undo -> %d, want 404", code)
	}
}

// TestLikesList: /likes shows personal likes; DELETE swipe removes an entry.
func TestLikesList(t *testing.T) {
	ts, st := newTestServer(t)
	seedDB(t, st, testSeedJSON)
	c := newClient(t, ts.URL)
	c.register("likes_t")

	code, out := c.do("GET", "/api/v1/recipes", nil, true)
	if code != 200 {
		t.Fatalf("recipes -> %d: %v", code, out)
	}
	recipes := out["recipes"].([]any)
	rid1 := recipes[0].(map[string]any)["id"].(string)
	rid2 := recipes[1].(map[string]any)["id"].(string)

	for _, rid := range []string{rid1, rid2} {
		if code, out := c.do("PUT", "/api/v1/recipes/"+rid+"/swipe",
			map[string]any{"direction": "like"}, true); code != 200 {
			t.Fatalf("swipe -> %d: %v", code, out)
		}
	}

	code, out = c.do("GET", "/api/v1/likes", nil, true)
	likes, _ := out["recipes"].([]any)
	if code != 200 || len(likes) != 2 {
		t.Fatalf("likes -> %d: %v", code, out)
	}

	// DELETE swipe removes it; idempotent (second delete still 204).
	if code, out := c.do("DELETE", "/api/v1/recipes/"+rid1+"/swipe", nil, true); code != 204 {
		t.Fatalf("unswipe -> %d: %v", code, out)
	}
	if code, _ := c.do("DELETE", "/api/v1/recipes/"+rid1+"/swipe", nil, true); code != 204 {
		t.Fatalf("unswipe again -> %d, want 204", code)
	}
	code, out = c.do("GET", "/api/v1/likes", nil, true)
	if likes, _ := out["recipes"].([]any); code != 200 || len(likes) != 1 {
		t.Fatalf("likes after unlike -> %d: %v", code, out)
	}
}

func TestShoppingGeneration(t *testing.T) {
	ts, st := newTestServer(t)
	seedDB(t, st, testSeedJSON)
	c := newClient(t, ts.URL)
	c.register("shop_t")

	code, out := c.do("PUT", "/api/v1/preferences",
		map[string]any{"household_size": 2, "meals_per_day": 1,
			"meal_types": []string{"lunch"}, "diet_type": "any",
			"allergens": []string{}, "excluded_ingredient_ids": []int64{}, "goal": ""}, true)
	if code != 200 {
		t.Fatalf("prefs -> %d: %v", code, out)
	}
	code, out = c.do("POST", "/api/v1/menus/generate",
		map[string]any{"week_start": "2026-10-12"}, true)
	if code != 200 {
		t.Fatalf("generate -> %d: %v", code, out)
	}
	menuID := out["menu"].(map[string]any)["id"].(string)

	code, out = c.do("GET", "/api/v1/menus/"+menuID+"/shopping-list", nil, true)
	if code != 200 {
		t.Fatalf("shopping-list -> %d: %v", code, out)
	}
	list, _ := out["list"].(map[string]any)
	litems, _ := list["items"].([]any)
	listID := list["id"].(string)

	// молоко 200+500=700ml merged, мука 100g, соль pinch, морковь 2pcs -> 4 lines
	if len(litems) != 4 {
		t.Fatalf("expected 4 list items, got %v", litems)
	}
	var milkQty float64
	var milkUnit string
	var flourID float64
	var milkID any
	for _, it := range litems {
		m := it.(map[string]any)
		switch m["name"] {
		case "молоко":
			milkQty, _ = m["quantity"].(float64)
			milkUnit, _ = m["unit"].(string)
			milkID = m["ingredient_id"]
		case "мука":
			flourID, _ = m["ingredient_id"].(float64)
		}
	}
	if milkQty != 700 || milkUnit != "ml" {
		t.Fatalf("milk = %v %s, want 700 ml", milkQty, milkUnit)
	}

	// "Already have it" flow: PUT pantry + DELETE line, then regenerate.
	if code, _ := c.do("PUT", "/api/v1/pantry/"+itoa(flourID), nil, true); code != 204 {
		t.Fatalf("pantry add -> %d", code)
	}
	var flourItemID string
	for _, it := range litems {
		m := it.(map[string]any)
		if m["name"] == "мука" {
			flourItemID = m["id"].(string)
		}
	}
	if code, _ := c.do("DELETE", "/api/v1/shopping-lists/"+listID+"/items/"+flourItemID, nil, true); code != 204 {
		t.Fatalf("delete flour line -> %d", code)
	}
	code, out = c.do("POST", "/api/v1/shopping-lists/"+listID+"/regenerate", nil, true)
	if code != 200 {
		t.Fatalf("regenerate -> %d: %v", code, out)
	}
	litems, _ = out["list"].(map[string]any)["items"].([]any)
	for _, it := range litems {
		if it.(map[string]any)["name"] == "мука" {
			t.Fatal("pantry ingredient regenerated")
		}
	}
	if len(litems) != 3 {
		t.Fatalf("expected 3 items after pantry skip, got %v", litems)
	}
	_ = milkID

	// manual item + check-off survives regeneration
	code, out = c.do("POST", "/api/v1/shopping-lists/"+listID+"/items",
		map[string]any{"name": "хлеб", "quantity": 1, "unit": "pcs", "category": "bakery"}, true)
	if code != 200 {
		t.Fatalf("add manual -> %d: %v", code, out)
	}
	manualID := out["item"].(map[string]any)["id"].(string)
	code, _ = c.do("PATCH", "/api/v1/shopping-lists/"+listID+"/items/"+manualID,
		map[string]any{"checked": true}, true)
	if code != 200 {
		t.Fatalf("patch manual -> %d", code)
	}
	code, out = c.do("POST", "/api/v1/shopping-lists/"+listID+"/regenerate", nil, true)
	if code != 200 {
		t.Fatalf("regen2 -> %d", code)
	}
	var manual map[string]any
	for _, it := range out["list"].(map[string]any)["items"].([]any) {
		m := it.(map[string]any)
		if m["id"] == manualID {
			manual = m
		}
	}
	if manual == nil || manual["checked"] != true || manual["is_manual"] != true {
		t.Fatalf("manual item lost/checked reset: %v", manual)
	}
}

func TestSwap(t *testing.T) {
	ts, st := newTestServer(t)
	seedDB(t, st, testSeedJSON)
	c := newClient(t, ts.URL)
	c.register("swap_t")
	c.do("PUT", "/api/v1/preferences",
		map[string]any{"household_size": 2, "meals_per_day": 1,
			"meal_types": []string{"lunch"}, "diet_type": "any",
			"allergens": []string{}, "excluded_ingredient_ids": []int64{}, "goal": ""}, true)
	code, out := c.do("POST", "/api/v1/menus/generate",
		map[string]any{"week_start": "2026-10-12"}, true)
	if code != 200 {
		t.Fatalf("generate -> %d: %v", code, out)
	}
	menu := out["menu"].(map[string]any)
	items := menu["items"].([]any)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %v", items)
	}
	menuID := menu["id"].(string)

	// Free one recipe by deleting its item; it becomes the swap candidate.
	victim := items[2].(map[string]any)
	code, _ = c.do("DELETE", "/api/v1/menus/"+menuID+"/items/"+victim["id"].(string), nil, true)
	if code != 204 {
		t.Fatalf("delete item -> %d", code)
	}

	item := items[0].(map[string]any)
	itemID := item["id"].(string)
	oldRecipe := item["recipe"].(map[string]any)["id"].(string)
	code, out = c.do("POST", "/api/v1/menus/"+menuID+"/items/"+itemID+"/swap", nil, true)
	if code != 200 {
		t.Fatalf("swap -> %d: %v", code, out)
	}
	newRecipe := out["item"].(map[string]any)["recipe"].(map[string]any)["id"].(string)
	if newRecipe == oldRecipe {
		t.Fatal("swap returned the same recipe")
	}
}

// TestHouseholdSharing: two users join one household — favorites/menus become
// shared, a member's dislike vetoes a recipe, and leaving isolates again.
func TestHouseholdSharing(t *testing.T) {
	ts, st := newTestServer(t)
	seedDB(t, st, testSeedJSON)

	a := newClient(t, ts.URL)
	a.register("owner_a")
	b := newClient(t, ts.URL)
	b.register("member_b")

	// A's personal household: 1 member + invite code.
	code, out := a.do("GET", "/api/v1/household", nil, true)
	if code != 200 {
		t.Fatalf("get household -> %d: %v", code, out)
	}
	h := out["household"].(map[string]any)
	invite := h["invite_code"].(string)
	if len(h["members"].([]any)) != 1 {
		t.Fatalf("expected 1 member, got %v", h["members"])
	}

	// B joins by code.
	code, out = b.do("POST", "/api/v1/household/join", map[string]any{"code": invite}, true)
	if code != 200 {
		t.Fatalf("join -> %d: %v", code, out)
	}
	if len(out["household"].(map[string]any)["members"].([]any)) != 2 {
		t.Fatalf("expected 2 members after join")
	}
	// Bad code -> 404.
	code, _ = b.do("POST", "/api/v1/household/join", map[string]any{"code": "ZZZZ-ZZZZ"}, true)
	if code != 404 {
		t.Fatalf("join bad code -> %d, want 404", code)
	}

	// Shared favorites: B's bookmark is visible to A.
	code, out = b.do("GET", "/api/v1/recipes", nil, true)
	if code != 200 {
		t.Fatalf("recipes -> %d", code)
	}
	recipes := out["recipes"].([]any)
	rid := recipes[0].(map[string]any)["id"].(string)
	otherRid := recipes[1].(map[string]any)["id"].(string)
	if code, _ := b.do("PUT", "/api/v1/recipes/"+rid+"/favorite", nil, true); code != 204 {
		t.Fatalf("favorite -> %d", code)
	}
	code, out = a.do("GET", "/api/v1/favorites", nil, true)
	if code != 200 || len(out["recipes"].([]any)) != 1 {
		t.Fatalf("shared favorites -> %d: %v", code, out)
	}

	// Dislike = veto: member B's dislike keeps the recipe out of the menu.
	if code, _ := b.do("PUT", "/api/v1/recipes/"+otherRid+"/swipe",
		map[string]any{"direction": "dislike"}, true); code != 200 {
		t.Fatalf("dislike -> %d", code)
	}
	code, out = a.do("PUT", "/api/v1/preferences",
		map[string]any{"household_size": 2, "meals_per_day": 1,
			"meal_types": []string{"lunch"}, "diet_type": "any",
			"allergens": []string{}, "excluded_ingredient_ids": []int64{}, "goal": ""}, true)
	if code != 200 {
		t.Fatalf("prefs -> %d", code)
	}
	code, out = a.do("POST", "/api/v1/menus/generate",
		map[string]any{"week_start": "2026-10-12"}, true)
	if code != 200 {
		t.Fatalf("generate -> %d: %v", code, out)
	}
	for _, it := range out["menu"].(map[string]any)["items"].([]any) {
		got := it.(map[string]any)["recipe"].(map[string]any)["id"].(string)
		if got == otherRid {
			t.Fatal("vetoed recipe landed in the shared menu")
		}
	}

	// Leaving isolates: B gets a fresh solo household and loses shared data.
	code, out = b.do("POST", "/api/v1/household/leave", nil, true)
	if code != 200 {
		t.Fatalf("leave -> %d: %v", code, out)
	}
	if len(out["household"].(map[string]any)["members"].([]any)) != 1 {
		t.Fatal("expected solo household after leave")
	}
	code, out = b.do("GET", "/api/v1/favorites", nil, true)
	if code != 200 || len(out["recipes"].([]any)) != 0 {
		t.Fatalf("favorites after leave -> %d: %v", code, out)
	}
}

func TestUsernameTaken(t *testing.T) {
	ts, _ := newTestServer(t)
	c := newClient(t, ts.URL)
	c.register("dup_user")
	code, out := c.do("POST", "/api/v1/auth/register",
		map[string]any{"username": "DUP_USER", "password": "password123"}, false)
	if code != 409 {
		t.Fatalf("duplicate username -> %d: %v", code, out)
	}
	if c2 := out["error"].(map[string]any)["code"]; c2 != "username_taken" {
		t.Fatalf("code = %v, want username_taken", c2)
	}
}

func itoa(f float64) string {
	return strconv.Itoa(int(f))
}
