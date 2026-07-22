# Architecture decisions

The detailed architecture map is maintained at
[ARCHITECTURE.md](../ARCHITECTURE.md), alongside the codebase entry point. This
document captures decisions that affect how future changes should be made.

## Decision: keep the package model small

The service has three responsibilities: process lifecycle (`cmd`), mutable
runtime inputs (`config`), and HTTP/streaming behaviour (`routes`). A new
package should represent a stable responsibility, not merely split a file by
size. This keeps dependency direction obvious to both maintainers and agents.

## Decision: favor explicit test seams at system boundaries

The stream handler exposes the `execCommandContext` seam so tests can model
`ffmpeg` behaviour without depending on a local binary or network stream.
File watching similarly accepts a reload function in its internal loop. Keep
new external boundaries injectable when a deterministic unit test is practical.

## Decision: preserve the last known good channel list

Runtime input files are commonly changed with atomic-save editors or can be
temporarily malformed. A reload failure must not take an already-running DVR
offline. Startup remains strict because there is no safe previous state.
