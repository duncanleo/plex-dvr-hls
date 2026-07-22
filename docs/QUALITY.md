# Quality and verification

## Required local checks

```sh
make check
```

This runs, in order:

1. `make docs-check` — repository knowledge has required files, headings, and
   index links.
2. `make fmt-check` — every Go file is already formatted.
3. `make vet` — Go static analysis.
4. `make test` — package unit tests.

Run `make test-race` whenever a change affects shared state, file watching,
goroutines, request cancellation, or process lifecycle. It is also the right
pre-merge check for a non-trivial change to `config` or streaming.

## Definition of done

- The requested behaviour is covered by a deterministic test or an explicit
  note explaining why it cannot be automated.
- Formatting, static analysis, and relevant tests pass.
- User-visible configuration or behaviour is reflected in the README and
  operations documentation.
- Architectural or reliability decisions are recorded where future work can
  discover them.
- No secrets, private stream URLs, generated state, or local configuration are
  included in the diff.

## Release notes

Tagged GitHub releases generate their body with Cocogitto's `full_hash`
template. The tag prefix and changelog inclusion rules are defined in
[`cog.toml`](../cog.toml); the generated `CHANGELOG.md` is workflow output and
is not committed.

## Keeping entropy low

Treat review feedback and repeated failures as candidates for a durable rule:
document a decision when context is enough, or encode it in a test/check when
it must be enforced. Avoid broad cleanup mixed with a behaviour change; make a
small, dedicated follow-up plan instead.
