# Security model

This document describes what MealSwipe does for security and what the operator
must still take care of. Vulnerability reporting: [SECURITY.md](../../SECURITY.md).

## Implemented controls

| Area | Control |
|---|---|
| Passwords | bcrypt (cost 12); plaintext never stored or logged |
| Sessions | opaque 256-bit tokens in HttpOnly `SameSite=Lax` cookie; only SHA-256 hash kept in DB; sliding expiry, server-side revocation on logout |
| CSRF | session-bound token required via `X-CSRF-Token` on every mutation + Origin-header check when present |
| SQL injection | all queries parameterized via pgx args; no string-built SQL |
| XSS | React escapes output by default; CSP + `X-Content-Type-Options`/`X-Frame-Options`/`Referrer-Policy` headers |
| Rate limiting | per-IP token bucket: strict on `/auth/*`, generous global cap |
| Input safety | 1 MiB JSON body cap; server-side validation of types/enums |
| AuthZ | every user-scoped query filters by session `user_id`; foreign IDs → 404 |
| Secrets | env-only; `.env` gitignored; no secrets in images or bundles |
| Errors | JSON envelope; internals never leaked to clients; slog JSON logs |
| Containers | backend runs as uid 10001, frontend nginx unprivileged; Postgres port unpublished |

## Operator responsibilities

- **HTTPS** — terminate TLS on your proxy and set `COOKIE_SECURE=true`.
  Over plain HTTP the session cookie is necessarily `Secure=false`.
- **`POSTGRES_PASSWORD`** — set a strong value in `.env`.
- Keep host/docker images updated (`dependabot` covers repo deps).
- Don't expose the Postgres port or the backend port publicly — only the
  frontend (or your TLS proxy) should be reachable.
- Review `.env` permissions (`chmod 600`) on multi-user hosts.

## Known limitations (by design, for MVP)

- Username+password accounts only — no email, no verification / password
  reset flow; a lost password means a new account (or manual DB fix).
- Household invite codes are the only join mechanism — anyone holding the
  code can join; rotate it if it leaks (Settings → Обновить код).
- Rate limiting is in-memory per backend instance (single instance is the
  supported topology anyway).
- Recipe nutrition/cost values are estimates — never presented as verified
  data (`≈` marker in UI, `nutrition_estimate` flag in API).
- Allergen filtering relies on curated `allergen_tags`; absence of a tag is
  **not** a safety guarantee — the UI always shows full ingredient lists.
