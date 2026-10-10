# Changelog

All notable changes to this project will be documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
the project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Initial public release of MealSwipe.
- Swipe-based recipe discovery with keyboard, pointer and button controls.
- Email/password accounts with session-cookie auth and CSRF protection.
- Onboarding preferences: household size, meals per day, budget, max cook time,
  diet type, allergens, excluded ingredients.
- Deterministic weekly menu generator honoring hard dietary constraints with a
  documented scoring model.
- Favorites collection with search, filtering and sorting.
- Weekly menu grid: per-slot swap, servings, lock, regenerate.
- Shopping list: ingredient aggregation with unit normalisation, categories,
  check-off, manual items, pantry ("already have it") exclusions.
- Curated seed recipe dataset (Russian) + optional TheMealDB importer.
- Docker Compose stack (PostgreSQL + Go backend + nginx frontend) with health
  checks and a dev-only compose for local development.
- Backup/restore scripts for PostgreSQL.
