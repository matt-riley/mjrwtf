# Security Policy

## Supported Versions

We take security seriously and actively maintain security updates for the following versions:

| Version | Supported          | Status                    |
| ------- | ------------------ | ------------------------- |
| latest  | :white_check_mark: | Active development        |
| < 1.0   | :x:                | Pre-release, not for production |

**Note:** This project is currently in active development. Once version 1.0 is released, we will provide security support for the current major version and one previous major version.

## Security Features

mjr.wtf implements multiple security controls to protect users and the system:

### Input Validation
- **URL Validation**: All URLs are validated and sanitized before shortening
- **Short Code Validation**: Strict alphanumeric validation (3-20 characters, `[a-zA-Z0-9_-]`)
- **Scheme Allowlist**: Only `http://` and `https://` schemes are permitted

### Authentication & Authorization
- **Bearer Token Authentication**: API endpoints require valid authentication tokens.
  - Configure tokens via `AUTH_TOKENS` (preferred; comma-separated) or `AUTH_TOKEN` (legacy; used only if `AUTH_TOKENS` is unset).
  - Avoid passing tokens via CLI arguments or logging them.
- **Ownership Verification**: Deleting URLs and viewing analytics is restricted to the creator (`created_by`).
- **Prometheus metrics auth (optional)**: `/metrics` is public by default; set `METRICS_AUTH_ENABLED=true` to require Bearer auth (uses the same tokens as the API).
- **Rate limiting**: Rate limiting applies to `/{shortCode}` and `/api/*`; configure via `REDIRECT_RATE_LIMIT_PER_MINUTE` (default: 120) and `API_RATE_LIMIT_PER_MINUTE` (default: 60).
- **Constant-Time Comparison**: Token validation uses constant-time comparison to prevent timing attacks.

### Database Security
- **SQL Injection Prevention**: All database queries use parameterized statements via sqlc
- **Prepared Statements**: Type-safe, compiled SQL queries prevent injection attacks
- **Input Sanitization**: All user inputs are validated before database operations

### Logging & Monitoring
- **Structured Logging**: All requests and errors are logged using zerolog
- **Request Tracing**: Request IDs for tracking and debugging

### Secrets Management
- **Environment Variables**: All secrets stored in environment variables
- **No Hardcoded Credentials**: Code is free of hardcoded secrets
- **`.env` Support**: Local development uses `.env` files (never committed)

## Reporting a Vulnerability

We appreciate responsible disclosure of security vulnerabilities. If you discover a security issue, please follow these steps:

### How to Report

**DO NOT** open a public GitHub issue for security vulnerabilities.

Instead, please report security vulnerabilities through one of these channels:

1. **GitHub Security Advisories** (Preferred)
   - Navigate to the repository's Security tab
   - Click "Report a vulnerability"
   - Fill out the private vulnerability report form
   - This creates a private discussion with maintainers

2. **Email**
   - Send details to: security@mjr.wtf (if configured)
   - Use PGP encryption if possible (key available on request)
   - Include "SECURITY" in the subject line

### What to Include

Please provide as much information as possible:

- **Vulnerability Type**: SQL injection, XSS, authentication bypass, etc.
- **Affected Component**: URL shortening, redirects, API endpoints, etc.
- **Attack Vector**: How the vulnerability can be exploited
- **Impact Assessment**: What data/systems are at risk
- **Proof of Concept**: Steps to reproduce (code, curl commands, etc.)
- **Suggested Fix**: If you have recommendations (optional)
- **Disclosure Timeline**: When you plan to publicly disclose (if applicable)

### Example Report Format

```
**Title:** SQL Injection in URL creation endpoint

**Severity:** High

**Description:**
The /api/urls endpoint is vulnerable to SQL injection through the 
'short_code' parameter when certain special characters are used.

**Steps to Reproduce:**
1. Send POST request to /api/urls
2. Include payload: {"short_code": "test'; DROP TABLE urls--", "url": "https://example.com"}
3. Observe database error indicating successful injection

**Impact:**
An authenticated attacker could execute arbitrary SQL commands, 
potentially accessing or modifying all data in the database.

**Suggested Fix:**
Ensure all database queries use parameterized statements. 
The current implementation in url_repository.go line 45 uses 
string concatenation.

**Discovery Date:** 2024-01-15
```

## Response Timeline

We are committed to addressing security vulnerabilities promptly:

| Severity | Response Time | Fix Timeline | Disclosure Timeline |
|----------|--------------|--------------|---------------------|
| **Critical** | 24 hours | 7 days | 30 days after fix |
| **High** | 48 hours | 14 days | 60 days after fix |
| **Medium** | 5 days | 30 days | 90 days after fix |
| **Low** | 10 days | 60 days | 90 days after fix |

### Severity Definitions

**Critical**
- Remote code execution (RCE)
- Authentication bypass allowing admin access
- SQL injection with data exfiltration
- Exposure of all user data or credentials
- Complete system compromise

**High**
- Privilege escalation
- Significant data exposure (partial user data)
- SQL injection with limited impact
- Bypass of security controls
- Denial of Service affecting all users

**Medium**
- Information disclosure (non-sensitive)
- Cross-Site Scripting (XSS) with limited impact
- Cross-Site Request Forgery (CSRF)
- Open redirect vulnerabilities
- Security misconfiguration

**Low**
- Information leakage (minimal impact)
- Missing security headers
- Verbose error messages
- Non-exploitable edge cases

## CodeQL Security Scanning

This project uses GitHub's CodeQL for continuous security analysis.

### Automated Scanning

CodeQL runs automatically on:
- **Every Pull Request**: All PRs are scanned before merge
- **Every Push to Main**: Production branch is continuously monitored
- **Weekly Schedule**: Monday at 6:00 UTC for dependency vulnerability detection

### Query Suites

We use the `security-extended` query suite, which includes:
- All default security queries
- Additional security checks beyond standard SAST
- Queries for common vulnerability patterns (OWASP Top 10)

### CodeQL Triage Process

#### 1. Alert Generation
When CodeQL identifies a potential vulnerability:
- Alert appears in repository's Security tab → Code Scanning Alerts
- GitHub creates a notification for repository administrators
- Alert includes severity, category, and affected code location

#### 2. Initial Triage (Within 48 hours)
Security team reviews the alert:
- **Severity Assessment**: Confirm or adjust CodeQL's severity rating
- **False Positive Check**: Determine if it's a genuine vulnerability
- **Impact Analysis**: Assess real-world exploitability
- **Priority Assignment**: Based on severity and exploitability

#### 3. Remediation
- **Critical/High**: Assigned immediately, hotfix if needed
- **Medium**: Included in next sprint/release
- **Low**: Backlog for future address
- **False Positive**: Dismissed with justification

#### 4. Verification
After fix is implemented:
- Re-run CodeQL scan to confirm resolution
- Add regression test to prevent reintroduction
- Update security documentation if needed

#### 5. Documentation
All security findings are documented:
- Fix committed with security context
- Internal security log updated
- Public disclosure (if warranted) after fix deployment

### Dismissing False Positives

If a CodeQL alert is a false positive:
1. Document why it's not exploitable
2. Add code comment explaining the safety
3. Dismiss in GitHub with detailed reason
4. Consider adding suppression comment if pattern is intentional

Example suppression comment:
```go
// CodeQL[go/sql-injection] - False positive: parameter is validated via regex before use
```

### CodeQL Query Customization

We may add custom CodeQL queries for:
- Project-specific security patterns
- Business logic vulnerabilities
- Custom validation rules

Custom queries are located in `.github/codeql/` (if present).

## Security Best Practices for Contributors

### Before Submitting Code

**Security Checklist:**
- [ ] All user inputs are validated in the domain layer
- [ ] No hardcoded credentials, API keys, or secrets
- [ ] Database queries use parameterized statements (sqlc)
- [ ] Error messages don't leak sensitive information
- [ ] Authentication/authorization checks are in place
- [ ] Logging doesn't expose sensitive data (passwords, tokens, PII)
- [ ] Dependencies are up-to-date (no known vulnerabilities)
- [ ] Input validation uses allowlists, not denylists

### Secure Coding Patterns

#### Input Validation (Domain Layer)
```go
func (u *URL) Validate() error {
    // Always validate at domain layer
    if !isValidShortCode(u.ShortCode) {
        return ErrInvalidShortCode
    }
    
    // Validate and sanitize URL
    parsed, err := url.Parse(u.OriginalURL)
    if err != nil {
        return ErrInvalidURL
    }
    
    // Prevent open redirects
    if !isAllowedScheme(parsed.Scheme) {
        return ErrDisallowedScheme
    }
    
    return nil
}
```

#### Authentication
```go
// Use constant-time comparison
func validateToken(provided, expected string) bool {
    return subtle.ConstantTimeCompare(
        []byte(provided), 
        []byte(expected),
    ) == 1
}
```

#### Authorization
```go
// Always verify ownership
func (s *URLService) Delete(ctx context.Context, shortCode, userID string) error {
    url, err := s.repo.FindByShortCode(ctx, shortCode)
    if err != nil {
        return err
    }
    
    if url.CreatedBy != userID {
        return ErrUnauthorized
    }
    
    return s.repo.Delete(ctx, shortCode)
}
```

### Dependency Security

- **Regular Updates**: Dependencies are updated weekly via Renovate
- **Vulnerability Scanning**: `go mod verify` checks module integrity
- **Version Pinning**: Production uses exact versions, not ranges
- **Minimal Dependencies**: Only use necessary dependencies

### Secrets Management

**Development:**
- Use `.env` files (never commit them)
- `.env.example` provides template without secrets
- Git ignores all `.env` files

**Production:**
- Use environment variables or secrets management service
- Rotate tokens/keys regularly
- Never log secrets or include in error messages

### Threat Model

**Primary Threats:**
1. **Malicious URL Distribution**: Attacker shortens phishing/malware URLs
2. **Open Redirect**: Redirect to attacker-controlled sites
3. **API Abuse**: Mass creation of short URLs for spam
4. **Information Disclosure**: Unauthorized access to analytics
5. **SQL Injection**: Database compromise via input fields
6. **Cross-Site Scripting**: Script injection via user-generated content

**Current Mitigations:**
- URL validation and scheme allowlisting
- Authentication (Bearer token)
- Parameterized queries (sqlc)
- Authorization checks on all operations

**Planned/Future Mitigations:**
- Private IP protection (SSRF prevention)
- Rate limiting for API endpoints
- Security headers (CSP, X-Frame-Options, etc.)
- Output encoding for user-generated content

## Incident Response

If a security vulnerability is exploited:

1. **Containment**: Immediately disable affected functionality if possible
2. **Assessment**: Determine scope of breach and affected users
3. **Notification**: Inform affected users and stakeholders
4. **Remediation**: Deploy fix to production
5. **Post-Mortem**: Document incident and improve processes
6. **Disclosure**: Public disclosure after fix deployment (if warranted)

## Security Contact

For security-related questions or concerns:
- **Security Issues**: Use GitHub Security Advisories or email security@mjr.wtf
- **General Questions**: Open a GitHub Discussion (for non-sensitive topics)
- **Urgent Issues**: Mark report as "Critical" for immediate attention

## Acknowledgments

We appreciate security researchers who responsibly disclose vulnerabilities. With your permission, we will acknowledge your contribution in:
- Release notes for the security fix
- Security hall of fame (if applicable)
- CVE credits (for serious vulnerabilities)

Thank you for helping keep mjr.wtf secure!

---

**Last Updated**: December 2024  
**Policy Version**: 1.0
