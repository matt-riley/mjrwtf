# Domain Layer

This directory contains the core domain models and business logic for the mjr.wtf URL shortener, following hexagonal architecture principles.

## Overview

The domain layer is the heart of the application, containing:
- **Domain entities**: Core business objects (URL, Click)
- **Repository interfaces (Ports)**: Contracts for data persistence
- **Domain validation logic**: Business rules and invariants
- **Domain errors**: Specific error types for business rule violations

## Domain Models

### URL (`internal/domain/url/`)

Represents a shortened URL in the system.

**Fields:**
- `ID`: Unique identifier (int64)
- `ShortCode`: The short identifier used in the shortened URL (e.g., "abc123" in https://mjr.wtf/abc123)
- `OriginalURL`: The destination URL
- `CreatedAt`: Timestamp when the URL was created
- `CreatedBy`: Identifier of the user/system that created the URL

**Domain Rules:**

1. **Short Code Format**:
   - Must be 3-20 characters long
   - Can only contain alphanumeric characters, underscores, and hyphens
   - Must be unique across all URLs
   - Cannot be empty

2. **Original URL Format**:
   - Must be a valid URL with proper format
   - Must have a scheme (http or https only)
   - Must have a host
   - Cannot be empty

3. **Created By**:
   - Cannot be empty or whitespace-only
   - Represents the creator (user ID, API key identifier, or system name)

**Validation:**
- Use `NewURL()` to create a new URL with automatic validation
- Use `ValidateShortCode()` for standalone short code validation
- Use `ValidateOriginalURL()` for standalone URL validation

### Click (`internal/domain/click/`)

Represents an analytics click event on a shortened URL.

**Fields:**
- `ID`: Unique identifier (int64)
- `URLID`: Foreign key reference to the URL that was clicked
- `ClickedAt`: Timestamp when the click occurred
- `Referrer`: HTTP Referer header (optional)
- `Country`: ISO 3166-1 alpha-2 country code (optional)
- `UserAgent`: User-Agent header (optional)

**Domain Rules:**

1. **URL ID**:
   - Must be positive (> 0)
   - Must reference an existing URL

2. **Country Code**:
   - Must be empty or exactly 2 characters (ISO 3166-1 alpha-2 format)
   - Examples: "US", "GB", "CA"

3. **Optional Fields**:
   - Referrer, Country, and UserAgent are all optional
   - Empty strings are allowed for these fields

**Validation:**
- Use `NewClick()` to create a new click with automatic validation

## Repository Interfaces (Ports)

Following hexagonal architecture, repository interfaces are defined in the domain layer but implemented in the adapters layer.

### URLRepository (`internal/domain/url/repository.go`)

**Methods:**

- `Create(url *URL) error`: Creates a new shortened URL
  - Returns `ErrDuplicateShortCode` if the short code already exists

- `FindByShortCode(shortCode string) (*URL, error)`: Retrieves a URL by its short code
  - Returns `ErrURLNotFound` if the URL doesn't exist

- `Delete(shortCode string) error`: Removes a URL by its short code
  - Returns `ErrURLNotFound` if the URL doesn't exist

- `List(createdBy string, limit, offset int) ([]*URL, error)`: Retrieves URLs with filtering and pagination
  - `createdBy`: filter by creator (empty string means no filter)
  - `limit`: maximum number of results (0 means no limit)
  - `offset`: number of results to skip

- `ListByCreatedByAndTimeRange(createdBy string, startTime, endTime time.Time) ([]*URL, error)`: Retrieves URLs by creator within a time range

### ClickRepository (`internal/domain/click/repository.go`)

**Methods:**

- `Record(click *Click) error`: Records a new click event

- `GetStatsByURL(urlID int64) (*Stats, error)`: Retrieves aggregate statistics for a URL
  - Returns total count, breakdown by country, referrer, and date

- `GetStatsByURLAndTimeRange(urlID int64, startTime, endTime time.Time) (*TimeRangeStats, error)`: Retrieves statistics for a time range

- `GetTotalClickCount(urlID int64) (int64, error)`: Returns total click count for a URL

- `GetClicksByCountry(urlID int64) (map[string]int64, error)`: Returns clicks grouped by country

## Domain Errors

### URL Errors (`internal/domain/url/errors.go`)

- `ErrURLNotFound`: URL not found by short code
- `ErrDuplicateShortCode`: Short code already exists
- `ErrEmptyShortCode`: Short code is empty
- `ErrInvalidShortCode`: Short code format is invalid
- `ErrEmptyOriginalURL`: Original URL is empty
- `ErrInvalidOriginalURL`: Original URL format is invalid
- `ErrMissingURLScheme`: URL doesn't have a scheme
- `ErrInvalidURLScheme`: URL has unsupported scheme (not http/https)
- `ErrMissingURLHost`: URL doesn't have a host
- `ErrInvalidCreatedBy`: Created by is empty

### Click Errors (`internal/domain/click/errors.go`)

- `ErrInvalidURLID`: URL ID is zero or negative
- `ErrInvalidCountryCode`: Country code is not exactly 2 characters
- `ErrClickNotFound`: Click not found

## Usage Examples

### Creating a URL

```go
import "github.com/matt-riley/mjrwtf/internal/domain/url"

// Create a new URL with validation
u, err := url.NewURL("abc123", "https://example.com", "user123")
if err != nil {
    // Handle validation error
    switch err {
    case url.ErrInvalidShortCode:
        // Handle invalid short code
    case url.ErrInvalidOriginalURL:
        // Handle invalid URL
    }
}

// URL is now ready to be persisted via repository
err = urlRepo.Create(u)
if err == url.ErrDuplicateShortCode {
    // Handle duplicate short code
}
```

### Recording a Click

```go
import "github.com/matt-riley/mjrwtf/internal/domain/click"

// Create a new click event
c, err := click.NewClick(urlID, "https://google.com", "US", "Mozilla/5.0")
if err != nil {
    // Handle validation error
}

// Record the click
err = clickRepo.Record(c)
if err != nil {
    // Handle error
}
```

### Validating Data

```go
import "github.com/matt-riley/mjrwtf/internal/domain/url"

// Validate a short code before use
if err := url.ValidateShortCode("abc123"); err != nil {
    // Handle invalid short code
}

// Validate a URL before use
if err := url.ValidateOriginalURL("https://example.com"); err != nil {
    // Handle invalid URL
}
```

## Design Principles

1. **Hexagonal Architecture**: Domain layer is independent of infrastructure concerns
2. **Repository Pattern**: Interfaces defined in domain, implementations in adapters
3. **Domain-Driven Design**: Rich domain models with behavior and validation
4. **Explicit Error Types**: Clear, specific errors for different validation failures
5. **Immutability**: Domain entities should be treated as immutable after creation
6. **Single Responsibility**: Each domain model has a clear, focused purpose

## Testing

All domain models include comprehensive unit tests:
- `internal/domain/url/url_test.go`: Tests for URL validation and creation
- `internal/domain/click/click_test.go`: Tests for Click validation and creation

Run tests:
```bash
go test ./internal/domain/...
```

## Implementation Notes

1. **Value Objects**: Short codes could be extracted into value objects if additional behavior is needed
2. **Aggregate Roots**: URL is the aggregate root for the URL-Click relationship
3. **Domain Events**: Future enhancement could add domain events for URL creation/clicks
4. **Time Handling**: All timestamps use `time.Time` for consistency
5. **Validation**: Validation happens at construction time via `New*()` functions
