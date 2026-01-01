# TUI CLI: UX, navigation, and keybindings (design)

This document records the agreed UX/navigation model for the **mjr.wtf** Bubble Tea TUI.

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
- Provide a clear “back” path to the list.

### 4) Delete confirmation

Purpose: prevent accidental deletes.

Behaviors:
- Confirming calls `DELETE /api/urls/{shortCode}`.
- Success: toast + return to list + refresh.
- Failure: toast + return to list (selection preserved if possible).

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
