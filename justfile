set shell := ["bash", "-euo", "pipefail", "-c"]

gotestsum := "go run gotest.tools/gotestsum@v1.13.0"
govulncheck := "go run golang.org/x/vuln/cmd/govulncheck@v1.9.0"
coverage_threshold := "80"

default:
    @just --list

# Vet and lint with the govengers baseline (.golangci.yml, copied from govengers/standards)
lint:
    go vet ./...
    go vet -tags e2e ./test/e2e/...
    golangci-lint run ./...

# Unit tests with the race detector; fails below the coverage threshold
test:
    mkdir -p sandbox
    {{gotestsum}} --format testname --jsonfile sandbox/unit.json -- -race -coverprofile=sandbox/unit.cover ./...
    go tool cover -func=sandbox/unit.cover > sandbox/unit.func
    awk -v threshold={{coverage_threshold}} '/^total:/ { found = 1; sub("%", "", $3); print "coverage " $3 "%"; if ($3 + 0 < threshold) { print "coverage is below " threshold "%"; exit 1 } } END { if (!found) { print "no coverage total"; exit 1 } }' sandbox/unit.func

# End-to-end tests: build the example program in test/e2e/testdata/app and run it as a process
test-e2e:
    mkdir -p sandbox
    {{gotestsum}} --format testname --jsonfile sandbox/e2e.json -- -tags e2e -count=1 ./test/e2e/...

# Scan for known vulnerabilities in reachable code
vulncheck:
    {{govulncheck}} ./...

# Everything CI runs
check: lint test test-e2e vulncheck
