# World Table Tennis (WTT) Management System

A comprehensive backend and web application for managing table tennis tournaments, players, matches, and real-time Elo rating tracking.

This project goes beyond a simple internal tool, offering an immersive, WTT-branded dark-mode UI for both admins and public viewers, along with a robust domain-driven Go backend.

## Features

- **Multi-Format Tournaments**: Supports creating and managing `singles`, `doubles`, and `teams` events.
- **Dynamic Elo Rating System**: Players have independent and tracked `Singles` and `Doubles` Elo ratings. The system automatically calculates and records rating changes when matches are finished (using paired team averages for doubles).
- **Premium User Interface**: Built with Tailwind CSS, utilizing glassmorphism, WTT aesthetic (red, black, and gold theme), and seamless micro-interactions without page reloads using **HTMX**.
- **Admin Dashboard**: A secure internal hub for managing the entire ecosystem (Players, Events, Scorekeeping).
- **Public Rankings**: A stunning, auto-updating global leaderboard separating men's/women's singles and doubles players by their Elo points natively styled like official broadcast templates.

## Technologies Used

- **Go (Golang)**: Core backend processing and structure (Domain-Driven Design/Clean Architecture).
- **Fiber v2**: High-performance HTTP web framework.
- **SQLite + Bun ORM**: Lightweight SQL engine coupled with a fast Go ORM for persistence logic.
- **HTMX**: For SPA-like dynamic HTML form submissions and table partial updates.
- **Tailwind CSS**: Utility-first CSS framework for rapid, premium styling.

---

## Application Routes

### Public Views
| Method | Route | Description |
| :--- | :--- | :--- |
| `GET` | `/rankings/singles` | Views the global WTT singles rankings, ordered by Singles Elo. |
| `GET` | `/rankings/doubles` | Views the global WTT doubles rankings, ordered by Doubles Elo. |
| `GET` | `/leaderboard` | *Legacy Redirect* -> Automatically routes to `/rankings/singles`. |

### Admin Dashboard (HTML)
*Note: All `/admin` endpoints are protected by Session Authentication. Default credentials are `admin/password`.*
| Method | Route | Description |
| :--- | :--- | :--- |
| `GET` | `/admin/login` | Renders the login portal. |
| `POST`| `/admin/login` | Authenticates and provisions a secure session. |
| `POST`| `/admin/logout` | Revokes the current session and redirects to login. |
| `GET` | `/admin` | Root admin portal hub and navigation. |
| `GET` | `/admin/players` | Table of all registered athletes, along with a form to add a new player. |
| `GET` | `/admin/tournaments` | List of all system tournaments along with an event creation form. |
| `GET` | `/admin/matches` | Scorekeeping panel to generate multi-format matches, assign teams, and record winners. |

### API / form-action Endpoints (Used via HTMX)
| Method | Route | Payload Type | Description |
| :--- | :--- | :--- | :--- |
| `POST` | `/players` | Form/JSON | Registers a new athlete (`firstName`, `lastName`, `birthdate`, `gender`, `country`). Returns an HTML table row. |
| `POST` | `/tournaments` | Form/JSON | Creates an event (`name`, `type`, `startDate`, `endDate`). Returns an HTML table row. |
| `POST` | `/matches/create` | Form/JSON | Drafts an active match assigning players to `Team A` & `Team B`. Supports array input for doubles. Returns an HTML table row. |
| `POST` | `/matches/finish` | Form/JSON | Concludes a match via `winnerTeam` selection. Triggers automatic Elo calculations and persists. |

## Quick Start

1. **Verify Dependencies**: Make sure Go (1.20+) and SQLite3 are installed on your machine.
2. **Apply DB Migrations**:
   Run the SQL scripts to build the WTT-flavored schema.
   ```bash
   rm -f table_tennis.db
   sqlite3 table_tennis.db < cmd/migrations/001_initial.sql
   sqlite3 table_tennis.db < cmd/migrations/002_wtt_features.sql
   ```
3. **Run the Server**:
   ```bash
   go run ./cmd/server
   ```
4. Access the App: 
   Open `http://localhost:8080/admin` in your browser.
