# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - 2024-01-28

### Added
- **Modular Package Structure**: Refactored the codebase from a flat `main` package into a modular structure for better organization and maintainability.
  - `cmd/`: Contains the main application entry point and embedded web assets.
  - `internal/app/`: Houses business logic layers (model, service, handler, router).
  - `internal/pkg/`: Contains utility packages (helpers, response) for reusable functions.
- **HTTP Response Utilities**: Introduced reusable functions in `internal/pkg/response` for consistent JSON and error responses.
  - `SendJSON()`: Sends JSON responses with proper status codes.
  - `SendError()`: Sends standardized error responses.
- **Helper Utilities**: Organized helper functions (e.g., path parsing) into `internal/pkg/helpers` for better code reuse.
- **Go-Chi Router**: Replaced the standard `http.ServeMux` with go-chi for more flexible, idiomatic Go routing and potential middleware support.
- **Test File Relocation**: Moved test files to their corresponding package directories to align with Go testing conventions.
- **Dependencies**: Added `github.com/go-chi/chi/v5` for advanced routing capabilities.

### Changed
- **Package Organization**: Split the monolithic `main` package into focused packages (e.g., `handler`, `service`, `model`) to improve separation of concerns.
- **HTTP Handlers**: Refactored all HTTP response handling to use centralized utility functions, reducing code duplication and improving error handling consistency.
- **Imports and Exports**: Updated method signatures to be exported where needed for cross-package access, following idiomatic Go practices.
- **Build and Dependencies**: Updated `go.mod` and `go.sum` with new dependencies; ensured the application builds and runs correctly post-refactoring.

### Removed
- **Old Flat Structure**: Eliminated the original flat file structure (e.g., single `app.go`, `handlers.go`) in favor of the new modular layout.

### Why These Changes?
- **Maintainability**: The modular structure makes the codebase easier to navigate, test, and extend.
- **Scalability**: Separating concerns (e.g., business logic in `service`, routing in `router`) allows for better growth and feature additions.
- **Idiomatic Go**: Adopting go-chi and proper package organization aligns with Go best practices.
- **Reusability**: Centralized utilities reduce duplication and ensure consistent behavior across the application.
- **Testing**: Colocating tests with their packages improves test organization and execution.