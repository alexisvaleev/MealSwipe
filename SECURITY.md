# Security Policy

## Supported versions

MealSwipe is pre-1.0 and maintained on a best-effort basis. Only the latest
release / `main` branch receives security fixes.

| Version | Supported |
|---|---|
| latest release / main | ✅ |
| older releases | ❌ |

## Reporting a vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**
Publicly disclosing a flaw before a fix exists puts every self-hosted
instance at risk.

Instead, report privately via one of these channels:

- **GitHub private vulnerability reporting** (preferred): repository page →
  *Security* tab → *Report a vulnerability* (requires the maintainer to have
  enabled Private Vulnerability Reporting — check the Security tab first).
- If no private channel is available yet, open a minimal issue asking the
  maintainer for a secure contact — **without technical details**.

Please include: affected version/commit, impact, reproduction steps, and any
suggested mitigation. Reports are reviewed as soon as reasonably possible;
because this is a volunteer-run project, no response-time SLA is promised.

## What happens next

1. The report is triaged and the impact confirmed.
2. A fix is prepared and released.
3. Credit is given to the reporter in release notes (unless you prefer
   anonymity).

## Security model notes (self-hosted context)

- Passwords are stored as bcrypt hashes; sessions are opaque tokens stored
  hashed (SHA-256) server-side with CSRF protection.
- The frontend session cookie is `HttpOnly`, `SameSite=Lax`; enable
  `COOKIE_SECURE=true` when serving over HTTPS.
- Auth endpoints are rate-limited per IP; all requests are size-capped.
- PostgreSQL is not published on the host by the default compose file — keep
  it that way.
- Secrets belong in `.env` (gitignored), never in the repository.
