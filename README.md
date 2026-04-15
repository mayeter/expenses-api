# Expenses API (MVP)

REST API for expense records: categories, **per-user lists**, and expenses scoped by list. When `INTERNAL_API_SECRET` is set, the Next BFF must send `X-Internal-Secret` and `X-Clerk-User-Id`.

## Stack

| Layer      | Choice                          |
| ---------- | ------------------------------- |
| Language   | **Go 1.22+**                    |
| HTTP       | **[Fiber](https://gofiber.io/)** v2 |
| Database   | **PostgreSQL**                  |
| ORM        | **GORM**                        |

**Hosting:** Railway, Render, Fly.io, etc. Deploy with `Dockerfile`.

## API (summary)

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | `/health` | Health check |
| GET | `/api/categories` | Categories (seeded on first run) |
| GET | `/api/lists?scope=mine` or `scope=shared` | User lists (`shared` empty until sharing exists) |
| POST | `/api/lists` | Create list |
| PATCH | `/api/lists/:listId` | Update list (`name`, `isFavorite`) |
| DELETE | `/api/lists/:listId` | Delete list (owner; soft-deletes expenses) |
| GET | `/api/lists/:listId/expenses` | List expenses. Query: `from`, `to` (`YYYY-MM-DD`) |
| POST | `/api/lists/:listId/expenses` | Create expense |
| PATCH | `/api/lists/:listId/expenses/:id` | Partial update |
| DELETE | `/api/lists/:listId/expenses/:id` | Soft delete |

**POST / PATCH body:** `amount` (decimal TRY), `categoryId`, optional `note`, optional `occurredAt` (`RFC3339` or `YYYY-MM-DD`). Responses use `currency: "TRY"`.

## Schema notes

- `expenses.clerk_user_id` — set when using Clerk + BFF.
- `expenses.list_id` — FK to `expense_lists`.

## Local development

```bash
docker compose up -d
cp .env.example .env
go run ./cmd/server
```

Default port **:8080**.

## Environment

| Variable | Required | Description |
| -------- | -------- | ----------- |
| `DATABASE_URL` | yes | Postgres DSN |
| `PORT` | no | Default `8080` |
| `ALLOWED_ORIGINS` | no | CORS (tighten in prod, e.g. Vercel origin instead of `*`) |
| `INTERNAL_API_SECRET` | no | Shared secret for BFF; empty skips internal auth |

## Vercel

This service is a long-running API, typically on **Render / Railway / Fly**. The Next frontend may live on Vercel and call the API via the server-side BFF proxy.

## Repo

This directory may be a standalone **git** repository.
