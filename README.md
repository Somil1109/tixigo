# Tixigo

Tixigo is an India-only cinema ticket-booking platform: a React + TypeScript frontend and a Go + Chi API backed by PostgreSQL.

## Product decisions

- India-only cinema platform, with all showtimes in `Asia/Kolkata`.
- Public movie discovery; verified customers must sign in to hold, book, cancel, or join a waitlist.
- One account system with customer, organiser, and admin roles. New users begin as customers; admins promote organisers.
- Venues use structured seat-layout JSON. Each screening receives its own snapshot of the venue seats.
- Up to 10 arbitrary seats per booking. Holds last 10 minutes.
- Waitlist entries specify category and quantity. Offers reserve matching seats for one hour, using flexible fulfilment.
- Simulated payment in v1, QR tickets by email and from booking history.

## Repository layout

```
frontend/    React + TypeScript + Vite + Tailwind application
backend/     Go + Chi API, SQL migrations, and Dockerfile
docker-compose.yml
```

## Local development

1. Copy `backend/.env.example` to `backend/.env` and set long JWT secrets.
2. Start dependencies: `docker compose up -d`.
3. Run database migrations: `cd backend && go run ./cmd/migrate`.
4. Start the API: `cd backend && go run ./cmd/server`.
5. Install frontend dependencies: `cd frontend && pnpm install`.
6. Start the app: `cd frontend && pnpm dev`.

Mailpit is available at `http://localhost:8025` in development.

## Deployment

- Web: Vercel
- API: Render
- Database: Neon PostgreSQL
- Email: Resend
- Posters: Cloudinary

Production configuration lives in `render.yaml`, `frontend/vercel.json`, and `infra/OPERATIONS.md`. GitHub Actions runs backend unit/integration checks, vet, the frontend build, and the Playwright booking journey.
