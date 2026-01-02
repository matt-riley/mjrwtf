---
title: TUI CLI
description: Interactive terminal UI for managing short URLs.
---

This document covers how to install and use the **mjr.wtf** Bubble Tea TUI, plus the agreed UX/navigation model.

## Demo

![Interactive TUI showing URL management with vim-like keybindings, create form, and analytics views](/images/tui/demo.gif)

*The mjr.wtf TUI in action - navigate URLs, create short links, view analytics, all with vim-like keybindings.*

## Install / run

The TUI is shipped as part of the `mjr` CLI.

```bash
# Build from source
make build-mjr
./bin/mjr tui

# Or run without building a binary
go run ./cmd/mjr tui

# Or install with Go
go install github.com/matt-riley/mjrwtf/cmd/mjr@latest
mjr tui
```

## Configuration

Configuration precedence:

1. Flags
2. Environment variables
3. Config file
4. Defaults

### Environment variables

- `MJR_BASE_URL` (default: `http://localhost:8080`)
- `MJR_TOKEN` (required for authenticated API calls)

```bash
# Local server
export MJR_BASE_URL=http://localhost:8080
export MJR_TOKEN=token-current
mjr tui

# Against prod
export MJR_BASE_URL=https://mjr.wtf
export MJR_TOKEN=token-current
mjr tui
```

### Config file

Default search path:

1. `~/.config/mjrwtf/config.yaml` (preferred)
2. `~/.config/mjrwtf/config.toml`

Example YAML:

```yaml
base_url: http://localhost:8080
token: token-current
```

## Common workflows

From the URL list (default screen):

- **List / refresh**: start the app; press `r` to refresh
- **Create**: press `c`, fill the form, submit
- **Analytics**: select a URL then press `a`
- **Delete**: select a URL, press `d`, then confirm with `Enter`/`y`

## Security notes

- Avoid passing tokens on the command line (`--token ...`) since they can be captured in shell history and process lists.
- Prefer `MJR_TOKEN` or a config file; if you must paste a token interactively, use a technique like `read -s` in your shell.

## Troubleshooting

- **401 Unauthorized**: ensure your token matches one of the server's configured `AUTH_TOKENS`/`AUTH_TOKEN`.
- **429 Too Many Requests**: you hit the API rate limit; wait for `Retry-After` and/or refresh less frequently.

## UX, navigation, and keybindings (design)

This section records the agreed UX/navigation model for the **mjr.wtf** Bubble Tea TUI.

## Entrypoint

The TUI is invoked from the main CLI binary:

```bash
mjr tui
```

## Screens and navigation model

### 1) URL list (default screen)

Purpose: browse/manage short URLs.

Layout (conceptual):
- Header: app name + active base URL
- Main list: URLs (short code + destination + created at)
- Footer/status bar: key hints + current page + transient messages

Behaviors:
- On start, fetch `GET /api/urls` and render a selectable list.
- Selection moves with vim-like keys (`j/k`) and arrows.
- Pagination is explicit (next/prev page).
- Filtering is a client-side filter over the currently loaded page unless/until we implement server-side filtering.

### 2) Create URL (modal / form)

Purpose: create a new short URL.

Inputs:
- `original_url` (required)
- `short_code` (optional; allow blank to auto-generate if supported server-side)

Text fields: prefer vim-like cursor movement/editing where feasible.

Behaviors:
- Submitting calls `POST /api/urls`.
- Success: toast + return to list + refresh.
- Validation errors: show inline error + keep the form open.

### 3) Analytics detail

Purpose: view click analytics for the selected short URL.

Behaviors:
- Opening calls `GET /api/urls/{shortCode}/analytics`.
- Show totals and breakdowns (as supported by the endpoint response).
- Support an optional time range (RFC3339 `start_time`/`end_time`).
  - Validate: start/end provided together; `start_time < end_time`.
- Provide a clear "back" path to the list.
- Large breakdown maps should be usable via truncation and/or scrolling.

### 4) Delete confirmation

Purpose: prevent accidental deletes.

Behaviors:
- Confirming calls `DELETE /api/urls/{shortCode}`.
- Success: toast + return to list; update list optimistically (remove the item) and/or refresh.
- NotFound: show a non-fatal status message and refresh the list.
- Failure: toast + return to list (selection preserved if possible).

Key idea: deletion is always a **two-step** interaction — `d` opens the confirmation view, then a second explicit action confirms.

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `q` | Quit |
| `r` | Refresh current view |

### URL list

| Key | Action |
|-----|--------|
| `j` / `k` | Move selection down/up (vim-like) |
| `↑` / `↓` | Move selection down/up |
| `n` / `p` | Next / previous page |
| `/` | Filter (enter filter mode) |
| `c` | Create URL |
| `d` | Delete selected URL (opens confirmation) |
| `a` | Analytics for selected URL |

### Delete confirmation

| Key | Action |
|-----|--------|
| `Enter` / `y` | Confirm delete |
| `Esc` / `n` | Cancel and return to list |

### Analytics detail

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll down/up |
| `↑` / `↓` | Scroll down/up |
| `t` | Set time range (optional) |
| `b` / `Esc` | Back to list |
| `r` | Refresh analytics |

### Analytics time range mode

| Key | Action |
|-----|--------|
| `Tab` | Switch between start/end fields |
| `Enter` | Next field / apply |
| `Esc` | Cancel |

### Filter mode

| Key | Action |
|-----|--------|
| `Enter` | Apply filter |
| `Esc` | Cancel / clear filter |

## Loading and error UX

- While an API call is in flight, show a spinner and keep the UI responsive.
- Errors should appear as a transient toast in the status bar/footer.
- Startup config warnings should also be shown as a toast.

### Config warning toast

If both config files exist:

- `~/.config/mjrwtf/config.yaml`
- `~/.config/mjrwtf/config.toml`

…the TUI should **prefer YAML** and show a toast explaining that only one config file is needed.

## Config and auth behavior

### Sources and precedence

Configuration resolution is:

1. **Flags**
2. **Environment variables**
3. **Config file**
4. **Defaults**

### Config file location

Default search path:

1. `~/.config/mjrwtf/config.yaml` (preferred)
2. `~/.config/mjrwtf/config.toml`

If both exist, prefer YAML and show a toast.

### Suggested config fields

- `base_url`: base URL for the mjr.wtf API (e.g. `https://mjr.wtf`)
- `token`: API bearer token

Example YAML:

```yaml
base_url: http://localhost:8080
token: token-current
```

Example TOML:

```toml
base_url = "http://localhost:8080"
token = "token-current"
```

## API references

This TUI uses the existing OpenAPI contract:

- `GET /api/urls`
- `POST /api/urls`
- `DELETE /api/urls/{shortCode}`
- `GET /api/urls/{shortCode}/analytics`

See: `openapi.yaml`
