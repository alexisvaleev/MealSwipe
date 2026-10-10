# Roadmap

Status: **MVP** — core loop works end to end. Rough plan for what comes next;
order is indicative, not a commitment. Ideas welcome via issues.

## Near term

- [ ] E-mail verification + password reset flow
- [ ] English UI (i18n infrastructure already in place — `src/i18n/`)
- [ ] "Reroll dislikes" flow on the swipe deck
- [ ] Recipe creation/editing by users
- [ ] More seed recipes; review of allergen coverage

## Mid term

- [ ] Real nutrition data from a verified source (e.g. USDA FoodData Central)
- [ ] Price estimation from regional store data (opt-in)
- [ ] Shareable menus / household collaboration
- [ ] PWA (offline shell, installable)
- [ ] Import from more sources (structured recipe sites via schema.org JSON-LD)

## Later / exploratory

- [ ] Pantry inventory tracking
- [ ] Leftover-aware planning ("use up X before it expires")
- [ ] Native mobile apps
- [ ] Optional AI-assisted suggestions (explicitly opt-in; the core planner
      stays deterministic by design)
