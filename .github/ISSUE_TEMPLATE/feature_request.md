---
name: Feature Request
about: Propose a new feature for mjr.wtf URL shortener
title: '[FEATURE] '
labels: ['feature', 'needs-triage']
assignees: ''
---

## User Story

**As a** [type of user]  
**I want** [capability or feature]  
**So that** [benefit or value]

## Problem Statement

<!-- Describe the problem this feature solves -->

## Proposed Solution

<!-- Describe your preferred solution -->

## Acceptance Criteria

<!-- Define specific, measurable criteria using Given-When-Then format -->

- [ ] Given [context], when [action], then [outcome]
- [ ] Given [context], when [action], then [outcome]
- [ ] All existing tests continue to pass
- [ ] New functionality has unit tests
- [ ] Documentation updated (if applicable)

## Technical Considerations

### Database Changes
<!-- List any schema changes, migrations, or new queries needed -->

### API Changes
<!-- List new endpoints or changes to existing endpoints -->

### Architecture Impact
<!-- Which layers are affected? (domain/adapters/infrastructure) -->

### Security Considerations
<!-- Authentication, authorization, input validation, etc. -->

### Performance Impact
<!-- Expected impact on performance, scalability concerns -->

## Implementation Guidance

### Files Likely to Change
<!-- Help Copilot by listing files that need modification -->
- [ ] `internal/domain/<entity>/` - [ Description ]
- [ ] `internal/adapters/repository/` - [ Description ]
- [ ] `internal/migrations/` - [ Description ]

### Required Steps
<!-- Ordered list of implementation steps -->
1. Create migration for schema changes
2. Update domain entity and validation
3. Add repository methods
4. Update sqlc queries
5. Add tests (domain + repository)
6. Update API handlers (if applicable)

## Dependencies

<!-- Link related issues -->
- Blocks: #
- Blocked by: #
- Related: #

## Additional Context

<!-- Screenshots, examples, references, etc. -->

---

**Priority:** [ P0: Critical | P1: High | P2: Medium | P3: Low ]  
**Complexity:** [ XS | S | M | L | XL ]  
**Component:** [ URL Management | Click Tracking | Analytics | Authentication | API | Infrastructure ]
