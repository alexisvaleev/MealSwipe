# Contributing to MealSwipe

Thanks for your interest! MealSwipe is a small self-hosted project — contributions
are welcome and the process is intentionally lightweight.

## Ways to contribute

- **Bug reports** — open an issue using the *Bug Report* template. Include steps
  to reproduce, expected/actual behavior, and logs **with secrets removed**.
- **Feature requests** — open an issue using the *Feature Request* template.
- **Code** — fork, branch, PR (details below).
- **Docs & translations** — fixes and improvements are always useful.

## Development setup

Prerequisites: Go ≥ 1.27, Node ≥ 22, Docker + Compose, make (optional).

```bash
git clone https://github.com/alexisvaleev/MealSwipe.git && cd MealSwipe
cp .env.example .env                 # fill POSTGRES_PASSWORD
docker compose -f compose.dev.yaml up -d          # postgres only
cd backend && go run ./cmd/server serve &          # migrations+seed run on start
cd frontend && npm install && npm run dev          # http://localhost:5173
```

Details: [docs/en/DEVELOPMENT.md](docs/en/DEVELOPMENT.md).

## Workflow

1. Fork the repo and create a branch: `git checkout -b feat/my-change`
   (prefixes: `feat/`, `fix/`, `docs/`, `chore/`).
2. Make your change. Keep PRs focused — one concern per PR.
3. Run checks locally (same as CI):

   ```bash
   cd backend && gofmt -l . && go vet ./... && go test ./...
   cd frontend && npm run typecheck && npm run lint && npm test && npm run build
   ```

4. Commit with a clear message describing *why* (conventional-commit style like
   `feat: add pantry view` is appreciated but not enforced).
5. Push and open a Pull Request — fill in the template (summary, test plan,
   API/DB/config/breaking-change boxes).
6. A maintainer reviews; address feedback in follow-up commits.

## Code expectations

- **Backend**: idiomatic Go, stdlib first. New endpoints need a handler test or
  an integration test where practical. Planner/shopping logic stays pure and
  unit-tested. Run `gofmt`.
- **Frontend**: TypeScript strict, no `any` leaks, Russian UI strings via
  `src/i18n/ru.ts` (not hardcoded), accessible basics.
- **Database**: schema changes go in a new `backend/migrations/NNNN_name.up.sql`
  + matching `.down.sql`. Never edit applied migrations.
- **Docs**: update README/docs when you change behavior, config, or API.
- **Breaking changes**: call them out explicitly in the PR template.

## Reporting security issues

Do **not** open public issues for vulnerabilities — see
[SECURITY.md](SECURITY.md).
