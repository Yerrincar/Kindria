<<<<<<< HEAD
# Kindria

TUI EPUB library manager.

**Demo:** [`assets/kindria.mp4`](assets/kindria.mp4)

## Highlights

- EPUB library management in a fast terminal UI
- Split workflow: sidebar navigation + content panel
- Dedicated views for **Library**, **To-Be Read**, **Add Book**, **Kindle Sync**, and **Themes**
- Multi-file add flow with duplicate checks and import stats (`Inserted / Failed / Duplicated`)
- Kindle synchronization pipeline with conversion to EPUB (via Calibre)
- Status and rating management (`Read`, `Unread`, `To Be Read`, stars)
- Reading date tracking when status changes to `Read`
- Theme selection with persistent saved preference
- Cover rendering and caching in graphics-capable terminals
- Vim-style keybindings plus arrow-key support

## Install

### Quick setup (recommended)

```bash
make init
make run
```

### Build from source (Go)

Requires Go (`go 1.25+` in `go.mod`):

```bash
go build -o bin/kindria .
./bin/kindria
```

### Prebuilt Linux binaries

Not published yet. For now, build from source.

## Prerequisites

<details>
<summary>Click to expand</summary>

- **Go**: required to build/run from source
- **Calibre (`ebook-convert`)**: required for Kindle sync conversion flow
- **GVFS / gio**: required for Kindle MTP detection/copy on Linux
- **sqlite3 CLI**: only needed for `make db-init` on fresh DB bootstrap

</details>

## Quick Start

```bash
make init
make run
```

Or directly:

```bash
go run .
```

## Documentation

- This `README.md` is the main setup/usage document
- In-app key hints are shown in each screen (Library, Add Book, Kindle, Themes)

## Terminal Notes

Cover rendering works best in terminals with graphics protocol support.

- Best experience: **Kitty**
- Also works with terminals/protocols supported by `go-termimg`
- In terminals without graphics support, core management features still work

Kindle sync depends on host integration (MTP + `gio`) and is Linux-oriented.

## Data Storage

Kindria currently stores project data in local repository paths plus a user config entry for theme.

| Data | Location |
|---|---|
| Library database | `./books.db` |
| Imported books | `./books/` |
| Cover cache | `./cache/covers/` |
| Log file | `./kindria.log` |
| Theme setting | `${XDG_CONFIG_HOME}/kindria/theme.json` or `~/.config/kindria/theme.json` |

## Dependency Notes

- SQL access code in `internal/core/db/` is generated via `sqlc` from:
  - `internal/core/platform/storage/queries/books.sql`
  - `internal/core/platform/storage/migrations/*.sql`

## Attribution

Created and maintained in this repository by the Kindria project author.

## License

MIT License. See [`LICENSE`](LICENSE).
=======
### Kindria, a TUI E-book library 
assets/kindria.mp4
>>>>>>> 4b2c372420cf7c71dcc8a93eae1013262e1171f0
