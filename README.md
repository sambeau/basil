# Basil

A web server for the Parsley programming language.

## Prerequisites

- [Go](https://golang.org/dl/) 1.24 or later

## Quick Start

Create a new project:

```bash
basil --init myproject
cd myproject
basil
```

Your site will be running at http://localhost:8080

## Getting Started

### Build

```bash
go build -o basil .
```

### Run

```bash
go run .
```

Or after building:

```bash
./basil
```

### Test

```bash
go test ./...
```

## Project Structure

```
basil/
├── .github/
│   └── copilot-instructions.md
├── go.mod
├── main.go
└── README.md
```

## License

MIT
