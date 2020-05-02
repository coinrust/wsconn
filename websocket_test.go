package wsconn

import (
	"fmt"
	"testing"
)

func TestWsConn_AllInOne(t *testing.T) {
	host := "zb.live"
	wsURL := fmt.Sprintf("wss://api.%v/websocket", host) // zb.live
	ws := NewWs(
		WsUrlOption(wsURL),
		WsDumpOption(true),
		WsAutoReconnectOption(true),
		WsMessageHandleFuncOption(func(bytes []byte) error {
			t.Logf("%v", string(bytes))
			return nil
		}),
		WsErrorHandleFuncOption(func(err error) {
			t.Logf("%v", err)
		}),
	)
	ch := fmt.Sprintf("%v_depth", "zbqc") // zbqc_depth
	sub := map[string]string{
		"event":   "addChannel",
		"channel": ch,
	}
	ws.Subscribe(sub)

	select {}
}
