package eventbus

// ClientDisconnectedPayload 客户端断开连接的事件负载
type ClientDisconnectedPayload struct {
	ClientID string
	Reason   string
}

// ClientConnectedPayload 客户端连接的事件负载
type ClientConnectedPayload struct {
	ClientID   string
	RemoteAddr string
}

// TrafficReportPayload 流量上报的事件负载
type TrafficReportPayload struct {
	MappingName string
	Bytes       int64
}
