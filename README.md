# gopiss

Get the current urine tank level on the International Space Station.

Connects to NASA's public ISSLIVE telemetry stream via Lightstreamer and
returns the latest reading from `NODE3000005` — the urine tank sensor.

## Install

```sh
go get github.com/rcy/gopiss@latest
```

## Usage

### Get a single reading

```go
package main

import (
    "fmt"

    "github.com/rcy/gopiss"
)

func main() {
    level, err := gopiss.GetISSUrineTankLevel()
    if err != nil {
        panic(err)
    }
    fmt.Printf("ISS urine tank level: %.1f%%\n", level)
}
```

### Watch for changes

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rcy/gopiss"
)

func main() {
    ctx := context.Background()
    ch, err := gopiss.WatchISSUrineTankLevel(ctx)
    if err != nil {
        log.Fatal(err)
    }

    for level := range ch {
        fmt.Printf("ISS urine tank level: %.1f%%\n", level)
    }
}
```

## API

```go
func GetISSUrineTankLevel() (float64, error)
```

Returns the urine tank fill percentage from the latest telemetry snapshot.
Connects, reads one value, and closes the connection.

```go
func WatchISSUrineTankLevel(ctx context.Context) (<-chan float64, error)
```

Returns a channel that receives the tank level whenever it changes. The
connection stays open and the background goroutine exits when `ctx` is
cancelled.

Both functions require internet access (connect to `push.lightstreamer.com`
via WebSocket). No authentication needed.
