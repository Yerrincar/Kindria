# Architecture

## Overview

Kindria is a Go terminal application built with Bubble Tea.  
It manages local EPUB books in `./books`, stores metadata in SQLite (`./books.db`), and renders a multi-view TUI for browsing and book operations.

## Runtime Flow

1. `main.go` opens `./books.db` and builds `metadata.Handler`.
2. Startup sync runs `InsertBooks()` to discover/import new local `.epub` files from `./books`.
3. Existing rows are loaded with `SelectBooks()` and passed to `tui.InitialModel(...)`.
4. TUI runs in Bubble Tea alt screen.
5. Cover cache update starts in background.

## Project Structure

- `main.go`: app bootstrap, DB open, TUI startup, logging.
- `internal/tui/model.go`: UI states, input handling, rendering, add-book flow, Kindle flow wiring.
- `internal/tui/theme/themes.go`: palettes + persisted theme selection.
- `internal/core/api/books/bookMetadata.go`: metadata extraction, DB orchestration, cover pipeline entry points.
- `internal/core/db/`: sqlc-generated query layer.
- `internal/core/platform/storage/queries/books.sql`: source SQL used by sqlc.
- `internal/core/platform/storage/migrations/`: goose-style SQL migrations.
- `tools/kindleBookExtraction.go`: Kindle detection (`gio`), MTP copy, conversion via Calibre, sync result stats.
- `internal/utils/`: shared helpers for copy/delete and visual helpers.

## Main Functional Flows

### Add Book

1. User enters file picker view.
2. User selects one or more files.
3. On synchronize/import key, each selected file is validated:
4. Duplicate checks run against DB (`file_name`) and local `./books` filenames.
5. Valid files are copied to `./books`.
6. `InsertBooks()` runs to extract metadata and insert only missing books.
7. Library data is refreshed in UI and import stats are shown.

### Kindle Synchronize

1. Kindle mount is detected via `gio mount -li` MTP URI parsing.
2. Kindle documents URI is listed with `gio list`.
3. Convertible formats are filtered (`.epub`, `.azw`, `.azw3`, `.mobi`, `.pdf`, `.txt`).
4. Files are copied to temp storage using `gio copy`.
5. Non-EPUB files are converted with `ebook-convert` to `.epub`.
6. Duplicate checks run (DB + local folder).
7. New books are copied into `./books`.
8. `InsertBooks()` and `SelectBooks()` refresh app data.

### Status / Reading Date

- Book status changes are persisted through `UpdateStatus`.
- `reading_date` is set when status becomes `Read`.
- `reading_date` is cleared for other statuses.
- To-Be Read view is filtered so only books in that status remain visible after updates.

## Theme System

- Themes are selected in the TUI `Themes` state.
- Palette fields drive border, highlight, normal text, and home logo gradient.
- Selection is persisted in:
  - `${XDG_CONFIG_HOME}/kindria/theme.json`
  - or `~/.config/kindria/theme.json`

## Notes and Constraints

- Kindle sync is Linux-oriented and depends on GVFS/MTP tooling (`gio`).
- Conversion depends on Calibre CLI (`ebook-convert`).
- SQLite schema includes runtime safety for `reading_date` (`EnsureReadingDateColumn`) to support existing DBs.
