package cloudproxy

import (
	"path/filepath"

	"github.com/inoxlang/inox/internal/core"
	"github.com/inoxlang/inox/internal/core/permkind"
)

func createContexts(host core.Host, proxyArgs CloudProxyArgs) (ctx, topCtx *core.Context) {
	databaseDir := core.DirPathFrom(filepath.Dir(proxyArgs.Config.AnonymousAccountDatabasePath))
	databaseDirPattern := databaseDir.ToPrefixPattern()

	perms := []core.Permission{
		core.WebsocketPermission{
			Kind_:    permkind.Provide,
			Endpoint: host,
		},
		core.WebsocketPermission{
			Kind_: permkind.Read,
		},
		core.WebsocketPermission{
			Kind_: permkind.Write,
		},
		core.FilesystemPermission{
			Kind_:  permkind.Read,
			Entity: databaseDirPattern,
		},
		core.FilesystemPermission{
			Kind_:  permkind.Write,
			Entity: databaseDirPattern,
		},
	}

	topCtx = core.NewContextWithEmptyState(core.ContextConfig{
		Filesystem:          proxyArgs.Filesystem,
		Permissions:         perms,
		ParentStdLibContext: proxyArgs.GoContext,
	}, proxyArgs.OutW)

	ctx = core.NewContextWithEmptyState(core.ContextConfig{
		ParentContext: topCtx,
		Permissions:   perms,
	}, proxyArgs.OutW)

	return ctx, topCtx
}
