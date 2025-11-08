# Domain Layer Instructions

These files contain pure business logic with no external dependencies.

## Rules

1. **No External Dependencies**: Never import database, HTTP, or infrastructure packages
2. **Validation Required**: All entities must validate their own state
3. **Immutability**: Entities should be immutable after creation where possible
4. **Domain Errors**: Use domain-specific errors from `errors.go`
5. **Repository Interfaces**: Define interfaces (ports) but never implementations

## Entity Structure Pattern

```go
type EntityName struct {
    ID        string
    // fields...
    CreatedAt time.Time
}

func NewEntityName(...) (*EntityName, error) {
    entity := &EntityName{...}
    if err := entity.Validate(); err != nil {
        return nil, err
    }
    return entity, nil
}

func (e *EntityName) Validate() error {
    // validation logic
    return nil
}
```

## Repository Interface Pattern

```go
type EntityRepository interface {
    Create(ctx context.Context, entity *Entity) error
    FindByID(ctx context.Context, id string) (*Entity, error)
    // other methods...
}
```

## Testing

- Test validation thoroughly with table-driven tests
- Test all edge cases (empty strings, nil values, boundary conditions)
- No database required for domain tests
