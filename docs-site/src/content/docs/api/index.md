---
title: API
description: OpenAPI spec, endpoints, authentication, and examples.
---


This document provides information about the mjr.wtf URL shortener API.

## OpenAPI Specification

The complete API is documented using OpenAPI 3.0. The specification file is located at [`openapi.yaml`](https://github.com/matt-riley/mjrwtf/blob/main/openapi.yaml) in the repository root.

### Viewing the Documentation

**Interactive Documentation:**
- [SwaggerUI](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/matt-riley/mjrwtf/main/openapi.yaml) - Interactive API documentation with "Try it out" feature
- [ReDoc](https://redocly.github.io/redoc/?url=https://raw.githubusercontent.com/matt-riley/mjrwtf/main/openapi.yaml) - Clean, responsive API reference documentation

**Local Validation:**
```bash
# Using Make
make validate-openapi

# Using swagger-cli directly
npm install -g @apidevtools/swagger-cli
swagger-cli validate openapi.yaml
```

## API Overview

### Base URLs

- **Production:** `https://mjr.wtf`
- **Local Development:** `http://localhost:8080`

### Authentication

Most endpoints require Bearer token authentication:

```bash
Authorization: Bearer YOUR_TOKEN_HERE
```

Configure your token via `AUTH_TOKENS` (preferred, comma-separated) or `AUTH_TOKEN` (legacy). See the Authentication section in the main README for details.

### Content Type

All API requests and responses use `application/json` content type unless otherwise specified.

## Endpoints

### URL Management

#### Create Shortened URL

**POST** `/api/urls`

Creates a new shortened URL.

**Authentication:** Required

**Request Body:**
```json
{
  "original_url": "https://example.com/very/long/url/path"
}
```

**Response (201 Created):**
```json
{
  "short_code": "abc123",
  "short_url": "https://mjr.wtf/abc123",
  "original_url": "https://example.com/very/long/url/path"
}
```

**Example:**
```bash
curl -X POST https://mjr.wtf/api/urls \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://example.com"}'
```

---

#### List URLs

**GET** `/api/urls`

Retrieves a paginated list of URLs for the current auth identity.

**Note:** mjr.wtf currently maps all valid tokens to a single shared identity (`created_by: "authenticated-user"`), so this behaves like a single-tenant list.

**Authentication:** Required

**Query Parameters:**
- `limit` (optional): Maximum number of URLs to return (0-100; values <= 0 use the default: 20)
- `offset` (optional): Number of URLs to skip for pagination (default: 0)

**Response (200 OK):**
```json
{
  "urls": [
    {
      "id": 1,
      "short_code": "abc123",
      "original_url": "https://example.com",
      "created_at": "2025-12-25T10:00:00Z",
      "created_by": "authenticated-user",
      "click_count": 42
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

**Example:**
```bash
curl https://mjr.wtf/api/urls?limit=10&offset=0 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

#### Delete URL

**DELETE** `/api/urls/{shortCode}`

Deletes a shortened URL. Requires authentication (multi-user/ownership is not implemented yet).

**Authentication:** Required

**Path Parameters:**
- `shortCode`: The short code to delete (e.g., "abc123")

**Response (204 No Content):**
No response body.

**Example:**
```bash
curl -X DELETE https://mjr.wtf/api/urls/abc123 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

### Analytics

#### Get URL Analytics

**GET** `/api/urls/{shortCode}/analytics`

Retrieves analytics data for a shortened URL including click counts, geographic distribution, and referrer information.

**Authentication:** Required (returns 403 if the auth identity does not match `created_by`)

**Note:** mjr.wtf currently maps all valid tokens to a single shared identity, so this typically behaves as "any valid token can view analytics".

**Path Parameters:**
- `shortCode`: The short code to get analytics for

**Query Parameters:**
- `start_time` (optional): Filter clicks from this time (RFC3339 format, e.g., "2025-11-20T00:00:00Z")
- `end_time` (optional): Filter clicks until this time (RFC3339 format, e.g., "2025-11-22T23:59:59Z")

**Note:** Both `start_time` and `end_time` must be provided together for time range queries. `start_time` must be strictly before `end_time`.

**Response (200 OK) - All-time statistics:**
```json
{
  "short_code": "abc123",
  "original_url": "https://example.com",
  "total_clicks": 150,
  "by_country": {
    "US": 75,
    "GB": 30,
    "DE": 25
  },
  "by_referrer": {
    "https://twitter.com": 50,
    "direct": 60
  },
  "by_date": {
    "2025-12-20": 30,
    "2025-12-21": 45
  }
}
```

**Response (200 OK) - Time range statistics:**
```json
{
  "short_code": "abc123",
  "original_url": "https://example.com",
  "total_clicks": 75,
  "by_country": {
    "US": 40,
    "GB": 20
  },
  "by_referrer": {
    "https://twitter.com": 30,
    "direct": 45
  },
  "start_time": "2025-11-20T00:00:00Z",
  "end_time": "2025-11-22T23:59:59Z"
}
```

**Example - All-time:**
```bash
curl https://mjr.wtf/api/urls/abc123/analytics \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Example - Time range:**
```bash
curl "https://mjr.wtf/api/urls/abc123/analytics?start_time=2025-11-20T00:00:00Z&end_time=2025-11-22T23:59:59Z" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

### Public Endpoints

#### Redirect

**GET** `/{shortCode}`

Redirects to the original URL associated with the short code. This endpoint is public and does not require authentication.

**Path Parameters:**
- `shortCode`: The short code to redirect (e.g., "abc123")

**Response (302 Found):**
Redirects to the original URL via the `Location` header.

**Response (404 Not Found):**
Returns HTML page if short code doesn't exist.

**Example:**
```bash
curl -L https://mjr.wtf/abc123
```

---

### Health & Monitoring

#### Health Check (Liveness)

**GET** `/health`

Lightweight liveness check. Does not validate external dependencies.

#### Readiness Check

**GET** `/ready`

Readiness check that validates dependencies (currently: database connectivity).

**Authentication:** None

**Response (200 OK):**
```json
{
  "status": "ok"
}
```

**Example:**
```bash
curl https://mjr.wtf/health
```

**Readiness Response (200 OK):**
```json
{
  "status": "ready"
}
```

**Readiness Response (503 Service Unavailable):**
```json
{
  "status": "unavailable"
}
```

**Readiness Example:**
```bash
curl https://mjr.wtf/ready
```

---

#### Prometheus Metrics

**GET** `/metrics`

Returns Prometheus metrics for monitoring.

**Authentication:** Optional (configurable via `METRICS_AUTH_ENABLED` environment variable)

**Response (200 OK):**
Returns Prometheus-formatted metrics in `text/plain` format.

**Example:**
```bash
# If authentication is enabled
curl https://mjr.wtf/metrics \
  -H "Authorization: Bearer YOUR_TOKEN"

# If authentication is disabled (default)
curl https://mjr.wtf/metrics
```

## Error Responses

All error responses follow a consistent format:

```json
{
  "error": "error message here"
}
```

### HTTP Status Codes

- **200 OK** - Request succeeded
- **201 Created** - Resource successfully created
- **204 No Content** - Request succeeded with no response body
- **302 Found** - Redirect response
- **400 Bad Request** - Invalid input or request format
- **401 Unauthorized** - Missing or invalid authentication token
- **403 Forbidden** - Insufficient permissions (e.g., trying to delete another user's URL)
- **404 Not Found** - Resource not found
- **409 Conflict** - Resource already exists (e.g., duplicate short code)
- **500 Internal Server Error** - Server error

### Common Error Examples

**Missing authentication:**
```json
{
  "error": "missing authorization header"
}
```

**Invalid URL format:**
```json
{
  "error": "original URL must be a valid http or https URL"
}
```

**URL not found:**
```json
{
  "error": "URL not found"
}
```

**Unauthorized deletion:**
```json
{
  "error": "unauthorized to delete this URL"
}
```

**Invalid time range:**
```json
{
  "error": "start_time must be strictly before end_time (equality not allowed)"
}
```

## Data Models

### URL Response Object

```typescript
{
  id: number;           // Unique identifier
  short_code: string;   // Short code (3-20 alphanumeric, underscore, hyphen)
  original_url: string; // Original URL
  created_at: string;   // ISO 8601 timestamp
  created_by: string;   // User ID of creator
  click_count: number;  // Total number of clicks
}
```

### Analytics Response Object

```typescript
{
  short_code: string;                  // Short code
  original_url: string;                // Original URL
  total_clicks: number;                // Total click count
  by_country: { [country: string]: number };   // Clicks by country (ISO 3166-1 alpha-2)
  by_referrer: { [referrer: string]: number }; // Clicks by referrer URL
  by_date?: { [date: string]: number };        // Clicks by date (YYYY-MM-DD) - only for all-time stats
  start_time?: string;                 // Start time (if time range query)
  end_time?: string;                   // End time (if time range query)
}
```

## Validation Rules

### Short Code
- Length: 3-20 characters
- Allowed characters: alphanumeric, underscore, hyphen (`a-zA-Z0-9_-`)
- Pattern: `^[a-zA-Z0-9_-]{3,20}$`

### Original URL
- Must include scheme (`http://` or `https://`)
- Must have a valid host
- Must be a valid URL format

### Time Range Queries
- Both `start_time` and `end_time` must be provided together
- Times must be in RFC3339 format (e.g., `2025-11-20T00:00:00Z`)
- `start_time` must be strictly before `end_time`

## Rate Limiting

Rate limiting is implemented on the redirect endpoint and authenticated API routes. Configure via `REDIRECT_RATE_LIMIT_PER_MINUTE` (default: 120) and `API_RATE_LIMIT_PER_MINUTE` (default: 60).

## Keeping the Spec in Sync

### Automated Validation

The OpenAPI specification is automatically validated in CI:

```yaml
# .github/workflows/ci.yml
- name: Validate OpenAPI spec
  run: |
    npm install -g @apidevtools/swagger-cli
    swagger-cli validate openapi.yaml
```

### Manual Validation

Before committing changes to the OpenAPI spec:

```bash
# Validate the spec
make validate-openapi

# Run all checks (includes OpenAPI validation)
make check
```

### Process for Keeping Spec in Sync

When making API changes:

1. **Update the code** - Make changes to handlers, request/response types
2. **Update the OpenAPI spec** - Update `openapi.yaml` to reflect the changes
3. **Validate the spec** - Run `make validate-openapi` to ensure it's valid
4. **Update tests** - Ensure integration tests cover the new/changed behavior
5. **Update examples** - Update code examples in this document if needed
6. **Commit together** - Commit code changes and spec updates together

### CI Enforcement

The CI pipeline will fail if:
- The OpenAPI spec is invalid
- The spec doesn't validate against the OpenAPI 3.0 schema

This ensures the specification stays accurate and up-to-date with the implementation.

## Tools and Integrations

### Code Generation

The OpenAPI spec can be used to generate client libraries in various languages:

```bash
# Install OpenAPI Generator
npm install -g @openapitools/openapi-generator-cli

# Generate a TypeScript client
openapi-generator-cli generate -i openapi.yaml -g typescript-fetch -o clients/typescript

# Generate a Python client
openapi-generator-cli generate -i openapi.yaml -g python -o clients/python

# Generate a Go client
openapi-generator-cli generate -i openapi.yaml -g go -o clients/go
```

### API Testing

Use the OpenAPI spec for automated API testing:

```bash
# Install Dredd for API testing
npm install -g dredd

# Test API against the spec
dredd openapi.yaml http://localhost:8080
```

### Mock Server

Create a mock server from the spec:

```bash
# Install Prism
npm install -g @stoplight/prism-cli

# Run mock server
prism mock openapi.yaml
```

## Support

For issues or questions about the API:
- **GitHub Issues:** [https://github.com/matt-riley/mjrwtf/issues](https://github.com/matt-riley/mjrwtf/issues)
- **OpenAPI Spec:** [https://github.com/matt-riley/mjrwtf/blob/main/openapi.yaml](https://github.com/matt-riley/mjrwtf/blob/main/openapi.yaml)
