# Changelog

All notable changes to this project will be documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
the project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Shared households («наша кухня»): members join via invite code and share
  favorites, weekly menu, shopping lists and pantry; preferences and swipes
  stay personal. Menu generation merges all members' hard constraints
  (allergens/exclusions ∪, strictest diet, tightest budget/time) and treats
  any member's dislike as a veto.
- `GET/PATCH /household`, `POST /household/join|leave|invite/rotate` endpoints
  and a settings UI section (members, invite code copy/rotate, join, leave).

### Changed

- Accounts now use **username+password** instead of email+password — the app
  is self-hosted/local, no email needed. Migration `0002` renames the column
  in place; existing usernames keep working.

### Added (initial release)

- Initial public release of MealSwipe.
- Swipe-based recipe discovery with keyboard, pointer and button controls.
- Username/password accounts with session-cookie auth and CSRF protection.
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
