# GitHub Copilot Workspace Guide for mjr.wtf

This guide helps you work effectively with GitHub Copilot to generate high-quality PRs for the mjr.wtf URL shortener project.

## Quick Reference for Copilot PR Generation

### Before Creating a PR

1. **Ensure issue is well-scoped** - Use issue templates in `.github/ISSUE_TEMPLATE/`
2. **Verify environment** - Run: `sqlc generate && make check`
3. **Review custom agents** - Select appropriate agent(s) for the task (see Agent Selection Guide below)

### After PR is Generated

1. **Review all changes** - Don't blindly merge Copilot PRs
2. **Run tests locally** - Verify `make test` passes
3. **Check security implications** - Review authentication, validation, data handling
4. **Verify migrations** - Test both `up` and `down` if database changes
5. **Review test coverage** - Ensure new code has adequate tests

### Recommended Workflow (Copilot CLI)

For complex work in **Copilot CLI**, use **Research → Plan → Implement → Validate** with `/context` (stay <~60%), `/clear` between phases, and `/share` artifacts into `thoughts/`.

Tip: run `/skills reload` then `/skills info rpi-workflow` for copy/paste prompt templates.

## What Copilot Does Well For This Project

### ✅ Excellent Performance

Copilot excels at these tasks for mjr.wtf:

- **Implementing new domain entities** with proper validation
- **Adding new repository methods** with both SQLite and PostgreSQL support
- **Creating database migrations** in both SQLite and PostgreSQL syntax
- **Writing sqlc queries** following proper naming conventions
- **Adding test cases** to existing test suites using table-driven patterns
- **Fixing bugs** when given clear reproduction steps
- **Updating documentation** to reflect code changes

### ⚠️ Needs Guidance

Copilot needs more human guidance for:

- **Complex architectural changes** spanning multiple layers (break into smaller issues)
- **Performance optimization** requiring profiling and benchmarking
- **Security-critical code** (authentication, authorization, data protection)
- **Major refactoring** (provide detailed step-by-step instructions)
- **Database performance tuning** (index design, query optimization)

## How to Write Issues for Copilot

### Good Issue Example

```markdown
Title: Add URL expiration feature

## User Story
As a URL creator
I want to set an expiration date on short URLs
So that links automatically become invalid after a certain time

## Acceptance Criteria
- [ ] Given a URL with expiration date set, when the date passes, then accessing the URL returns 404
- [ ] Given an expired URL, when checking its status via API, then status shows "expired"
- [ ] URL expiration is optional (null/zero value means never expires)
- [ ] Expiration date must be in the future when creating/updating URL

## Files to Modify
- `internal/domain/url/url.go` - Add ExpiresAt field and validation
- `internal/migrations/sqlite/XXXXX_add_expiration.sql` - Add expires_at column (SQLite)
- `internal/migrations/postgres/XXXXX_add_expiration.sql` - Add expires_at column (PostgreSQL)
- `internal/adapters/repository/sqlc/sqlite/queries.sql` - Add expiration check to GetURLByShortCode
- `internal/adapters/repository/sqlc/postgres/queries.sql` - Add expiration check to GetURLByShortCode
- Tests for validation and repository operations

## Implementation Steps
1. Add ExpiresAt *time.Time field to URL entity
2. Add validation: if ExpiresAt is set, must be in future
3. Create migration to add expires_at column (nullable timestamp)
4. Update GetURLByShortCode query to check if URL is expired
5. Add test cases for expired and non-expired URLs
6. Update error handling to return ErrURLExpired
```

### Bad Issue Example

```markdown
Title: Make URLs better

We need to improve URLs somehow. Maybe add features?
```

**Why it's bad:**
- No clear acceptance criteria
- No specific files or implementation guidance
- Vague requirements
- No test expectations

## Agent Selection Guide

Use this table to select the appropriate custom agent(s) for different tasks:

| Task Type | Recommended Agent(s) | Why |
|-----------|---------------------|-----|
| New domain entity | `golang-expert` | Knows hexagonal architecture patterns |
| Repository implementation | `golang-expert` + `sqlc-query-specialist` | Database + Go expertise |
| Database migration | `migration-specialist` | Schema design and migration safety |
| SQL query optimization | `sqlc-query-specialist` | Query performance and sqlc patterns |
| Test coverage improvement | `test-specialist` | Testing patterns and coverage analysis |
| API endpoint | `golang-expert` + `api-designer` | HTTP handlers and API design |
| Security review | `security-expert` | Security best practices |
| Documentation | `documentation-writer` | Clear technical writing |
| Issue breakdown | `business-analyst` | Requirements and scoping |

### Selecting Multiple Agents

For complex tasks, you can select multiple agents. They'll collaborate on the PR:

- **Domain entity + repository**: `golang-expert` + `sqlc-query-specialist`
- **API with database**: `api-designer` + `golang-expert` + `sqlc-query-specialist`
- **Feature with security concerns**: `golang-expert` + `security-expert`

## Pre-PR Checklist

Before asking Copilot to generate a PR, ensure:

- [ ] Issue has clear acceptance criteria (Given-When-Then format)
- [ ] Files likely to change are listed in the issue
- [ ] Database changes (schema, queries, migrations) are identified
- [ ] Test requirements are specified (unit, integration, both databases)
- [ ] Breaking changes are noted (API changes, migration requirements)
- [ ] Security implications are considered and documented
- [ ] Dependencies on other issues are noted

## Post-PR Review Checklist

After Copilot generates a PR, review:

### Functionality
- [ ] All acceptance criteria met
- [ ] Code follows hexagonal architecture patterns
- [ ] Domain logic stays in domain layer (no external dependencies)
- [ ] Repository implementations in adapter layer

### Testing
- [ ] Tests added for new functionality
- [ ] Both positive and negative test cases included
- [ ] Repository tests cover both SQLite and PostgreSQL
- [ ] All tests pass (`make test`)
- [ ] Test coverage is adequate (>80% for new code)

### Database
- [ ] sqlc code regenerated if queries changed (`sqlc generate`)
- [ ] Migrations created for both SQLite and PostgreSQL
- [ ] Migration UP tested successfully
- [ ] Migration DOWN tested successfully (rollback works)
- [ ] Migrations are idempotent where possible

### Code Quality
- [ ] Code formatted (`make fmt`)
- [ ] Linter issues addressed (`make lint`, ignore known false positives)
- [ ] GoDoc comments added for exported functions
- [ ] No hardcoded credentials or secrets
- [ ] Error messages are helpful but don't leak sensitive data

### Documentation
- [ ] README.md updated if user-facing changes
- [ ] Comments added for non-obvious logic
- [ ] Schema documentation updated (`docs/schema.*.sql`)
- [ ] API documentation updated if endpoints changed

### Security
- [ ] User inputs validated in domain layer
- [ ] Authentication/authorization implemented for write operations
- [ ] SQL injection not possible (sqlc handles this)
- [ ] Sensitive data not logged
- [ ] Rate limiting considered for public endpoints

### Performance
- [ ] Database queries are efficient
- [ ] Appropriate indexes exist for new queries
- [ ] No N+1 query problems
- [ ] Large data sets handled appropriately

## Common Copilot Pitfalls

### Issue: Copilot Forgets to Run sqlc generate

**Symptom**: PR shows compilation errors "undefined: postgresrepo"

**Prevention**: Add to issue description:
```markdown
## Important
After modifying queries.sql files, you MUST run `sqlc generate` before building/testing.
```

### Issue: Migration Only Created for One Database

**Symptom**: PR has SQLite migration but no PostgreSQL migration

**Prevention**: Explicitly list both in "Files to Modify":
```markdown
- `internal/migrations/sqlite/XXXXX_feature.sql`
- `internal/migrations/postgres/XXXXX_feature.sql`
```

### Issue: Tests Only for One Database

**Symptom**: PR has `*_sqlite_test.go` but no `*_postgres_test.go`

**Prevention**: In acceptance criteria, add:
```markdown
- [ ] Repository tests pass for both SQLite and PostgreSQL
```

### Issue: Domain Layer Imports External Dependencies

**Symptom**: Domain code imports `database/sql` or HTTP packages

**Prevention**: In issue, emphasize:
```markdown
## Architecture Requirements
Domain layer (`internal/domain/`) must have NO external dependencies.
Only define entities, validation, and repository interfaces.
```

### Issue: Missing Error Handling

**Symptom**: Error cases not properly handled or tested

**Prevention**: In acceptance criteria, include negative cases:
```markdown
- [ ] Given invalid input, when creating URL, then appropriate error returned
- [ ] Given database error, when saving URL, then error properly mapped to domain error
```

## Tips for Writing Better Issues

1. **Use templates** - Start with `.github/ISSUE_TEMPLATE/` templates
2. **Be specific** - List exact files, functions, and acceptance criteria
3. **Include examples** - Show expected input/output or API requests/responses
4. **Break down large tasks** - Create separate issues for each logical unit
5. **Specify tests** - Describe what tests should be added
6. **Note gotchas** - Call out project-specific patterns or requirements
7. **Link related issues** - Reference dependencies or related work

## Example: Feature Implementation Flow

### 1. Create Well-Scoped Issue

Use feature request template with:
- Clear user story
- Specific acceptance criteria
- List of files to modify
- Implementation steps
- Test requirements

### 2. Select Appropriate Agent(s)

For a new feature with database changes:
- Primary: `golang-expert`
- Secondary: `sqlc-query-specialist`
- Secondary: `migration-specialist`

### 3. Copilot Generates PR

Copilot will:
- Read project instructions
- Run setup steps (sqlc generate, build, test)
- Implement changes following architecture patterns
- Generate tests
- Update documentation

### 4. Human Review

Review checklist above, focusing on:
- Security implications
- Test adequacy
- Migration safety
- Architecture compliance

### 5. Iterate if Needed

Request changes via PR comments. Copilot can address feedback iteratively.

### 6. Merge

After approval and passing CI, merge the PR.

## Resources

- **Project documentation**: `.github/copilot-instructions.md` - Comprehensive project guide
- **Contributing guide**: `CONTRIBUTING.md` - Development workflow
- **Domain documentation**: `internal/domain/README.md` - Domain layer patterns
- **Schema documentation**: `docs/README.md` - Database schema details

---

**Remember**: Copilot is a powerful assistant, but human judgment is essential for architecture, security, and quality decisions. Use Copilot to accelerate development, not replace thoughtful engineering.
