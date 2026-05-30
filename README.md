# gincontextleak

[![Go Reference](https://pkg.go.dev/badge/github.com/haruyama480/gincontextleak.svg)](https://pkg.go.dev/github.com/haruyama480/gincontextleak)
[![Release](https://img.shields.io/github/v/release/haruyama480/gincontextleak)](https://github.com/haruyama480/gincontextleak/releases/latest)
[![CI](https://github.com/haruyama480/gincontextleak/actions/workflows/release.yaml/badge.svg)](https://github.com/haruyama480/gincontextleak/actions/workflows/release.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/haruyama480/gincontextleak)](https://goreportcard.com/report/github.com/haruyama480/gincontextleak)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/haruyama480/gincontextleak/blob/main/LICENSE)

A static analysis tool that detects unsafe passing of `*gin.Context` to functions expecting `context.Context`.

`gin.Context` is not goroutine-safe. Passing it where a `context.Context` is expected can cause data races when the receiver later uses the context concurrently. This linter finds such call sites. When invoked with `-fix`, it rewrites the argument to `ctx.Request.Context()` to avoid the issue.

## Background

- [gin-gonic/gin#4117](https://github.com/gin-gonic/gin/issues/4117)
- https://github.com/haruyama480/go-gin-context-conflict
- https://engineering.nifty.co.jp/blog/35119

## Installation

```bash
go install github.com/haruyama480/gincontextleak/cmd/gincontextleak@latest
```

```bash
brew install haruyama480/tap/gincontextleak
```

## Usage

Check your code:

```bash
gincontextleak ./...
```

Automatically fix issues:

```bash
gincontextleak -fix ./...
```

## Version information

Go tool convention (`-V` flag only):

```bash
gincontextleak -V=full
go version -m $(which gincontextleak)
```

## Limitations

The linter only detects cases where `*gin.Context` is passed directly as an argument to a function or method parameter of type `context.Context`.

Although `*gin.Context` implements the `context.Context` interface, the following patterns are **not** detected:

- Variable assignments (`var ctx context.Context = c` or `ctx = c`)
- Returning `*gin.Context` from a function (`return c`)
- Storing it in structs, slices, maps, channels, etc.
- Capturing it in closures or goroutines
- Indirect passing via `interface{}` or other intermediate types

Detecting all of these cases would require dataflow analysis, which is not currently implemented.

## Links

- [Go Reference / godoc](https://pkg.go.dev/github.com/haruyama480/gincontextleak)
- [Releases & Artifacts](https://github.com/haruyama480/gincontextleak/releases) — Prebuilt binaries (Linux/macOS/Windows) via GoReleaser
- [CI/CD](https://github.com/haruyama480/gincontextleak/actions)
- [Homebrew Tap](https://github.com/haruyama480/homebrew-tap)
- [Source Code](https://github.com/haruyama480/gincontextleak)
