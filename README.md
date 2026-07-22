# Table Tennis Management System

A comprehensive backend and web application for managing table tennis events, players, matches, and real-time Elo rating tracking.

This project goes beyond a simple internal tool, offering a WTT-branded dark-mode UI for both admins and public viewers, along with a robust **Domain-Driven Design (DDD)** Go backend.

## Features

- **Multi-Format Events**: Supports `singles`, `doubles`, `mixed_doubles`, and `teams` tournaments.
- **Flexible Bracket Formats**: `groups_elimination`, `round_robin`, and `elimination` formats with per-division configuration.
- **Dynamic Elo Rating System**: Independent `Singles` and `Doubles` Elo ratings per player. Ratings are automatically calculated on match completion (using team averages for doubles).
- **Knockout Bracket Engine**: ITTF-compliant seeding that separates same-group players. Supports both automatic and admin-controlled drag-and-drop bracket seeding.
- **Team Match Support**: Full team competition format with sub-match aggregation (Olympic/Standard formats) and automatic parent-match winner resolution.
- **Real-time Score Entry**: Public QR-code-accessible score pages for each table. Supports automatic `window.close()` after match finalization.
- **Voice Announcements**: Automatic Latin American Spanish voice announcements when a match transitions to `in_progress`.
- **Live TV Dashboard**: A public-facing real-time Kanban board showing scheduled, in-progress, and finished matches — designed for venue display screens.
- **Internationalization (i18n)**: Fully translated interfaces for international events.
- **Event Table Management**: Enforces strict exclusivity for `in_progress` matches on assigned tables with real-time UI status updates.
- **Admin Dashboard**: Secure internal hub for managing the entire ecosystem (Players, Events, Score-keeping, Divisions).
- **Public Rankings**: Auto-updating global leaderboard separating singles and doubles by Elo points.
- **PDF/CSV Export**: Generate event summary reports as PDF or CSV.
- **Tournament Metrics**: Persistent `JSONB` analytics including Elo performance, division insights, and match-level statistics.

## Architecture

This project follows **Domain-Driven Design (DDD)** and **Clean Architecture** principles:

```
internal/
├── domain/          # Core business entities, value objects, and repository interfaces
│   ├── event/       # Tournament & Match aggregate roots, domain commands
│   ├── player/      # Player entity with Elo rating logic
│   ├── bracket/     # Bracket generation domain service
│   └── division/    # Division seeding and Elo-band logic
│
├── application/     # Use Cases — one file per use case
│   ├── event/       # CreateEvent, UpdateEvent, StartKnockout, GetDetailView, etc.
│   └── match/       # StartMatch, UpdateScore, FinishMatch, TeamMatchOrchestrator
│
├── infrastructure/  # External adapters: DB, PDF, QR, Notifications
│   └── persistence/bun/  # bun ORM repositories implementing domain interfaces
│
└── interfaces/
    └── http/handler/     # Thin HTTP handlers — parse request, call use case, render view
```

### Key Design Decisions

- **Thin Controllers**: HTTP handlers only parse HTTP requests and render views. All orchestration lives in Application Use Cases.
- **Command Objects**: `CreateEventCommand`, `UpdateEventCommand`, `FinishMatchCommand` — rich command structs replace 20+ parameter method signatures.
- **Infrastructure Segregation**: Bracket advancement, team sub-match aggregation, and Elo calculations happen in the infrastructure transaction layer (`MatchRepository.FinishMatch`) or domain services — never in HTTP handlers.
- **Domain Repository Interfaces**: `event.Repository`, `event.MatchRepository` defined in the domain layer; implemented in the `bun` infrastructure package.

## Technologies Used

- **Go (Golang)**: Core backend (DDD / Clean Architecture).
- **Fiber v2**: High-performance HTTP web framework.
- **PostgreSQL + Bun ORM**: Production-grade SQL with a fast Go ORM.
- **HTMX**: SPA-like partial HTML updates without a JavaScript framework.
- **Vanilla CSS + Go Templates**: Server-side rendered UI with micro-animations.

---

## Application Routes

### Public Views
| Method | Route | Description |
| :--- | :--- | :--- |
| `GET` | `/rankings/singles` | Global singles rankings, ordered by Singles Elo. |
| `GET` | `/rankings/doubles` | Global doubles rankings, ordered by Doubles Elo. |
| `GET` | `/events/:id` | Public event detail with live bracket & match list. |
| `GET` | `/events/:id/tv` | Live TV dashboard (Kanban board for venue screens). |
| `GET` | `/score/:pin` | Public score entry page for a match table. |

### Admin Dashboard
*All `/admin` endpoints are protected by session authentication.*
| Method | Route | Description |
| :--- | :--- | :--- |
| `GET` | `/admin/login` | Renders the login portal. |
| `POST`| `/admin/login` | Authenticates and provisions a secure session. |
| `GET` | `/admin/events` | List of all events + tournament creation form. |
| `GET` | `/admin/events/:id` | Full event detail, bracket editor, participant management. |
| `GET` | `/admin/events/:id/board` | Admin Kanban board for live match management. |
| `GET` | `/admin/players` | All registered athletes + add player form. |
| `GET` | `/admin/divisions` | Division configuration. |

### Key API Endpoints
| Method | Route | Description |
| :--- | :--- | :--- |
| `POST` | `/events` | Create a tournament. |
| `PUT` | `/events/:id` | Update a tournament (participants, rules, format). |
| `POST` | `/events/:id/start-knockout` | Trigger knockout stage generation. |
| `POST` | `/matches/create` | Draft a match assigning players to Team A & Team B. |
| `POST` | `/matches/finish` | Conclude a match. Triggers Elo updates, bracket advancement, and team-match aggregation atomically. |
| `POST` | `/matches/:id/score` | Submit set scores. Auto-resolves winner on completion. |

## Quick Start

1. **Verify Dependencies**: Go 1.21+ and PostgreSQL.
2. **Run the Server**:
   ```bash
   go run ./cmd/server
   ```
   The server auto-runs migrations on startup.
3. **Access the App**: Open `http://localhost:8080/admin` in your browser.

## Testing

```bash
go test ./...
go vet ./...
```

### Coverage gate

CI (`.github/workflows/ci.yml`) requires **≥ 90% statement coverage over `./internal/...`** before
anything ships. Reproduce the gate locally:

```bash
go test ./internal/... -covermode=atomic -coverprofile=coverage.out
go tool cover -func=coverage.out | tail -1     # total must be ≥ 90.0%
go tool cover -html=coverage.out               # find the gaps
```

`cmd/server` (wiring, `main.go`) and the one-off `cmd/migrate_*.go` scripts are excluded from the
threshold on purpose — they're composition roots, not logic.

## Deployment

Render deploys from the **`production`** branch, never from `master`. CI fast-forwards `production`
to a `master` commit only after the coverage gate passes, so a coverage regression can't reach the
service. This needs no Render deploy hook or API key — promotion is an ordinary `git push` from the
workflow using the built-in `GITHUB_TOKEN`.

```
push to master ──► test job (build + vet + coverage ≥ 90%)
                        │ pass                    │ fail
                        ▼                         ▼
              fast-forward production      nothing promoted,
                        │                  production untouched
                        ▼
              Render auto-deploys
```

One-time setup:

1. Create the branch from a known-good commit: `git push origin master:production`.
2. Render dashboard → service → *Settings* → set the tracked **Branch** to `production`
   (the `branch:` key in `render.yaml` only applies to Blueprint-managed services).
3. Leave **Auto-Deploy** on — it's now watching a branch only CI can advance.

Never push to `production` by hand. If it diverges from `master` the promotion step fails loudly;
reconcile with `git push origin master:production --force-with-lease`.

To change the bar, edit `COVERAGE_THRESHOLD` at the top of the workflow.
