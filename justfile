set shell := ["bash", "-euo", "pipefail", "-c"]

version := `git describe --tags --always --dirty 2>/dev/null || echo dev`

[doc('List recipes')]
[group('core')]
default:
    @just --list --unsorted

[doc('Check formatting (fails on unformatted files)')]
[group('check')]
fmt-check:
    #!/usr/bin/env bash
    set -euo pipefail
    out="$(gofmt -l .)"
    if [[ -n "$out" ]]; then
        echo "gofmt needed:" >&2
        echo "$out" >&2
        exit 1
    fi

[doc('Format all Go files')]
[group('check')]
fmt:
    gofmt -w .

[doc('Run go vet')]
[group('check')]
vet:
    go vet ./...

[doc('Run the unit tests')]
[group('check')]
test:
    go test ./...

[doc('Build the justz binary into ./justz')]
[group('build')]
build:
    go build -ldflags "-X main.version={{ version }}" -o justz ./cmd/justz

[doc('Everything CI runs: format check, vet, test, build')]
[group('check')]
ci: fmt-check vet test build
