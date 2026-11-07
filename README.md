# Todos API

Deeper Than Code Go example: Go 1.27, chi v5, pgx v5, and goose v3. Postgres arrives as `DATABASE_URL`; the process applies `db/migrations` on boot, then serves the shared Todos contract.

Live: https://api-production-a3ab.up.railway.app

| Request | Response |
|---|---|
| GET /health | 200 `{"ok":true}` |
| GET /todos | 200 `[{id,subject,details}]` ordered by id |
| GET /todos/:id | 200 object, or 404 `{"error":"not found"}` |
| POST /todos `{subject,details}` | 201 object |
| PUT /todos/:id `{subject,details}` | 200 object, or 404 |
| DELETE /todos/:id | 204 empty, or 404 |
| blank or missing subject or details | 400 `{"error":"subject and details are required"}` |
| malformed JSON | 400 `{"error":"invalid json"}` |

Prove it against the live URL:

```
node check.mjs https://api-production-a3ab.up.railway.app
```

Environment variables it reads: `DATABASE_URL` (required) and `PORT` (falls back to 8080).

Do not commit that URL.
