# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go run ./cmd/server                 # run the dev server (does NOT auto-run migrations, see below)
go build ./...                      # build everything
go vet ./...                        # vet
gofmt -l .                          # list unformatted files (CI fails if non-empty)
go test ./...                       # run all tests
go test ./internal/event/... -run TestName -v   # run a single test
```

### Coverage gate (CI-enforced)

CI requires **≥ 90% statement coverage over `./internal/...`**. Reproduce locally:

```bash
go test ./internal/... -covermode=atomic -coverprofile=coverage.out
go tool cover -func=coverage.out | tail -1     # total must be ≥ 90.0%
go tool cover -html=coverage.out               # find the gaps
```

`cmd/server` and the one-off `cmd/migrate_*.go` scripts are excluded on purpose — composition roots, not logic. New code under `internal/` needs tests to avoid dropping the total below threshold.

### Deployment

Render deploys from the `production` branch, never `master`. CI fast-forwards `production` to a `master` commit only after the coverage gate passes — this is a plain `git push` using `GITHUB_TOKEN`, no Render deploy hook/API key involved. **Never push to `production` by hand.** If it diverges, reconcile with `git push origin master:production --force-with-lease`.

## Architecture

DDD / Clean Architecture, organized in the standard four layers under `internal/`:

```
internal/domain/          # entities, value objects, repository interfaces (no framework deps)
internal/application/     # use cases, one file per use case, orchestration only
internal/infrastructure/  # DB (bun ORM), PDF, QR, push notifications — implements domain interfaces
internal/interfaces/http/ # Fiber handlers (thin: parse request → call use case → render view) + templates
```

Domain repository interfaces (e.g. `event.Repository`, `event.MatchRepository`) live in `internal/domain/*`; their bun-backed implementations live in `internal/infrastructure/persistence/bun/`.

### Tournament (parent) vs Event (child)

- `internal/domain/tournament/` (package `tournament`, table `tournaments`) is the **parent umbrella competition** (e.g. "Ranking Nacional II") — holds `Events []*event.Event`.
- `internal/domain/event/` (package `event`, table `events`) is the **child per-category sub-competition** (e.g. "Men's Singles", "Doubles Mixed") — carries `Format`, `GroupPassCount`, `StageRules`, `SkipElo`, etc., and links back to its parent via `Event.TournamentID *string`.

This used to be inverted (a migration years ago swapped the two concepts, leaving `TournamentID`/`EventID` scrambled across ~35 files) — it was corrected in full: struct/field names, repository method names (`GetOccupiedTablesByEvent`/`GetOccupiedTablesByTournament`, `GetByTournamentID`, `GetTournamentNumTables`, `DeleteByEvent`), constructors (`tournament.NewTournament`, `event.NewEvent`), error vars, the `events.tournament_category` → `event_category` DB column (migration `045`), and user-facing i18n labels. If you see `TournamentID` on an `event.Event`/`EventModel`, that's correct (it's the parent link); everywhere else in the `event` package family (`Match`, `Group`, `Team`, `StageRule`, `DivisionRule`) `EventID` is the child event's own ID.

Import-alias convention: most files under `internal/application/event/` and `internal/interfaces/http/handler/` alias `internal/domain/event` as `tournamentDomain` (an established, if confusing, repo-wide convention — not a bug, don't "fix" it file-by-file) and `internal/domain/tournament` as `eventDomain`/`tournamentDomain` varies by file — check each file's own import block rather than assuming.

One parent → many children: `CreateEventUseCase.Execute` (`internal/application/tournament/tournament_crud.go`) creates one parent `Tournament` row, then for each of up to 7 `CategoryConfig` inputs (singles men/women, doubles men/women/mixed, teams men/women) creates a child `Event`, further split per-division if `DivisionConfigs` differ — all atomically in one request.

### Other notable pieces

- `internal/domain/bracket/` — bracket generation domain service; ITTF-compliant knockout seeding that separates same-group players.
- `internal/domain/division/` — division seeding and Elo-band logic; division-specific format/rule overrides live as `DivisionConfig` on the child event. Division-specific *stage rules* (best-of/points/margin per division) are edited only on the event-edit-form page (`admin/partials/event-edit-form.html`) — the separate `/admin/divisions` page manages the `Division` entity itself (name/Elo-range/color), not stage-rule overrides.
- `internal/domain/tournaments/` (plural — distinct from singular `tournament`) — a small pub/sub domain-event dispatcher (`InMemoryDispatcher`, `PlayerEnrolledEvent`), unrelated to the tournament/event entities above.
- Elo ratings are tracked independently for `Singles` and `Doubles` per player; team matches use team averages.
- Team match aggregation, bracket advancement, and Elo updates happen inside the infrastructure transaction layer (`MatchRepository.FinishMatch`) or domain services — never in HTTP handlers.
- Command objects (`CreateEventCommand`, `UpdateEventCommand`, `FinishMatchCommand`) replace long positional-parameter method signatures at use-case boundaries.
- Migrations are plain numbered SQL files in `cmd/migrations/`, each needing a matching `cmd/migrate_0NN.go` one-off script (`//go:build ignore`, excluded from the coverage gate). Locally these are **not applied automatically** — run via `go run cmd/migrate_0NN.go`. In CI they *are* applied automatically: after `master` passes the coverage gate and is promoted to `production`, the `migrate` job in `.github/workflows/ci.yml` diffs the pushed commit against its parent for newly **added** `cmd/migrate_*.go` files (not modified/re-pushed ones) and runs each against the `DATABASE_URL` secret — so a migration merged to `master` is live on `production`'s DB with no manual step. `cmd/migrations/postgresql_init.sql` is a manually-maintained full-schema baseline for fresh installs — keep it in sync with the latest migration when adding one.
