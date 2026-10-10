# TODO — рабочий чеклист проекта

## Основание
- [x] SPEC.md (архитектура, API, схема БД, алгоритм)
- [ ] Миграция 0001_init (схема PostgreSQL)
- [ ] Seed-набор рецептов (~36, ru, ингредиенты + аллергены + картинки)

## Backend (Go)
- [ ] config, db pool, migrate runner, сидер
- [ ] auth (register/login/logout/me, сессии, CSRF, bcrypt)
- [ ] preferences, ingredients
- [ ] recipes + discover deck
- [ ] swipes + undo, favorites
- [ ] menus + planner + swap
- [ ] shopping lists + агрегация
- [ ] middleware: auth, csrf, origin, ratelimit, headers, body limit
- [ ] unit-тесты planner/shopping; integration-тесты под TEST_DATABASE_URL

## Frontend (React/TS)
- [ ] каркас Vite+Tailwind+i18n(ru)+api client
- [ ] auth pages + onboarding + settings
- [ ] swipe deck (framer-motion, кнопки, клавиатура, undo)
- [ ] favorites, recipe modal
- [ ] menu grid (swap, servings, lock, regenerate)
- [ ] shopping list (check, edit, manual add, regenerate)
- [ ] vitest, eslint, build

## Инфраструктура
- [ ] backend/Dockerfile, frontend/Dockerfile(+nginx), compose.yaml, compose.dev.yaml
- [ ] .env.example, .gitignore, .dockerignore, Makefile
- [ ] scripts/backup.sh, restore.sh; deployments/ proxy примеры

## Проверки
- [ ] go build/test/vet, npm build/test
- [ ] docker compose up end-to-end smoke
- [ ] e2e (playwright, опционально)
- [ ] скриншоты для README

## Open-source
- [ ] README (EN) + docs/en/* + docs/ru/*
- [ ] LICENSE (MIT — подтверждено), CONTRIBUTING, CODE_OF_CONDUCT, SECURITY.md, CHANGELOG
- [ ] .github: workflows, templates, dependabot
- [ ] security audit, публикация alexisvaleev/MealSwipe (public)
