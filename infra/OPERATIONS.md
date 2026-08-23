# Tixigo operations

## Deployment

- Create the API from `render.yaml`, then set `DATABASE_URL`, `WEB_ORIGIN`, Resend, and Cloudinary secrets in Render.
- Import `frontend/` into Vercel. Set `VITE_API_BASE_URL=https://<api-host>/api/v1` and `VITE_WS_BASE_URL=wss://<api-host>/ws`.
- Run `go run ./cmd/migrate` from the backend release environment before deploying a schema-dependent API version.

## Backups

Run `infra/backup-postgres.sh` daily from a secured scheduler with `DATABASE_URL` set. It creates a compressed custom-format dump and retains local dumps for 14 days. Copy dumps to encrypted object storage with lifecycle retention. Test `infra/restore-postgres.sh` against a non-production database monthly.

Never pass production database credentials on the command line or commit them to this repository.
