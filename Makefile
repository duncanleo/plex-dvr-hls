.PHONY: check docs-check fmt fmt-check test test-race vet

check: docs-check fmt-check vet test

docs-check:
	./scripts/check-docs.sh

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

fmt-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))" || \
		(printf 'Run make fmt to format these files:\n'; gofmt -l $$(find . -name '*.go' -not -path './vendor/*'); exit 1)

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...
