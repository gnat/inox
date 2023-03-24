package internal

import (
	core "github.com/inox-project/inox/internal/core"
	symbolic "github.com/inox-project/inox/internal/core/symbolic"
	net_symbolic "github.com/inox-project/inox/internal/globals/net/symbolic"
)

func init() {
	// register limitations
	core.LimRegistry.RegisterLimitation(WS_SIMUL_CONN_TOTAL_LIMIT_NAME, core.TotalLimitation, 0)
	core.LimRegistry.RegisterLimitation(TCP_SIMUL_CONN_TOTAL_LIMIT_NAME, core.TotalLimitation, 0)

	// register symbolic version of Go Functions
	core.RegisterSymbolicGoFunctions([]any{
		tcpConnect, func(ctx *symbolic.Context, host *symbolic.Host) (*net_symbolic.TcpConn, *symbolic.Error) {
			return &net_symbolic.TcpConn{}, nil
		},
		websocketConnect, func(ctx *symbolic.Context, u *symbolic.URL, opts ...*symbolic.Option) (*net_symbolic.WebsocketConnection, *symbolic.Error) {
			return &net_symbolic.WebsocketConnection{}, nil
		},
		NewWebsocketServer, func(ctx *symbolic.Context) (*net_symbolic.WebsocketServer, *symbolic.Error) {
			return &net_symbolic.WebsocketServer{}, nil
		},
		dnsResolve, func(ctx *symbolic.Context, domain *symbolic.String, recordTypeName *symbolic.String) (*symbolic.List, *symbolic.Error) {
			return symbolic.NewListOf(&symbolic.String{}), nil
		},
	})
}

func NewTcpNamespace() *core.Record {
	return core.NewRecordFromMap(core.ValMap{
		"connect": core.ValOf(tcpConnect),
	})
}

func NewDNSnamespace() *core.Record {
	f := func() (int, int) {
		return 1, 1
	}
	_, _ = f()
	return core.NewRecordFromMap(core.ValMap{
		"resolve": core.ValOf(dnsResolve),
	})
}

func NewWebsocketNamespace() *core.Record {
	return core.NewRecordFromMap(core.ValMap{
		"connect": core.ValOf(websocketConnect),
		"Server":  core.ValOf(NewWebsocketServer),
	})
}
