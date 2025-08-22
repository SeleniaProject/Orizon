# Contributing to Orizon

Thank you for your interest in contributing to Orizon! We welcome contributions from developers of all skill levels.

## 🚀 Quick Start

1. **Fork** the repository
2. **Clone** your fork: `git clone https://github.com/YOUR_USERNAME/Orizon.git`
3. **Create** a feature branch: `git checkout -b feature/amazing-feature`
4. **Make** your changes
5. **Test** your changes: `make test`
6. **Commit** your changes: `git commit -m "Add amazing feature"`
7. **Push** to your branch: `git push origin feature/amazing-feature`
8. **Open** a Pull Request

## 📋 Development Guidelines

### Code Style

- **Go Code**: Follow `gofmt` and `golint` standards
- **Orizon Code**: Use `orizon-fmt` for formatting
- **Comments**: Write clear, concise comments in English
- **Naming**: Use descriptive names for variables and functions

### Testing Requirements

- **Unit Tests**: All new features must include unit tests
- **Integration Tests**: Complex features should include integration tests
- **Performance Tests**: Performance-critical code should include benchmarks
- **Documentation Tests**: Public APIs should include documentation examples

### Commit Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or modifying tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(compiler): add dependent type checking
fix(lexer): handle Unicode escape sequences correctly
docs(readme): update installation instructions
test(parser): add tests for generic functions
```

## 🧪 Testing

### Running Tests

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/lexer

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./internal/lexer
```

### Writing Tests

- **Test Files**: Place tests in `*_test.go` files
- **Test Functions**: Name test functions `TestFunctionName`
- **Benchmark Functions**: Name benchmark functions `BenchmarkFunctionName`
- **Table Tests**: Use table-driven tests for multiple test cases

Example:
```go
func TestLexer(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []Token
    }{
        {
            name:     "simple identifier",
            input:    "hello",
            expected: []Token{{Type: IDENTIFIER, Value: "hello"}},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## 📚 Documentation

### Code Documentation

- **Public APIs**: Must have complete documentation
- **Complex Logic**: Include inline comments explaining the approach
- **Examples**: Provide usage examples for public functions

### Writing Documentation

- **Markdown**: Use Markdown for all documentation
- **Structure**: Follow existing documentation structure
- **Links**: Use relative links within the repository
- **Images**: Place images in `docs/images/` directory

## 🐛 Bug Reports

When reporting bugs, please include:

1. **Environment**: OS, Go version, Orizon version
2. **Steps to Reproduce**: Clear steps to reproduce the issue
3. **Expected Behavior**: What you expected to happen
4. **Actual Behavior**: What actually happened
5. **Code Sample**: Minimal code sample that reproduces the issue
6. **Error Messages**: Complete error messages and stack traces

Use our bug report template:

```markdown
**Environment:**
- OS: [e.g., Windows 11, Ubuntu 22.04]
- Go Version: [e.g., 1.24.3]
- Orizon Version: [e.g., 0.1.0-alpha]

**Steps to Reproduce:**
1. Create file `test.oriz` with content: ...
2. Run `orizon-compiler test.oriz`
3. Observe error

**Expected Behavior:**
The code should compile successfully.

**Actual Behavior:**
Compiler crashes with segmentation fault.

**Error Message:**
```
panic: runtime error: invalid memory address
...
```

**Additional Context:**
This happens only with generic functions that use dependent types.
```

## 💡 Feature Requests

We welcome feature requests! Please:

1. **Search** existing issues to avoid duplicates
2. **Describe** the problem you're trying to solve
3. **Propose** a solution or API design
4. **Consider** backward compatibility
5. **Provide** use cases and examples

## 🏗️ Development Setup

### Prerequisites

- **Go**: 1.24.3 or later
- **Git**: Latest version
- **Make**: For build automation (Linux/macOS)
- **PowerShell**: 5.1 or later (Windows)

### Setting Up Development Environment

```bash
# Clone the repository
git clone https://github.com/SeleniaProject/Orizon.git
cd Orizon

# Install dependencies (if any)
go mod download

# Build the project
make build          # Linux/macOS
.\build.ps1 build   # Windows

# Run tests
make test           # Linux/macOS
.\build.ps1 test    # Windows
```

### IDE Setup

#### VS Code

Install recommended extensions:
- Go extension
- Orizon Language Support (when available)

#### Vim/Neovim

Configure Go and LSP support:
```vim
" Add to your .vimrc or init.vim
Plug 'fatih/vim-go'
Plug 'neovim/nvim-lspconfig'
```

## 🌟 Areas for Contribution

We especially welcome contributions in these areas:

### High Priority
- **Parser Improvements**: Error recovery, performance optimization
- **Type System**: Dependent types, effect tracking
- **Standard Library**: Core data structures, algorithms
- **Documentation**: Tutorials, examples, API documentation
- **Testing**: Fuzzing, property-based testing

### Medium Priority
- **IDE Support**: Language Server Protocol features
- **Tooling**: Debugger, profiler, package manager
- **Backends**: WebAssembly, GPU compilation
- **Optimization**: Compile-time optimization passes

### Beginner Friendly
- **Examples**: More example programs
- **Documentation**: Fixing typos, improving clarity
- **Tests**: Adding test cases for edge cases
- **Tooling**: Improving error messages

## 📞 Getting Help

- **Discord**: Join our [Discord server](https://discord.gg/orizon-lang)
- **GitHub Discussions**: Use GitHub Discussions for questions
- **Issues**: Create an issue for bugs and feature requests
- **Email**: Contact maintainers at dev@orizon-lang.org

## 👥 Community Guidelines

### Code of Conduct

We follow the [Contributor Covenant](https://www.contributor-covenant.org/):

- **Be respectful** and inclusive
- **Be collaborative** and helpful
- **Be patient** with newcomers
- **Be constructive** in feedback

### Communication

- **English**: Primary language for all communications
- **Be clear**: Use clear, concise language
- **Be professional**: Maintain professional tone
- **Be patient**: Allow time for responses

## 🏆 Recognition

Contributors are recognized in several ways:

- **Contributors file**: Listed in CONTRIBUTORS.md
- **Release notes**: Mentioned in release announcements
- **Website**: Featured on project website (when available)
- **Swag**: Occasional contributor swag for significant contributions

## 📄 License

By contributing to Orizon, you agree that your contributions will be licensed under both the MIT and Apache 2.0 licenses. See [LICENSE](LICENSE) for details.

---

**Thank you for contributing to Orizon! Together, we're building the future of systems programming.** 🌟
