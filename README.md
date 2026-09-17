# Chirpy

A RESTful JSON API for a Twitter-like microblogging service, written in Go with
a PostgreSQL backend. Users register, post short messages ("chirps"), and
authenticate with JWT access tokens backed by revocable refresh tokens.

Built as the capstone project for the Boot.dev "Learn HTTP Servers in Go"
course.

## Features

- User registration and login with [argon2id](https://github.com/alexedwards/argon2id) password hashing
- JWT access tokens (1 hour) plus opaque, server-side refresh tokens (60 days) that can be revoked
- Chirp creation, listing, filtering by author, and owner-only deletion
- Profanity filtering on chirp bodies
- "Chirpy Red" memberships granted through an authenticated webhook from the payment provider
- Static file server with request metrics

## Requirements

| Tool | Version | Purpose |
|---|---|---|
| [Go](https://go.dev/dl/) | 1.27.1+ | Building and running the server |
| [Docker](https://docs.docker.com/get-docker/) + Compose | any recent | Running PostgreSQL locally |
| [goose](https://github.com/pressly/goose) | v3 | Database migrations |
| [sqlc](https://sqlc.dev/) | v1.30+ | Generating Go code from SQL (only needed if you change queries) |

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

## Setup

**1. Clone and install dependencies**

```bash
git clone https://github.com/thelol3882/chirpy.git
cd chirpy
go mod download
```

**2. Start PostgreSQL**

```bash
docker compose up -d
```

**3. Create the database**

The Compose file starts a Postgres server but does not create the application
database, so create it once:

```bash
docker compose exec postgres createdb -U postgres chirpy
```

**4. Create a `.env` file**

```bash
cp .env.example .env
```

Then fill in `SECRET_KEY` and `POLKA_KEY`. See
[Environment variables](#environment-variables) below.

**5. Run the migrations**

```bash
cd sql/schema
goose postgres "postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable" up
cd ../..
```

**6. Start the server**

```bash
go run .
```

The server listens on `http://localhost:8080` and serves the static site at
`/app/`.

### Environment variables

All four are required; the server exits at startup if any is missing.

| Variable | Example | Description |
|---|---|---|
| `DB_URL` | `postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable` | PostgreSQL connection string |
| `PLATFORM` | `dev` | Deployment environment. `POST /admin/reset` is only permitted when this is `dev` |
| `SECRET_KEY` | *(64 random bytes, base64)* | Signing key for JWTs. Generate with `openssl rand -base64 64` |
| `POLKA_KEY` | *(provided by Polka)* | API key the payment provider sends with webhook requests |

`.env` is gitignored and must never be committed.

## API reference

All request and response bodies are JSON. Timestamps are RFC 3339 in UTC.

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| GET | `/api/healthz` | — | Readiness check |
| POST | `/api/users` | — | Create a user |
| PUT | `/api/users` | Bearer (access) | Update own email and password |
| POST | `/api/login` | — | Log in; returns both tokens |
| POST | `/api/refresh` | Bearer (refresh) | Exchange a refresh token for a new access token |
| POST | `/api/revoke` | Bearer (refresh) | Revoke a refresh token |
| POST | `/api/chirps` | Bearer (access) | Create a chirp |
| GET | `/api/chirps` | — | List chirps, optionally filtered by author and sorted |
| GET | `/api/chirps/{chirpID}` | — | Fetch a single chirp |
| DELETE | `/api/chirps/{chirpID}` | Bearer (access) | Delete one of your own chirps |
| POST | `/api/polka/webhooks` | ApiKey | Membership webhook from the payment provider |
| GET | `/admin/metrics` | — | Fileserver hit counter (HTML) |
| POST | `/admin/reset` | — | Delete all users. Only when `PLATFORM=dev` |

Errors are returned as `{"error": "message"}` with the appropriate status code.

## Authentication

Chirpy issues two different tokens, and they are not interchangeable.

**Access token** — a JWT, valid for **1 hour**, issued by `/api/login` and
`/api/refresh`. Send it on authenticated requests:

```
Authorization: Bearer <access token>
```

It is stateless: the server verifies the signature and expiry without touching
the database, which is what keeps authenticated requests cheap.

**Refresh token** — an opaque 64-character hex string, valid for **60 days**,
issued by `/api/login`. It is stored server-side and is accepted *only* by
`/api/refresh` and `/api/revoke`, using the same header format. Because it is a
database row rather than a signature, it can be revoked.

**Revocation is not instant logout.** Revoking a refresh token prevents new
access tokens from being issued, but an access token already in the client's
hands remains valid until it expires. The worst-case window is therefore one
hour. This is the deliberate trade-off of stateless access tokens: no database
lookup per request, in exchange for delayed revocation.

## Endpoints

### `GET /api/healthz`

Returns `200 OK` with the plain-text body `OK`.

---

### `POST /api/users`

Creates a user. The password is hashed with argon2id before storage and is
never returned.

**Request**

```json
{
  "email": "lane@example.com",
  "password": "04234"
}
```

**Response** `201 Created`

```json
{
  "id": "5a47789c-a617-444a-8a80-b50359247804",
  "created_at": "2021-07-01T00:00:00Z",
  "updated_at": "2021-07-01T00:00:00Z",
  "email": "lane@example.com",
  "is_chirpy_red": false
}
```

| Status | Cause |
|---|---|
| 400 | `email` or `password` missing |
| 500 | Email already registered, or the user could not be created |

---

### `PUT /api/users`

Updates the email and password of the **authenticated** user. The user is
identified by the access token, never by the request body, so this endpoint
cannot modify another account.

**Headers:** `Authorization: Bearer <access token>`

**Request**

```json
{
  "email": "new@example.com",
  "password": "newpassword"
}
```

**Response** `200 OK` — the updated user, same shape as `POST /api/users`.

| Status | Cause |
|---|---|
| 400 | `email` or `password` missing |
| 401 | Missing or invalid access token |
| 500 | Email already taken, or the update failed |

---

### `POST /api/login`

**Request**

```json
{
  "email": "lane@example.com",
  "password": "04234"
}
```

**Response** `200 OK`

```json
{
  "id": "5a47789c-a617-444a-8a80-b50359247804",
  "created_at": "2021-07-01T00:00:00Z",
  "updated_at": "2021-07-01T00:00:00Z",
  "email": "lane@example.com",
  "is_chirpy_red": false,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "56aa826d22baab4b5ec2cea41a59ecbba03e542aedbb31d9b80326ac8ffcfa2a"
}
```

| Status | Cause |
|---|---|
| 400 | `email` or `password` missing |
| 401 | `Incorrect email or password` |

A wrong password and an unknown email return the identical 401 response, so the
endpoint cannot be used to discover which addresses are registered.

---

### `POST /api/refresh`

Issues a new access token. Takes no request body.

**Headers:** `Authorization: Bearer <refresh token>`

**Response** `200 OK`

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

| Status | Cause |
|---|---|
| 401 | Refresh token missing, unknown, expired, or revoked |

---

### `POST /api/revoke`

Revokes a refresh token. Takes no request body and returns no body.

**Headers:** `Authorization: Bearer <refresh token>`

**Response** `204 No Content`

| Status | Cause |
|---|---|
| 401 | Missing or malformed `Authorization` header |

---

### `POST /api/chirps`

Creates a chirp authored by the authenticated user. The author is taken from
the access token, not from the request body.

**Headers:** `Authorization: Bearer <access token>`

**Request**

```json
{
  "body": "Hello, world!"
}
```

**Response** `201 Created`

```json
{
  "id": "94b7e44c-3604-42e3-bc9a-8ff6f0f6d17b",
  "created_at": "2021-07-07T00:00:00Z",
  "updated_at": "2021-07-07T00:00:00Z",
  "body": "Hello, world!",
  "user_id": "5a47789c-a617-444a-8a80-b50359247804"
}
```

| Status | Cause |
|---|---|
| 400 | Body exceeds 140 characters |
| 401 | Missing or invalid access token |

**Note:** chirp bodies are filtered for profanity before they are stored. The
words `kerfuffle`, `sharbert` and `fornax` are replaced with `****`,
case-insensitively, on whole-word matches only. The request still succeeds with
`201`, and the response contains the censored text — so a stored chirp may
differ from what was submitted.

---

### `GET /api/chirps`

Returns chirps ordered by `created_at`, oldest first by default.

**Query parameters**

| Parameter | Values | Default | Description |
|---|---|---|---|
| `author_id` | UUID | — | Return only chirps by this user |
| `sort` | `asc`, `desc` | `asc` | Order by `created_at`, ascending or descending |

Both parameters may be combined. An unrecognised `sort` value is ignored and
the default ascending order is used.

```
GET /api/chirps
GET /api/chirps?author_id=5a47789c-a617-444a-8a80-b50359247804
GET /api/chirps?sort=desc
GET /api/chirps?author_id=5a47789c-a617-444a-8a80-b50359247804&sort=desc
```

**Response** `200 OK` — a JSON array of chirps. An author with no chirps, or an
unknown but well-formed UUID, returns `[]` rather than an error.

| Status | Cause |
|---|---|
| 400 | `author_id` is not a valid UUID |

Chirps sharing an identical `created_at` have no guaranteed order relative to
one another, and that order is not stable between `asc` and `desc`.

---

### `GET /api/chirps/{chirpID}`

**Response** `200 OK` — a single chirp.

| Status | Cause |
|---|---|
| 400 | `chirpID` is not a valid UUID |
| 404 | No chirp with that ID |

---

### `DELETE /api/chirps/{chirpID}`

Deletes a chirp. Only the author may delete their own chirp.

**Headers:** `Authorization: Bearer <access token>`

**Response** `204 No Content`

| Status | Cause |
|---|---|
| 400 | `chirpID` is not a valid UUID |
| 401 | Missing or invalid access token |
| 403 | The chirp exists but belongs to another user |
| 404 | No chirp with that ID |

---

### `POST /api/polka/webhooks`

Receives membership events from Polka, the payment provider. Authenticated with
a shared API key rather than a user token, using a different scheme:

```
Authorization: ApiKey <POLKA_KEY>
```

**Request**

```json
{
  "event": "user.upgraded",
  "data": {
    "user_id": "3311741c-680c-4546-99f3-fc9efac2036c"
  }
}
```

**Response** `204 No Content`

Any event other than `user.upgraded` is ignored and also answered with `204`.
Polka retries on any non-2xx response, so "received and intentionally ignored"
must be reported as success. The handler is idempotent: delivering the same
upgrade event repeatedly has the same effect as delivering it once.

| Status | Cause |
|---|---|
| 400 | `user_id` is not a valid UUID |
| 401 | API key missing, wrong, or sent with the wrong scheme |
| 404 | No user with that ID |

---

### `GET /admin/metrics`

Returns an HTML page showing how many times the `/app/` fileserver has been
hit.

---

### `POST /admin/reset`

Deletes all users, and every chirp and refresh token belonging to them via
`ON DELETE CASCADE`. Also resets the fileserver hit counter.

**Response** `200 OK`

| Status | Cause |
|---|---|
| 403 | `PLATFORM` is not set to `dev` |

## Development

**Run the tests**

```bash
go test ./...
```

The `internal/auth` package covers password hashing, JWT creation and
validation, and `Authorization` header parsing.

**Regenerate database code**

Queries live in `sql/queries/`. After editing them:

```bash
sqlc generate
```

This rewrites `internal/database/`, which is generated code and should not be
edited by hand.

**Add a migration**

Migrations live in `sql/schema/` and are applied in numeric order. Create a new
numbered file with `-- +goose Up` and `-- +goose Down` sections, then:

```bash
cd sql/schema
goose postgres "$DB_URL" up
```

Migrations are **append-only**. Never edit a migration that has already been
applied — goose records it as run and will not execute it again, so the change
would silently apply to new databases only. Fix mistakes with a new migration.

Test the `Down` section as well as the `Up`, since `goose up` never executes it
and a broken rollback stays hidden until you need it:

```bash
goose postgres "$DB_URL" down
goose postgres "$DB_URL" up
```

## Project layout

```
main.go              server setup, configuration, routes
auth.go              request authentication helper
json.go              JSON response helpers
users.go             user resource and handlers
chirps.go            chirp resource and handlers
refresh.go           refresh and revoke handlers
polka.go             payment provider webhook
admin.go             metrics and reset
internal/auth/       passwords, JWTs, Authorization header parsing
internal/database/   sqlc-generated code (do not edit)
sql/schema/          goose migrations
sql/queries/         sqlc query definitions
```
