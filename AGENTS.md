# AGENTS.md

## Project overview

A Go library that queries the ISS urine tank level from NASA's public ISSLIVE
Lightstreamer websocket feed.

- **Module**: `github.com/rcy/gopiss`
- **Go version**: 1.25.5
- **Dependencies**: `github.com/gorilla/websocket` (v1.5.3)

## Architecture

Single-file codebase (`piss.go`), package `gopiss`. Imports as
`github.com/rcy/gopiss`; call `gopiss.GetISSUrineTankLevel()`.

### `piss.go`

`GetISSUrineTankLevel() (float64, error)`:
1. Opens a WebSocket to `wss://push.lightstreamer.com/lightstreamer` using the
   `TLCP-2.4.0.lightstreamer.com` subprotocol.
2. Sends `wsok` to validate the transport.
3. Creates a Lightstreamer session against the `ISSLIVE` adapter set with a
   hardcoded public client ID.
4. Subscribes to item `NODE3000005` (the urine tank telemetry node) in MERGE
   mode with snapshot enabled.
5. Reads messages until it receives an update (`U`) for subscription `1`, then
   parses and returns the float64 value.

Uses a 10s dial timeout and a 20s read deadline on the websocket.

## Commands

No Makefile or scripts exist. Standard Go tooling:

```sh
go build ./...       # verify compilation
go test ./...        # run tests (none currently exist)
go vet ./...         # static analysis
```

## Style / conventions

- Standard Go style (`gofmt`). No custom lint rules or formatters configured.
- Error wrapping uses `fmt.Errorf("context: %w", err)`.
- Lines are kept short and well-commented with inline explanations of the
  Lightstreamer protocol steps.

## Testing

No tests exist. There is no CI configuration. Any tests would need to mock the
websocket connection — there is no interface abstraction over
`gorilla/websocket.Dialer`.

## Gotchas

- **No auto-retry or reconnection logic**: The function dials, reads one value,
  and closes the connection. Callers must handle retries themselves.
- **Hardcoded Lightstreamer client ID**: The `LS_cid` value
  (`mgQkwtwdysogQz2BJ4Ji kOj2Bg`) is NASA's generic public client identifier.
  The space in the middle is intentional — it's part of the Lightstreamer
  protocol's two-part CID format. Do not remove or escape it.
- **Read deadline**: A 20-second read deadline is set. If NASA's telemetry
  stream is slow, the call may return a timeout error.
