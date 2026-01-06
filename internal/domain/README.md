# Domain Layer

This directory contains the core domain models and business logic for the **mjr.wtf** URL shortener.

## Overview

The domain layer is the heart of the application and contains:

- **Domain entities**: core business objects (e.g. URL, Click)
- **Repository interfaces (ports)**: persistence contracts implemented by adapters (e.g. SQLite)
- **Validation logic**: business rules and invariants
- **Domain errors**: typed errors for business-rule violations

Subpackages you’ll commonly see:

- `internal/domain/url/` — shortened URL entity + validation + repository port
- `internal/domain/click/` — click analytics entity + repository port
- `internal/domain/urlstatus/` — URL status checking domain (alive/gone/etc.)
- `internal/domain/geolocation/` — geo lookup domain types

## Domain Models

### URL (`internal/domain/url/`)

Represents a shortened URL in the system.

**Fields:**

- `ID` (`int64`): database identifier
- `ShortCode` (`string`): the short identifier used in the shortened URL (e.g. `abc123` in `https://mjr.wtf/abc123`)
- `OriginalURL` (`string`): the destination URL
- `CreatedAt` (`time.Time`): timestamp when the URL was created
- `CreatedBy` (`string`): identifier of the user/system that created the URL

**Domain rules:**

1. **Short code**
   - Must be 3–20 characters long
   - Can only contain alphanumeric characters, underscores, and hyphens
   - Cannot be empty
2. **Original URL**
   - Must be parseable
   - Must have a scheme (`http` or `https` only)
   - Must have a host
   - Cannot be empty
3. **CreatedBy**
   - Cannot be empty or whitespace-only

**Validation:**

- Use `url.NewURL()` to create a new URL with automatic validation
- Use `url.ValidateShortCode()` / `url.ValidateOriginalURL()` for standalone validation

### Click (`internal/domain/click/`)

Represents an analytics click event on a shortened URL.

**Fields:**

- `ID` (`int64`): database identifier
- `URLID` (`int64`): foreign key reference to the URL that was clicked
- `ClickedAt` (`time.Time`): timestamp when the click occurred
- `Referrer` (`string`): raw HTTP `Referer` header value (optional)
- `ReferrerDomain` (`string`): derived hostname extracted from `Referrer` (optional)
- `Country` (`string`): ISO 3166-1 alpha-2 country code (optional)
- `UserAgent` (`string`): User-Agent header value (optional)

**Domain rules:**

- `URLID` must be positive (`> 0`)
- `Country` must be empty or exactly 2 characters

## Repository Interfaces (Ports)

Repository interfaces are defined in the domain layer (ports) and implemented in the adapters layer.

### URL Repository (`internal/domain/url/repository.go`)

```go
type Repository interface {
    Create(ctx context.Context, url *URL) error
    FindByShortCode(ctx context.Context, shortCode string) (*URL, error)
    Delete(ctx context.Context, shortCode string) error
    List(ctx context.Context, createdBy string, limit, offset int) ([]*URL, error)
    ListByCreatedByAndTimeRange(ctx context.Context, createdBy string, startTime, endTime time.Time) ([]*URL, error)
    Count(ctx context.Context, createdBy string) (int, error)
}
```

### Click Repository (`internal/domain/click/repository.go`)

```go
type Repository interface {
    Record(ctx context.Context, click *Click) error
    GetStatsByURL(ctx context.Context, urlID int64) (*Stats, error)
    GetStatsByURLAndTimeRange(ctx context.Context, urlID int64, startTime, endTime time.Time) (*TimeRangeStats, error)
    GetTotalClickCount(ctx context.Context, urlID int64) (int64, error)
    GetClicksByCountry(ctx context.Context, urlID int64) (map[string]int64, error)
}
```

## Domain Errors

See:

- `internal/domain/url/errors.go`
- `internal/domain/click/errors.go`

Notable URL errors include:

- `ErrURLNotFound`
- `ErrDuplicateShortCode`
- `ErrUnauthorizedDeletion`

## Usage Examples

### Creating a URL

```go
import (
    "context"

    "github.com/matt-riley/mjrwtf/internal/domain/url"
)

ctx := context.Background()

u, err := url.NewURL("abc123", "https://example.com", "user123")
if err != nil {
    // handle validation error
}

if err := urlRepo.Create(ctx, u); err != nil {
    // handle persistence error
}
```

### Recording a Click

```go
import (
    "context"

    "github.com/matt-riley/mjrwtf/internal/domain/click"
)

ctx := context.Background()

c, err := click.NewClick(urlID, "https://google.com", "US", "Mozilla/5.0")
if err != nil {
    // handle validation error
}

if err := clickRepo.Record(ctx, c); err != nil {
    // handle persistence error
}
```

## Testing

```bash
go test ./internal/domain/...
```
