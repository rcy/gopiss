package gopiss

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// connectAndSubscribe connects to NASA's public ISSLIVE Lightstreamer feed and
// subscribes to the urine tank item (NODE3000005). Returns the websocket
// connection (ready to read updates) or an error.
func connectAndSubscribe() (*websocket.Conn, error) {
	dialer := &websocket.Dialer{
		Subprotocols:     []string{"TLCP-2.4.0.lightstreamer.com"},
		HandshakeTimeout: 10 * time.Second,
	}

	ws, _, err := dialer.Dial("wss://push.lightstreamer.com/lightstreamer", nil)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	// 1. Validate the websocket transport.
	if err := ws.WriteMessage(websocket.TextMessage, []byte("wsok\r\n")); err != nil {
		ws.Close()
		return nil, fmt.Errorf("write wsok: %w", err)
	}
	if _, msg, err := ws.ReadMessage(); err != nil || strings.TrimSpace(string(msg)) != "WSOK" {
		ws.Close()
		return nil, fmt.Errorf("unexpected wsok response %q: %v", msg, err)
	}

	// 2. Create a session against the ISSLIVE adapter set. No auth needed.
	createVals := url.Values{}
	createVals.Set("LS_adapter_set", "ISSLIVE")
	createVals.Set("LS_cid", "mgQkwtwdysogQz2BJ4Ji kOj2Bg") // generic public client ID
	createMsg := "create_session\r\n" + createVals.Encode()
	if err := ws.WriteMessage(websocket.TextMessage, []byte(createMsg)); err != nil {
		ws.Close()
		return nil, fmt.Errorf("write create_session: %w", err)
	}

	var sessionID string
	_, resp, err := ws.ReadMessage()
	if err != nil {
		ws.Close()
		return nil, fmt.Errorf("read create_session response: %w", err)
	}
	for _, line := range strings.Split(string(resp), "\r\n") {
		fields := strings.Split(line, ",")
		if fields[0] == "CONOK" && len(fields) >= 2 {
			sessionID = fields[1]
		}
	}
	if sessionID == "" {
		ws.Close()
		return nil, fmt.Errorf("no session id in response: %s", resp)
	}

	// 3. Subscribe to the urine tank item in MERGE mode with a snapshot,
	// so the server immediately sends the current value.
	subVals := url.Values{}
	subVals.Set("LS_session", sessionID)
	subVals.Set("LS_op", "add")
	subVals.Set("LS_mode", "MERGE")
	subVals.Set("LS_subId", "1")
	subVals.Set("LS_group", "NODE3000005")
	subVals.Set("LS_schema", "Value")
	subVals.Set("LS_snapshot", "true")
	subVals.Set("LS_reqId", "1")
	subMsg := "control\r\n" + subVals.Encode()
	if err := ws.WriteMessage(websocket.TextMessage, []byte(subMsg)); err != nil {
		ws.Close()
		return nil, fmt.Errorf("write subscribe: %w", err)
	}

	return ws, nil
}

// GetISSUrineTankLevel connects to NASA's public ISSLIVE Lightstreamer feed,
// subscribes to the urine tank item (NODE3000005), and returns the latest
// value it receives before closing the connection.
func GetISSUrineTankLevel() (float64, error) {
	ws, err := connectAndSubscribe()
	if err != nil {
		return 0, err
	}
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(20 * time.Second))

	// Read messages until we see an update ("U") for subscription 1.
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			return 0, fmt.Errorf("read update: %w", err)
		}
		for _, line := range strings.Split(string(msg), "\r\n") {
			parts := strings.SplitN(line, ",", 4)
			if len(parts) != 4 || parts[0] != "U" || parts[1] != "1" {
				continue
			}
			value, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
			if err != nil {
				return 0, fmt.Errorf("parse value %q: %w", parts[3], err)
			}
			return value, nil
		}
	}
}

// WatchISSUrineTankLevel connects to the Lightstreamer feed and returns a
// channel that receives the urine tank level whenever the value changes. The
// connection is closed and the goroutine exits when ctx is cancelled.
func WatchISSUrineTankLevel(ctx context.Context) (<-chan float64, error) {
	ws, err := connectAndSubscribe()
	if err != nil {
		return nil, err
	}

	ch := make(chan float64)
	go func() {
		defer ws.Close()
		defer close(ch)

		for {
			_ = ws.SetReadDeadline(time.Time{}) // no deadline — rely on ctx

			select {
			case <-ctx.Done():
				return
			default:
			}

			_, msg, err := ws.ReadMessage()
			if err != nil {
				return
			}

			for _, line := range strings.Split(string(msg), "\r\n") {
				parts := strings.SplitN(line, ",", 4)
				if len(parts) != 4 || parts[0] != "U" || parts[1] != "1" {
					continue
				}
				value, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
				if err != nil {
					continue
				}
				select {
				case ch <- value:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}
