# Development

## Requirements

- Go 1.25+
- `sqlite3` CLI (needed for fresh `make db-init`)
- `ebook-convert` (Calibre CLI, needed for Kindle conversion)
- `gio` (GVFS tools, needed for Kindle MTP scan/copy on Linux)

## Quick Setup

```bash
make init
make run
```

## Common Commands

```bash
make run      # run app
make build    # build ./bin/kindria
make fmt      # go fmt ./...
make test     # go test ./...
make clean    # remove ./bin
make db-init  # bootstrap new books.db from migrations
```

## Database and Migrations

- SQLite DB file: `./books.db`
- Migration files: `internal/core/platform/storage/migrations/*.sql`
- Query source: `internal/core/platform/storage/queries/books.sql`
- Generated code: `internal/core/db/*.go`

Current flow:

- `make db-init` creates a fresh DB and applies each migration's `-- +goose Up` section.
- If DB already exists, `db-init` skips.
- Runtime also calls `EnsureReadingDateColumn()` so older DBs can still run.

## sqlc Workflow

After changing SQL in `internal/core/platform/storage/queries/books.sql`:

```bash
sqlc generate
go build ./...
```

## Kindle Sync Notes

- Kindle access uses MTP URIs (`mtp://...`) discovered via `gio`.
- Sync implementation lives in `tools/kindleBookExtraction.go`.
- Books are copied from Kindle to a temp dir, converted when needed, then copied to `./books`.
- Original files on Kindle are not modified.

## Logs and Debugging

- Main log file: `./kindria.log`
- App can dump goroutine stacks with `SIGUSR1` (handled in `main.go`).

Example:

```bash
pkill -USR1 -f kindria
```

## Known Platform Scope

- Primary support target today is Linux.
- Kindle synchronization depends on Linux MTP/GVFS behavior.
