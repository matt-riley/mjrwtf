# Test File Instructions

## Naming

- File: `<name>_test.go`
- Function: `Test<Type>_<Method>_<Scenario>`

## Patterns

- Use table-driven tests where it helps.
- Use in-memory SQLite for repo/integration tests when possible.
- Prefer polling-with-deadline for async behavior (avoid fixed sleeps).

## Common commands

```bash
make test
make test-unit
make test-integration
```
