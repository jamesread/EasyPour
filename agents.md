# Development Guidelines for EasyPour

## Code Quality Standards

### Test-Driven Development (TDD)
- **Always write tests first** before implementing features
- Follow the Red-Green-Refactor cycle:
  1. Write a failing test (Red)
  2. Write minimal code to pass (Green)
  3. Refactor while keeping tests passing
- Maintain high test coverage (>80% minimum)
- Tests should be fast, isolated, and repeatable

### Cyclomatic Complexity
- **Maximum complexity: 5**
- Refactor functions that exceed this limit
- Break down complex logic into smaller, focused functions
- Use early returns and guard clauses to reduce nesting

### Code Comments
- **Minimal comments preferred**
- Code should be self-documenting through:
  - Clear variable and function names
  - Small, focused functions
  - Expressive code structure
- Only comment when:
  - Explaining "why" not "what"
  - Documenting non-obvious business logic
  - Clarifying complex algorithms
- Avoid redundant comments that restate the code

## Best Practices

- Write small, single-responsibility functions
- Prefer composition over inheritance
- Use meaningful names that express intent
- Keep functions pure when possible
- Handle errors explicitly, don't ignore them