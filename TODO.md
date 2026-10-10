# TODO — рабочий чеклист проекта

## Основание
- [x] SPEC.md (архитектура, API, схема БД, алгоритм)
- [x] Миграция 0001_init (схема PostgreSQL)
- [x] Seed-набор рецептов (37, ru, ингредиенты + аллергены + картинки)

## Backend (Go)
- [x] config, db pool, migrate runner, сидер
- [x] auth (register/login/logout/me, сессии, CSRF, bcrypt)
- [x] preferences, ingredients
- [x] recipes + discover deck
- [x] swipes + undo, favorites
- [x] menus + planner + swap
- [x] shopping lists + агрегация
- [x] middleware: auth, csrf, origin, ratelimit, headers, body limit
- [x] unit-тесты planner/shopping; integration-тесты под TEST_DATABASE_URL

## Frontend (React/TS)
- [x] каркас Vite+Tailwind+i18n(ru)+api client
- [x] auth pages + onboarding + settings
- [x] swipe deck (framer-motion, кнопки, клавиатура, undo)
- [x] favorites, recipe modal
- [x] menu grid (swap, servings, lock, regenerate)
- [x] shopping list (check, edit, manual add, regenerate)
- [x] vitest (39 тестов), eslint, build

## Инфраструктура
- [x] backend/Dockerfile, frontend/Dockerfile(+nginx), compose.yaml, compose.dev.yaml
- [x] .env.example, .gitignore, .dockerignore, Makefile
- [x] scripts/backup.sh, restore.sh; deployments/ proxy примеры

## Проверки
- [x] go build/test/vet, npm typecheck/lint/test/build
- [x] docker compose up end-to-end smoke (register→prefs→swipe→menu→shopping, аллерген-фильтр)
- [x] e2e (playwright, системный chromium)
- [x] скриншоты для README (docs/screenshots/)

## Open-source
- [x] README (EN) + docs/en/* + docs/ru/*
- [x] LICENSE (MIT), CONTRIBUTING, CODE_OF_CONDUCT, SECURITY.md, CHANGELOG
- [x] .github: workflows, templates, dependabot
- [x] security audit, публикация alexisvaleev/MealSwipe (public) — CI на push зелёный
