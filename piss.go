package gopiss

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// GetISSUrineTankLevel connects to NASA's public ISSLIVE Lightstreamer feed,
// subscribes to the urine tank item (NODE3000005), and returns the latest
// value it receives before closing the connection.
func GetISSUrineTankLevel() (float64, error) {
	dialer := &websocket.Dialer{
		Subprotocols:     []string{"TLCP-2.4.0.lightstreamer.com"},
		HandshakeTimeout: 10 * time.Second,
	}

	ws, _, err := dialer.Dial("wss://push.lightstreamer.com/lightstreamer", nil)
	if err != nil {
		return 0, fmt.Errorf("dial: %w", err)
	}
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(20 * time.Second))

	// 1. Validate the websocket transport.
	if err := ws.WriteMessage(websocket.TextMessage, []byte("wsok\r\n")); err != nil {
		return 0, fmt.Errorf("write wsok: %w", err)
	}
	if _, msg, err := ws.ReadMessage(); err != nil || strings.TrimSpace(string(msg)) != "WSOK" {
		return 0, fmt.Errorf("unexpected wsok response %q: %v", msg, err)
	}

	// 2. Create a session against the ISSLIVE adapter set. No auth needed.
	createVals := url.Values{}
	createVals.Set("LS_adapter_set", "ISSLIVE")
	createVals.Set("LS_cid", "mgQkwtwdysogQz2BJ4Ji kOj2Bg") // generic public client ID
	createMsg := "create_session\r\n" + createVals.Encode()
	if err := ws.WriteMessage(websocket.TextMessage, []byte(createMsg)); err != nil {
		return 0, fmt.Errorf("write create_session: %w", err)
	}

	var sessionID string
	_, resp, err := ws.ReadMessage()
	if err != nil {
		return 0, fmt.Errorf("read create_session response: %w", err)
	}
	for _, line := range strings.Split(string(resp), "\r\n") {
		fields := strings.Split(line, ",")
		if fields[0] == "CONOK" && len(fields) >= 2 {
			sessionID = fields[1]
		}
	}
	if sessionID == "" {
		return 0, fmt.Errorf("no session id in response: %s", resp)
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
		return 0, fmt.Errorf("write subscribe: %w", err)
	}

	// 4. Read messages until we see an update ("U") for subscription 1.
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
