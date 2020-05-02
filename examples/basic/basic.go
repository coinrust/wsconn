package main

import (
	"fmt"
	"github.com/coinrust/wsconn"
	"log"
)

func main() {
	wsURL := "wss://api.zb.live/websocket"
	ws := wsconn.NewWs(
		wsconn.WsUrlOption(wsURL),
		wsconn.WsDumpOption(true),
		wsconn.WsAutoReconnectOption(true),
		wsconn.WsMessageHandleFuncOption(func(bytes []byte) error {
			log.Printf("%v", string(bytes))
			return nil
		}),
		wsconn.WsErrorHandleFuncOption(func(err error) {
			log.Printf("%v", err)
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
