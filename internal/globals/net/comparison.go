package internal

import core "github.com/inox-project/inox/internal/core"

func (conn *WebsocketConnection) Equal(ctx *core.Context, other Value, alreadyCompared map[uintptr]uintptr, depth int) bool {
	otherConn, ok := other.(*WebsocketConnection)
	return ok && conn == otherConn
}

func (s *WebsocketServer) Equal(ctx *core.Context, other Value, alreadyCompared map[uintptr]uintptr, depth int) bool {
	otherServer, ok := other.(*WebsocketServer)
	return ok && s == otherServer
}

func (conn *TcpConn) Equal(ctx *core.Context, other Value, alreadyCompared map[uintptr]uintptr, depth int) bool {
	otherConn, ok := other.(*TcpConn)
	return ok && conn == otherConn
}
