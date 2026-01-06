# HTML Templates (templ)

This directory contains **templ** templates for server-side HTML rendering.

## Structure

```
internal/adapters/http/templates/
├── layouts/  # Shared layouts (base, header, footer)
└── pages/    # Full page templates
```

## What is templ?

[templ](https://github.com/a-h/templ) is a templating language for Go that generates type-safe HTML. Templates are written in `.templ` files and generated into Go code.

## Development Workflow

### 1) Create/Edit templates

Create or modify `.templ` files in this directory (match the directory’s Go package name):

```templ
package layouts

templ ExampleLayout(title string, content templ.Component) {
    <html>
        <head><title>{ title }</title></head>
        <body>@content</body>
    </html>
}
```

### 2) Generate Go code

After editing templates, regenerate code:

```bash
make templ-generate
# or: make generate
```

Generated files (e.g. `*_templ.go`) are **checked in** to the repository; don’t edit them by hand.

### 3) Use in handlers

```go
import "github.com/matt-riley/mjrwtf/internal/adapters/http/templates/pages"

func HomeHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    _ = pages.Home().Render(r.Context(), w)
}
```

## Tips

### Auto-regeneration

```bash
make templ-watch
```

### Testing

Prefer testing the handlers that render templates (integration/handler tests), rather than unit-testing templates directly.
