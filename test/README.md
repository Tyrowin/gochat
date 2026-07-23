# Tests

- `unit/` — components in isolation
- `integration/` — real HTTP/WebSocket servers over real sockets
- `testhelpers/` — shared setup and assertion helpers

```bash
make test                # everything, with -race
make test-unit           # unit only
make test-integration    # integration only
make test-coverage       # coverage.out + coverage.html
```

Suite layout, helper reference, coverage numbers, and conventions for writing new tests:
[docs/guides/testing.md](../docs/guides/testing.md).
