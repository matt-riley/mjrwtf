---
name: Bug Report
about: Report a bug in mjr.wtf URL shortener
title: '[BUG] '
labels: ['bug', 'needs-triage']
assignees: ''
---

## Bug Description

<!-- Clear and concise description of the bug -->

## Steps to Reproduce

1. Go to '...'
2. Click on '....'
3. Scroll down to '....'
4. See error

## Expected Behavior

<!-- What should happen -->

## Actual Behavior

<!-- What actually happens -->

## Error Messages

<!-- Include full error messages, stack traces, log output -->

```
Paste error messages here
```

## Environment

- **Go Version:** [ Run: `go version` ]
- **Database:** [ SQLite | PostgreSQL ]
- **Database Version:** [ Run: `sqlite3 --version` or `psql --version` ]
- **OS:** [ e.g., macOS 13.5, Ubuntu 22.04 ]
- **Project Commit:** [ Run: `git rev-parse HEAD` ]

## Minimal Reproduction

<!-- If possible, provide minimal code to reproduce the issue -->

```go
// Minimal code that demonstrates the bug
```

## Impact

**Severity:** [ Critical | High | Medium | Low ]  
**Frequency:** [ Always | Often | Sometimes | Rare ]  
**Users Affected:** [ All | Specific scenario | Single user ]

## Acceptance Criteria for Fix

- [ ] Bug no longer reproducible with steps above
- [ ] Regression test added to prevent recurrence
- [ ] All existing tests continue to pass
- [ ] Root cause documented in PR description

## Additional Context

<!-- Screenshots, related issues, workarounds, etc. -->

---

**Component:** [ URL Management | Click Tracking | Analytics | Authentication | API | Database | Migration | Tests ]
