package wsconn

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"log"
	"net/http/httputil"
	"time"
)

type WsConn struct {
	WsParameter

	c                      *websocket.Conn
	dialer                 *websocket.Dialer
	writeBufferChan        chan []byte
	pingMessageBufferChan  chan []byte
	closeMessageBufferChan chan []byte
	subs                   []interface{}
	close                  chan struct{}
}

func (ws *WsConn) setDefaultWsParameter() {
	ws.AutoReconnect = true
	ws.ReSubscribe = true
}

func (ws *WsConn) connect() error {
	wsConn, resp, err := ws.dialer.Dial(ws.WsUrl, ws.ReqHeaders)
	if err != nil {
		log.Printf("[ws][%s] error: %s", ws.WsUrl, err.Error())
		if ws.IsDump && resp != nil {
			dumpData, _ := httputil.DumpResponse(resp, true)
			log.Printf("[ws][%s] %s", ws.WsUrl, string(dumpData))
		}
		return err
	}

	ws.c = wsConn

	if ws.IsDump {
		dumpData, _ := httputil.DumpResponse(resp, true)
		log.Printf("[ws][%s] %s", ws.WsUrl, string(dumpData))
	}

	return nil
}

func (ws *WsConn) reconnect() {
	ws.c.Close()

	var err error

	sleep := 1
	for retry := 1; retry <= 10; retry += 1 {
		time.Sleep(time.Duration(sleep) * time.Second)

		err = ws.connect()
		if err != nil {
			log.Printf("[ws][%s] websocket reconnect fail, %s", ws.WsUrl, err.Error())
		} else {
			break
		}

		sleep <<= 1
	}

	if err != nil {
		log.Printf("[ws][%s] retry reconnect fail, begin exiting.", ws.WsUrl)
		ws.Close()
		if ws.ErrorHandleFunc != nil {
			ws.ErrorHandleFunc(errors.New("retry reconnect fail"))
		}
	} else {
		// re-subscribe
		if ws.ReSubscribe {
			var subs []interface{}
			copy(subs, ws.subs)
			ws.subs = ws.subs[:0]
			for _, sub := range subs {
				ws.Subscribe(sub)
			}
		}
		if ws.ReSubscribeFunc != nil {
			err = ws.ReSubscribeFunc(ws)
			if err != nil {
				log.Printf("[ws] websocket re-subscribe fail, %s", err.Error())
			}
		}
	}
}

func (ws *WsConn) writeRequest() {
	var err error

	for {
		select {
		case <-ws.close:
			// stop the goroutine if the ws is closed
			// log.Printf("[ws][%s] close websocket, exiting write message goroutine.", ws.WsUrl)
			return
		case d := <-ws.writeBufferChan:
			err = ws.c.WriteMessage(websocket.TextMessage, d)
		case d := <-ws.pingMessageBufferChan:
			err = ws.c.WriteMessage(websocket.PingMessage, d)
		case d := <-ws.closeMessageBufferChan:
			err = ws.c.WriteMessage(websocket.CloseMessage, d)
		}

		if err != nil {
			log.Printf("[ws][%s] write error: %s", ws.WsUrl, err.Error())
			time.Sleep(time.Second)
		}
	}
}

func (ws *WsConn) Dialer() *websocket.Dialer {
	return ws.dialer
}

func (ws *WsConn) Subscribe(sub interface{}) error {
	data, err := json.Marshal(sub)
	if err != nil {
		log.Printf("[ws][%s] json encode error , %s", ws.WsUrl, err)
		return err
	}
	ws.writeBufferChan <- data
	if ws.ReSubscribe {
		ws.subs = append(ws.subs, sub)
	}
	return nil
}

func (ws *WsConn) SendMessage(msg []byte) {
	ws.writeBufferChan <- msg
}

func (ws *WsConn) SendPingMessage(msg []byte) {
	ws.pingMessageBufferChan <- msg
}

func (ws *WsConn) SendCloseMessage(msg []byte) {
	ws.closeMessageBufferChan <- msg
}

func (ws *WsConn) SendJsonMessage(m interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		log.Printf("[ws][%s] json encode error, %s", ws.WsUrl, err)
		return err
	}
	ws.writeBufferChan <- data
	return nil
}

func (ws *WsConn) ReceiveMessage(msg []byte) {
	ws.MessageHandleFunc(msg)
}

func (ws *WsConn) receiveMessage() {
	ws.c.SetCloseHandler(func(code int, text string) error {
		log.Printf("[ws][%s] websocket exiting [code=%d, text=%s]", ws.WsUrl, code, text)
		ws.Close()
		return nil
	})

	ws.c.SetPongHandler(func(pong string) error {
		// log.Printf("[ws][%s] pong received", ws.WsUrl)
		ws.c.SetReadDeadline(time.Now().Add(ws.readDeadLineTime))
		return nil
	})

	ws.c.SetPingHandler(func(ping string) error {
		//log.Printf("[ws][%s] ping received", ws.WsUrl)
		ws.c.SetReadDeadline(time.Now().Add(ws.readDeadLineTime))
		err := ws.c.WriteMessage(websocket.PongMessage, nil)
		if err == nil {
			//log.Printf("[ws][%s] pong sent", ws.WsUrl)
		}
		return err
	})

	for {

		t, msg, err := ws.c.ReadMessage()

		if len(ws.close) > 0 {
			// stop goroutine if the ws is closed
			//log.Printf("[ws][%s] close websocket, exiting receive message goroutine.", ws.WsUrl)
			return
		}

		if err != nil {
			log.Printf("[ws][%s] error: %s", ws.WsUrl, err)
			if ws.AutoReconnect {
				log.Printf("[ws][%s] Unexpected Closed, Begin Retry Connect.", ws.WsUrl)
				ws.reconnect()
				continue
			}

			if ws.ErrorHandleFunc != nil {
				ws.ErrorHandleFunc(err)
			}

			return
		}

		ws.c.SetReadDeadline(time.Now().Add(ws.readDeadLineTime))

		switch t {
		case websocket.TextMessage:
			ws.MessageHandleFunc(msg)
		case websocket.BinaryMessage:
			ws.MessageHandleFunc(msg)
		case websocket.CloseMessage:
			ws.Close()
			return
		default:
			log.Printf("[ws][%s] error websocket message type, content is :\n %s \n", ws.WsUrl, string(msg))
		}
	}
}

func (ws *WsConn) Close() {
	ws.close <- struct{}{} // one for the write goroutine
	ws.close <- struct{}{} // another for the read goroutine

	err := ws.c.Close()
	if err != nil {
		log.Printf("[ws][%s] close websocket error: %s", ws.WsUrl, err)
	}

	if ws.IsDump {
		log.Printf("[ws][%s] connection closed", ws.WsUrl)
	}
}

func NewWs(opts ...WsParameterOption) *WsConn {
	ws := &WsConn{}
	ws.setDefaultWsParameter()

	for _, opt := range opts {
		opt(&ws.WsParameter)
	}

	if ws.WsParameter.Dialer != nil {
		ws.dialer = ws.WsParameter.Dialer
	} else {
		ws.dialer = DefaultDialer()
	}
	ws.readDeadLineTime = time.Minute

	if err := ws.connect(); err != nil {
		log.Panic(err)
	}

	ws.close = make(chan struct{}, 2)
	ws.pingMessageBufferChan = make(chan []byte, 10)
	ws.closeMessageBufferChan = make(chan []byte, 1)
	ws.writeBufferChan = make(chan []byte, 10)

	go ws.writeRequest()
	go ws.receiveMessage()
	return ws
}

func DefaultDialer() *websocket.Dialer {
	return websocket.DefaultDialer
}
