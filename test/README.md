# Test Configuration for GoChat

## Test Structure

The test suite is organized into the following directories:

- `test/unit/` - Unit tests for individual components and functions
- `test/integration/` - Integration tests that test multiple components together
- `test/testhelpers/` - Common utilities and helpers for testing

## Running Tests

### Run All Tests

```bash
make test
```

### Run Tests with Coverage

```bash
make test-coverage
```

### Run Specific Test Types

```bash
# Unit tests only
go test ./test/unit/...

# Integration tests only
go test ./test/integration/...

# All tests in test directory
go test ./test/...
```

### Run Tests with Verbose Output

```bash
go test -v ./test/...
```

### Run Tests with Race Detection

```bash
make race
```

## Test Guidelines

1. **Unit Tests** should test individual functions or methods in isolation
2. **Integration Tests** should test the interaction between components
3. **Use Test Helpers** for common setup and assertion patterns
4. **Follow Naming Conventions**:
   - Test functions: `TestFunctionName`
   - Test files: `*_test.go`
5. **Use Table-Driven Tests** when testing multiple scenarios
6. **Add Test Documentation** explaining what each test validates

## Coverage Goals

- Maintain minimum 80% code coverage
- Focus on testing critical paths and error conditions
- Ensure all public APIs are tested

## Test Data

- Keep test data minimal and focused
- Use test helpers to create consistent test fixtures
- Avoid external dependencies in unit tests
