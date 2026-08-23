# Tixigo

Tixigo is a full-stack cinema ticket-booking application built as an internship pre-interview project. It demonstrates authentication, role-based workflows, transactional seat reservations, real-time updates, and relational data modelling without depending on a real payment provider.

## Highlights

- Public movie discovery with city, date, language, genre, and price filters.
- Customer registration, email verification, password reset, JWT access tokens, and rotating refresh tokens in HttpOnly cookies.
- Visual seat selection with a maximum of 10 seats per booking.
- Atomic PostgreSQL seat holds that expire after 10 minutes.
- Per-screening WebSocket updates when seats are held, released, booked, or cancelled.
- Simulated checkout with QR tickets, ticket email, booking history, and cancellation until 24 hours before a show.
- Category-based waitlists with flexible seat fulfilment and one-hour offers.
- Organiser movie submissions, structured screening creation, occupancy statistics, and screening cancellation.
- Admin venue management, organiser promotion, and movie approval.
- Single-use ticket admission for organisers and admins.

## Technology

| Layer | Technology |
| --- | --- |
| Frontend | React, TypeScript, Vite, React Router, CSS |
| Backend | Go, Chi, pgx |
| Database | PostgreSQL with versioned SQL migrations |
| Realtime | WebSockets |
| Email | Resend in production, Mailpit locally |
| Media | Cloudinary |
| Testing | Go testing and Playwright |
| Deployment | Vercel, Render, Neon |

## Architecture

```text
React SPA
  ├── REST requests ───────────────> Go + Chi API
  └── per-screening WebSocket ─────> realtime hub
                                         │
Go API ── stores, transactions, RBAC ──> PostgreSQL
  ├── email notifications ─────────────> Resend / Mailpit
  └── poster uploads ──────────────────> Cloudinary
```

The monorepo is intentionally split by deployable application:

```text
frontend/       React application and Playwright tests
backend/        Go API, domain stores, commands, and migrations
infra/          backup, restore, and operations documentation
.github/        CI workflow
docker-compose.yml
render.yaml
```

## Important design decisions

### Preventing double booking

Seat selection is not trusted in memory. A hold runs inside a PostgreSQL transaction and updates only seats whose current state is `available`. Concurrent requests for the same seat cannot both succeed.

### Expiring reservations

A successful hold receives an expiry timestamp. A background worker releases expired seats, while API queries also treat expired holds as unavailable for checkout.

### Authentication

Short-lived access tokens are sent in the `Authorization` header. Refresh tokens are rotated and stored in HttpOnly cookies. The frontend coordinates simultaneous refresh attempts and retries a failed authenticated request once.

### Realtime updates

Clients subscribe only to the screening they are viewing. WebSocket events contain a small refresh notification; PostgreSQL remains the source of truth and the client reloads the authoritative seat map.

### Payment scope

Payment is deliberately simulated. The project focuses on booking correctness and clearly labels checkout as simulated rather than pretending to integrate a payment gateway.

## Local setup

Requirements: Docker, Go 1.26+, Node.js, and pnpm.

1. Create the backend configuration:

   ```bash
   cp backend/.env.example backend/.env
   ```

2. Replace both JWT secrets in `backend/.env` with different values of at least 32 characters.

3. Start PostgreSQL and Mailpit:

   ```bash
   docker compose up -d
   ```

4. Apply the database migrations:

   ```bash
   cd backend
   go run ./cmd/migrate
   ```

5. Start the API:

   ```bash
   go run ./cmd/server
   ```

6. In another terminal, start the frontend:

   ```bash
   cd frontend
   pnpm install
   pnpm dev
   ```

Open the application at `http://localhost:5173`. Development email is available in Mailpit at `http://localhost:8025`.

Cloudinary credentials are only required for poster uploads. When using Resend, set `RESEND_API_KEY` and a verified `EMAIL_FROM`; otherwise local email uses Mailpit.

## Preparing demo accounts

Register accounts through the application so the normal verification flow is exercised. Open each verification link from Mailpit.

Bootstrap the first administrator after registering it:

```bash
cd backend
go run ./cmd/promote-admin --email admin@example.com
```

Sign in as that administrator and promote another registered account to organiser from the admin dashboard. Do not commit real demo passwords to the repository.

## Suggested demonstration

The following walkthrough takes roughly five minutes:

1. Sign in as an organiser and create a movie with a venue, showtime, and category prices.
2. Sign in as an admin and approve the movie.
3. Open the published movie in two browser windows.
4. Hold a seat in the first window and show it updating immediately in the second.
5. Complete the simulated checkout and display the QR ticket in booking history.
6. Enter the ticket reference on the admission page, then show that a second admission is rejected.

## Tests

Run backend unit tests and static analysis:

```bash
cd backend
go test ./...
go vet ./...
```

The seat-race integration test runs when a dedicated PostgreSQL URL is provided:

```bash
TIXIGO_TEST_DATABASE_URL='postgres://tixigo:tixigo@localhost:5432/tixigo_test?sslmode=disable' go test ./internal/seat
```

Run the frontend build and booking journey:

```bash
cd frontend
pnpm build
pnpm exec playwright install chromium
pnpm test:e2e
```

GitHub Actions runs the database-backed Go suite, vet, frontend build, and Playwright test on pushes and pull requests.

## API overview

- `/api/v1/auth/*` — account and session lifecycle
- `/api/v1/movies/*` — public discovery
- `/api/v1/screenings/*` — seat maps, holds, and waitlists
- `/api/v1/bookings/*` — customer tickets and cancellation
- `/api/v1/organiser/*` — movie and screening management
- `/api/v1/admin/*` — users, venues, and approvals
- `/api/v1/tickets/admit` — staff ticket validation
- `/ws/screenings/{id}` — live seat notifications

## Scope and trade-offs

This is a focused demonstration project, not a commercial ticketing platform. Real payments, refunds, horizontally distributed WebSockets, native mobile apps, and advanced analytics are intentionally outside the current scope. The in-memory realtime hub is appropriate for a single API instance; a multi-instance deployment would require a shared event layer such as Redis Pub/Sub.

Deployment and backup notes are available in [infra/OPERATIONS.md](infra/OPERATIONS.md).
