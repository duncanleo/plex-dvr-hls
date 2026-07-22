# Agent guide

This repository is a Go HTTP service that emulates the HDHomeRun endpoints Plex
uses for DVRs and streams channel sources through `ffmpeg`.

## Start here

1. Read [ARCHITECTURE.md](ARCHITECTURE.md) for package ownership and runtime
   behaviour.
2. Use [docs/README.md](docs/README.md) as the index for operational and
   quality guidance. Read only the document relevant to the task.
3. Inspect the code and its adjacent `*_test.go` files before changing a
   package.
4. Run `make check` before handing off a change. Run `make test-race` for
   changes that touch shared state, streaming, file watching, or shutdown.

## Repository map

| Path | Responsibility |
| --- | --- |
| `cmd/` | Process lifecycle, HTTP server wiring, and graceful shutdown. |
| `config/` | Configuration, channel loading, playlist support, and reload-safe shared state. |
| `routes/` | HDHomeRun-compatible HTTP handlers and `ffmpeg` stream orchestration. |
| `routes/templates/` | Embedded XMLTV template. |
| `docs/` | Versioned project knowledge and execution plans. |
| `scripts/` | Dependency-free repository checks used locally and in CI. |

## Working conventions

- Keep changes scoped to the requested behaviour. Prefer a small, focused diff
  over unrelated cleanup.
- Write commit subjects in Conventional Commits form:
  `<type>(<optional scope>): <description>`. Use a type recognized by
  `cog.toml` (for example `feat`, `fix`, `docs`, `test`, or `chore`).
- Every commit an agent creates must include a `Co-Authored-By` trailer for
  that agent's own configured name and email. Configure an identity before
  committing if one is not already available.
- Run `gofmt` on every changed Go file. Do not hand-format generated output.
- Add or update tests with behaviour changes. Route tests should use the
  existing test seams rather than a real `ffmpeg` binary or external stream.
- Treat `channels.json` and `config.json` as user-owned runtime inputs. Never
  add real credentials, stream URLs, or generated `.device_id` files to git.
- When behaviour, configuration, reliability limits, or package boundaries
  change, update the relevant document in `docs/` in the same change.
- Record multi-step work with decisions or follow-up debt in
  `docs/plans/active/`. Move it to `docs/plans/completed/` when complete.

## Guardrails

- `config` owns the mutable channel list. Access it via `GetChannel`,
  `GetChannels`, `GetChannelCount`, or `SetChannels`; do not read or write the
  underlying slice from another package.
- A route may depend on `config`; `config` must not depend on `routes` or
  `cmd`.
- Channel IDs exposed by `/lineup.json` are one-based, volatile slice indexes.
  Do not present them as stable identifiers.
- Streaming must stop child `ffmpeg` processes when either the client
  disconnects or the server shuts down. Preserve both cancellation paths.
- Failed hot reloads keep the last valid channel configuration. Startup is the
  only time missing/invalid channels are fatal.

## Verification

```sh
make check
make test-race
```

`make check` verifies documentation structure, formatting, static analysis,
and the unit suite. It intentionally needs no third-party linting tool so it
works in a fresh Go environment.
