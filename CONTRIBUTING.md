# Contributing to mjr.wtf

Thank you for your interest in contributing to mjr.wtf! This guide will help you get started.

## Development Workflow

### 1. Fork and Clone

```bash
git clone <your-fork-url>
cd mjrwtf
```

### 2. Setup Environment

```bash
# Copy environment configuration
cp .env.example .env

# Generate database code (required for build/test)
sqlc generate
# (Optional) Generate templates if using templ
templ generate
# Build migration tool
make build-migrate

# Run tests to verify setup
make test
```

### 3. Create Feature Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/bug-description
```

### 4. Make Changes

Follow these guidelines when making changes:

- **Hexagonal Architecture**: Keep domain logic in `internal/domain/`, implementations in `internal/adapters/`
- **Run sqlc generate**: After modifying SQL queries in `sqlc/*/queries.sql`
- **Run templ generate**: After modifying templates in `internal/adapters/http/templates/`
- **Run make check**: Before committing (`sqlc generate && templ generate && make check`)
- **Write tests**: All new features and bug fixes require tests

### 5. Commit Your Changes

Use conventional commit format:

```
feat: add URL expiration feature
fix: correct click count calculation
docs: update API documentation
test: add tests for URL validation
refactor: simplify repository error handling
```

### 6. Create Pull Request

- Fill out the PR template completely
- Reference related issues (e.g., "Closes #123")
- Ensure all tests pass
- Request review from maintainers

## Code Quality Standards

### Tests Required

- All new features must have unit tests
- All bug fixes must have regression tests
- Aim for >80% test coverage
- Test both happy path and error cases
- Repository tests must cover both SQLite and PostgreSQL (if applicable)

### Linting

- Run `make lint` before committing
- Fix all linter warnings (except known false positives)
- Known false positive: "undefined: postgresrepo/sqliterepo" - ignore if tests pass

### Documentation

- Update README.md for user-facing changes
- Update inline comments for complex logic
- Add GoDoc comments for exported functions
- Update schema documentation in `docs/` for database changes

### Commit Messages

Use conventional commits format:
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation only
- `test:` - Adding or updating tests
- `refactor:` - Code change that neither fixes a bug nor adds a feature
- `chore:` - Maintenance tasks

## For GitHub Copilot

This project is optimized for GitHub Copilot coding agent assistance.

### Automated PR Generation

When creating issues for Copilot to work on:

1. **Use issue templates** (`.github/ISSUE_TEMPLATE/`)
2. **Provide clear acceptance criteria** using Given-When-Then format
3. **List files that need modification**
4. **Specify test requirements**
5. **Note any security implications**

### What Copilot Can Do

✅ **Excellent for:**
- Implementing new domain entities with validation
- Adding new repository methods with tests
- Creating database migrations (both SQLite and PostgreSQL)
- Writing sqlc queries
- Adding test cases to existing test suites
- Fixing bugs with clear reproduction steps
- Updating documentation

⚠️ **Needs Human Guidance:**
- Complex architectural changes spanning multiple layers
- Performance optimization requiring profiling
- Security-critical code (always requires human review)
- Major refactoring (break into smaller issues)
- Production deployment decisions

### What Requires Human Review

Even when Copilot generates the PR, **always review**:

- [ ] Security implications of changes
- [ ] Performance impact of database queries
- [ ] Breaking changes to public APIs
- [ ] Migration safety (both up and down)
- [ ] Test coverage adequacy
- [ ] Documentation completeness

## Architecture Guidelines

### Hexagonal Architecture (Ports & Adapters)

**Domain Layer** (`internal/domain/`):
- Pure business logic, no external dependencies
- Define entities with validation
- Define repository interfaces (ports)
- Never import database, HTTP, or infrastructure packages

**Adapter Layer** (`internal/adapters/`):
- Implement repository interfaces
- Use sqlc-generated code for database access
- Map between domain entities and database models
- Handle database-specific error mapping

**Infrastructure Layer** (`internal/infrastructure/`):
- Configuration management
- Logging setup
- Database connection management
- Application initialization

### Critical Workflows

#### After Changing SQL Queries

```bash
sqlc generate          # Regenerate database code
go build ./...         # Verify compilation
make test              # Run all tests
```

#### After Creating Migration

```bash
make build-migrate     # Rebuild migrate tool (migrations are embedded)
make migrate-up        # Apply migration
make migrate-status    # Verify migration applied
make migrate-down      # Test rollback
make migrate-up        # Re-apply for final state
```

#### Before Committing

```bash
make fmt               # Format code
sqlc generate          # Ensure generated code is up-to-date
make check             # Run fmt, vet, lint, test
```

### Layer Boundaries

Dependencies flow inward toward the domain:

```
Infrastructure → Adapters → Domain
     ↓              ↓          ↑
  Config        Repositories  Entities
  Logging       (impl)        Interfaces
  Database                    Validation
```

**Rules:**
- Domain NEVER imports from adapters or infrastructure
- Adapters import domain interfaces
- Infrastructure wires everything together

## Getting Help

- **Read first**: `.github/copilot-instructions.md` has comprehensive project documentation
- **Check existing issues**: Someone may have already asked
- **Check existing PRs**: See how similar changes were implemented
- **Ask in discussions**: For questions before opening issues
- **Tag maintainers**: For urgent issues or clarifications

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow
- Assume good intentions

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

---

Thank you for contributing to mjr.wtf! 🎉
