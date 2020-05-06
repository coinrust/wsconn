package wsconn

import (
	"github.com/gorilla/websocket"
	"time"
)

type WsParameter struct {
	WsUrl             string
	ReqHeaders        map[string][]string
	Dialer            *websocket.Dialer
	MessageHandleFunc func([]byte) error
	ReSubscribeFunc   func() error // 短线重连后触发，你可以重新订阅，因为有些订阅跟当前时间相关
	ErrorHandleFunc   func(err error)
	AutoReconnect     bool // 自动重连(默认: true)
	ReSubscribe       bool // 自动重新订阅(默认: true)
	IsDump            bool
	readDeadLineTime  time.Duration
}

type WsParameterOption func(p *WsParameter)

func WsUrlOption(wsUrl string) WsParameterOption {
	return func(p *WsParameter) {
		p.WsUrl = wsUrl
	}
}

func WsReqHeaderOption(key, value string) WsParameterOption {
	return func(p *WsParameter) {
		p.ReqHeaders[key] = append(p.ReqHeaders[key], value)
	}
}

func WsDialerOption(dialer *websocket.Dialer) WsParameterOption {
	return func(p *WsParameter) {
		p.Dialer = dialer
	}
}

func WsAutoReconnectOption(autoReconnect bool) WsParameterOption {
	return func(p *WsParameter) {
		p.AutoReconnect = autoReconnect
	}
}

func WsDumpOption(dump bool) WsParameterOption {
	return func(p *WsParameter) {
		p.IsDump = dump
	}
}

func WsMessageHandleFuncOption(f func([]byte) error) WsParameterOption {
	return func(p *WsParameter) {
		p.MessageHandleFunc = f
	}
}

func WsErrorHandleFuncOption(f func(err error)) WsParameterOption {
	return func(p *WsParameter) {
		p.ErrorHandleFunc = f
	}
}
