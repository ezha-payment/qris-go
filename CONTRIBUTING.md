# Contributing

Thanks for considering a contribution to qris-go.

## Running tests locally

The library depends only on the Go standard library. With Go 1.26 or later:

```sh
go build ./...
go test ./...
go test -race ./...
go test -cover ./...
```

Golden tests under `testdata/` compare parser output against checked-in JSON
fixtures. After an intentional change to parser output, regenerate them:

```sh
go test -update ./...
```

Review the resulting diff carefully before committing regenerated fixtures.

## Linting

CI runs [golangci-lint](https://golangci-lint.run). Run it locally before
opening a pull request:

```sh
golangci-lint run ./...
```

Code must also be `gofmt`-clean:

```sh
gofmt -l .
```

## Code style

- Follow standard Go conventions; keep the code `gofmt`-formatted.
- Write godoc comments as full sentences for every exported symbol.
- The parser is intentionally permissive so it can decode real-world payloads;
  the validation layer is strict. Keep that separation: do not move validation
  rules into the parser.
- Prefer the standard library. New third-party dependencies are out of scope.
- Add or update tests for any behavioral change. New payload shapes belong in
  `testdata/` with a matching golden file where applicable.

## Pull request process

1. Open an issue describing the change before significant work, so the approach
   can be agreed upon.
2. Create a topic branch from `main`.
3. Ensure `go build`, `go test`, `golangci-lint run`, and `gofmt -l .` all pass.
4. Keep commits focused and write clear commit messages explaining the why.
5. Update `CHANGELOG.md` under the `[Unreleased]` section when user-facing
   behavior changes.
6. Open the pull request against `main` and describe the change and its
   rationale.
