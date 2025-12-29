---
title: Error handling
description: Error handling policy and conventions.
---


This project follows idiomatic Go error handling:

- Return errors, don’t panic (except in tests / truly unrecoverable programmer errors).
- Preserve **error identity** across boundaries so callers can use `errors.Is` / `errors.As`.
- Add **context** at abstraction boundaries without losing identity.

## Domain layer (`internal/domain/...`)

- Prefer **sentinel** or **typed** domain errors for validation and business rules (e.g. `url.ErrInvalidShortCode`).
- The domain should not generally expose low-level implementation error *identity* (DB driver errors, `net/url` parse errors, etc.).
- If helpful for debugging, domain code may include low-level error *text* while preserving the domain sentinel identity.
  - Example policy: `fmt.Errorf("%w: %v", url.ErrInvalidOriginalURL, err)` preserves `url.ErrInvalidOriginalURL` but does not allow `errors.Is(err, <parseErr>)`.

## Application & infrastructure layers (`internal/application`, `internal/infrastructure/...`)

- When adding context, wrap the underlying error with `%w`:

```go
return nil, fmt.Errorf("failed to create shortened URL: %w", err)
```

This keeps the underlying identity so upstream layers/tests can use `errors.Is/As`.

## Creating new errors

- Use `errors.New("literal")` for constant, non-formatted errors.
- Use `fmt.Errorf` only when formatting is required and/or when wrapping with `%w`.

## Testing guidance

- Prefer `errors.Is(err, want)` (or `errors.As`) over string matching.
- String matching is reserved for cases where there is no stable sentinel/typed error to assert on (e.g. human-facing log or HTTP response bodies).
