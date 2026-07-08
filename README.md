# gopiss

Get the current urine tank level on the International Space Station.

Connects to NASA's public ISSLIVE telemetry stream via Lightstreamer and
returns the latest reading from `NODE3000005` — the urine tank sensor.

## Install

```sh
go get github.com/rcy/gopiss@latest
```

## Usage

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

## API

```go
func GetISSUrineTankLevel() (float64, error)
```

Returns the urine tank fill percentage from the latest telemetry snapshot.
Requires internet access (connects to `push.lightstreamer.com` via WebSocket).
No authentication needed.
