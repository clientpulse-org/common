# Documentation

This directory contains documentation for all packages and middleware in the clientpulse-org/common repository.

## Package Documentation

### Authentication
- [**auth.md**](./auth.md) - JWT and Telegram authentication functionality
  - JWT token issuance from Telegram user data
  - JWT middleware for HTTP endpoints
  - Token validation and refresh
  - Complete authentication flow examples

## Getting Started

Each package documentation includes:
- ✅ **Installation instructions**
- ✅ **Quick start examples**  
- ✅ **API reference**
- ✅ **Usage patterns**
- ✅ **Security considerations**
- ✅ **Testing guidelines**

## Repository Structure

```
clientpulse-org/common/
├── docs/                    # 📚 Documentation
│   ├── README.md           # This file
│   └── auth.md             # Authentication package docs
├── pkg/                    # 📦 Packages
│   └── auth/              # Authentication package
│       ├── jwt.go         # JWT functionality
│       ├── telegram.go    # Telegram auth
│       └── *_test.go      # Tests
├── go.mod                 # Go module
└── README.md              # Project overview
```

## Contributing

When adding new packages or middleware:

1. **Create package** in `pkg/` directory
2. **Add documentation** in `docs/` directory
3. **Update this README** with new documentation links
4. **Include examples** and usage patterns
5. **Add comprehensive tests**

## Support

For questions or issues:
- Check package-specific documentation
- Review test files for usage examples
- Create an issue in the repository