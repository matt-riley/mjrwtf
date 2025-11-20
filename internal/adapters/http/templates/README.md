# HTML Templates

This directory contains Templ templates for server-side HTML rendering.

## Structure

```
templates/
├── layouts/       # Layout templates (base, header, footer)
├── pages/         # Full page templates
└── components/    # Reusable UI components (future)
```

## What is Templ?

[Templ](https://github.com/a-h/templ) is a templating language for Go that generates type-safe HTML. Templates are written in `.templ` files and compiled to Go code.

## Development Workflow

### 1. Create/Edit Templates

Create or modify `.templ` files in this directory:

```templ
// Example: components/button.templ
package components

templ Button(text string, href string) {
    <a href={ href } class="btn">
        { text }
    </a>
}
```

### 2. Generate Go Code

After creating or modifying templates, generate the Go code:

```bash
make templ-generate
```

This creates `*_templ.go` files alongside your `.templ` files. These generated files are ignored by git.

### 3. Use in Handlers

Import and use the generated components in your handlers:

```go
import "github.com/matt-riley/mjrwtf/internal/adapters/http/templates/pages"

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    pages.Home().Render(r.Context(), w)
}
```

## Available Templates

### Layouts

- **base.templ**: Base layout with header, navigation, footer, and Tailwind CSS

### Pages

- **home.templ**: Landing page with features and getting started guide
- **error.templ**: Error pages (404 Not Found, 500 Internal Server Error)

## Development Tips

### Auto-regeneration

For development, use watch mode to automatically regenerate templates on save:

```bash
make templ-watch
```

### Component Composition

Templ supports component composition. Components can call other components:

```templ
// layouts/base.templ
templ Base(title string, content templ.Component) {
    <!DOCTYPE html>
    <html>
        <head><title>{ title }</title></head>
        <body>
            @Header()
            @content
            @Footer()
        </body>
    </html>
}

// pages/home.templ
templ Home() {
    @layouts.Base("Home", homeContent())
}

templ homeContent() {
    <h1>Welcome</h1>
}
```

### Passing Data

Templates can accept any Go types as parameters:

```templ
templ UserProfile(user *domain.User) {
    <div class="profile">
        <h2>{ user.Name }</h2>
        <p>{ user.Email }</p>
    </div>
}
```

### Conditional Rendering

Use Go's `if` statements directly:

```templ
templ Message(text string, isError bool) {
    if isError {
        <div class="error">{ text }</div>
    } else {
        <div class="success">{ text }</div>
    }
}
```

### Loops

Iterate over slices using Go's `for` loops:

```templ
templ List(items []string) {
    <ul>
        for _, item := range items {
            <li>{ item }</li>
        }
    </ul>
}
```

## Styling

Templates use [Tailwind CSS](https://tailwindcss.com/) via CDN for styling. In production, consider:

1. Using a build step with Tailwind CLI for smaller CSS bundles
2. Self-hosting Tailwind CSS instead of using CDN
3. Adding custom CSS for branding

## Best Practices

1. **Keep templates simple**: Complex logic belongs in handlers/use cases
2. **Use components**: Break down pages into reusable components
3. **Type safety**: Leverage Templ's type safety by passing domain types
4. **Accessibility**: Include proper ARIA labels and semantic HTML
5. **Performance**: Templates are compiled to efficient Go code
6. **Testing**: Test handlers that use templates, not templates directly

## References

- [Templ Documentation](https://templ.guide/)
- [Templ GitHub](https://github.com/a-h/templ)
- [Tailwind CSS Documentation](https://tailwindcss.com/docs)
