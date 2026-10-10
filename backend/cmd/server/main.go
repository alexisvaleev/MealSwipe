// Command server is the MealSwipe backend binary.
// Subcommands (os.Args[1]): serve (default), migrate, seed, import-mealdb [N].
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/alexisvaleev/mealswipe/backend/data"
	"github.com/alexisvaleev/mealswipe/backend/internal/config"
	"github.com/alexisvaleev/mealswipe/backend/internal/db"
	"github.com/alexisvaleev/mealswipe/backend/internal/httpapi"
	"github.com/alexisvaleev/mealswipe/backend/internal/mealdb"
	"github.com/alexisvaleev/mealswipe/backend/internal/seed"
	"github.com/alexisvaleev/mealswipe/backend/internal/store"
)

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	cmd := "serve"
	if len(args) > 1 {
		cmd = args[1]
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		return 2
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	ctx := context.Background()

	switch cmd {
	case "serve":
		return serve(ctx, cfg, log)
	case "migrate":
		pool, err := db.Connect(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Error("connect", "err", err)
			return 1
		}
		defer pool.Close()
		if err := db.Migrate(ctx, pool); err != nil {
			log.Error("migrate", "err", err)
			return 1
		}
		log.Info("migrations applied")
		return 0
	case "seed":
		pool, err := db.Connect(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Error("connect", "err", err)
			return 1
		}
		defer pool.Close()
		n, err := seed.Seed(ctx, pool, data.RecipesJSON)
		if err != nil {
			log.Error("seed", "err", err)
			return 1
		}
		log.Info("seeded", "recipes", n)
		return 0
	case "import-mealdb":
		n := 10
		if len(args) > 2 {
			if v, err := strconv.Atoi(args[2]); err == nil && v > 0 {
				n = v
			}
		}
		pool, err := db.Connect(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Error("connect", "err", err)
			return 1
		}
		defer pool.Close()
		done, err := mealdb.Import(ctx, pool, cfg.MealDBAPIKey, n, log)
		if err != nil {
			log.Error("import-mealdb", "err", err, "imported", done)
			return 1
		}
		log.Info("import-mealdb done", "imported", done)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "usage: %s [serve|migrate|seed|import-mealdb N]\n", args[0])
		return 2
	}
}

func serve(ctx context.Context, cfg config.Config, log *slog.Logger) int {
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("connect", "err", err)
		return 1
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Error("migrate", "err", err)
		return 1
	}
	st := store.New(pool)
	if cfg.SeedOnStart {
		if n, err := st.RecipeCount(ctx); err == nil && n == 0 {
			if seeded, err := seed.Seed(ctx, pool, data.RecipesJSON); err != nil {
				log.Error("seed", "err", err)
			} else {
				log.Info("seeded", "recipes", seeded)
			}
		}
	}
	if err := st.DeleteExpiredSessions(ctx); err != nil {
		log.Warn("session cleanup", "err", err)
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewServer(cfg, st, log).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
	}

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		// Hourly housekeeping of expired sessions (SPEC §11).
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-sigCtx.Done():
				return
			case <-t.C:
				_ = st.DeleteExpiredSessions(ctx)
			}
		}
	}()

	go func() {
		log.Info("listening", "addr", cfg.HTTPAddr, "app", cfg.AppName)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen", "err", err)
			stop()
		}
	}()

	<-sigCtx.Done()
	log.Info("shutting down")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error("shutdown", "err", err)
		return 1
	}
	return 0
}
