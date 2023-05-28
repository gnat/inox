package internal

import (
	"io"

	"github.com/inoxlang/inox/internal/afs"
	"github.com/inoxlang/inox/internal/config"
	core "github.com/inoxlang/inox/internal/core"
	_chrome "github.com/inoxlang/inox/internal/globals/chrome"
	_containers "github.com/inoxlang/inox/internal/globals/containers"
	_dom "github.com/inoxlang/inox/internal/globals/dom"
	_env "github.com/inoxlang/inox/internal/globals/env"
	_fs "github.com/inoxlang/inox/internal/globals/fs"
	_help "github.com/inoxlang/inox/internal/globals/help"
	_html "github.com/inoxlang/inox/internal/globals/html"
	_http "github.com/inoxlang/inox/internal/globals/http"
	_locdb "github.com/inoxlang/inox/internal/globals/local_db"
	_net "github.com/inoxlang/inox/internal/globals/net"
	_s3 "github.com/inoxlang/inox/internal/globals/s3"
	_shell "github.com/inoxlang/inox/internal/globals/shell"
	_sql "github.com/inoxlang/inox/internal/globals/sql"
	_strmanip "github.com/inoxlang/inox/internal/globals/strmanip"
	pprint "github.com/inoxlang/inox/internal/pretty_print"
	"github.com/inoxlang/inox/internal/utils"
	"github.com/rs/zerolog"
)

var (
	DEFAULT_SCRIPT_LIMITATIONS = []core.Limitation{
		{Name: _fs.FS_READ_LIMIT_NAME, Kind: core.ByteRateLimitation, Value: 100_000_000},
		{Name: _fs.FS_WRITE_LIMIT_NAME, Kind: core.ByteRateLimitation, Value: 100_000_000},

		{Name: _fs.FS_NEW_FILE_RATE_LIMIT_NAME, Kind: core.SimpleRateLimitation, Value: 100},
		{Name: _fs.FS_TOTAL_NEW_FILE_LIMIT_NAME, Kind: core.ByteRateLimitation, Value: 10_000},

		{Name: _net.HTTP_REQUEST_RATE_LIMIT_NAME, Kind: core.ByteRateLimitation, Value: 100},
		{Name: _net.WS_SIMUL_CONN_TOTAL_LIMIT_NAME, Kind: core.TotalLimitation, Value: 10},
		{Name: _net.TCP_SIMUL_CONN_TOTAL_LIMIT_NAME, Kind: core.TotalLimitation, Value: 10},
	}

	DEFAULT_LOG_PRINT_CONFIG = &core.PrettyPrintConfig{
		PrettyPrintConfig: pprint.PrettyPrintConfig{
			MaxDepth: 10,
			Colorize: false,
			Compact:  true,
		},
	}

	DEFAULT_PRETTY_PRINT_CONFIG = &core.PrettyPrintConfig{
		PrettyPrintConfig: pprint.PrettyPrintConfig{
			MaxDepth: 7,
			Colorize: config.SHOULD_COLORIZE,
			Colors: utils.If(config.INITIAL_COLORS_SET && config.INITIAL_BG_COLOR.IsDarkBackgroundColor(),
				&pprint.DEFAULT_DARKMODE_PRINT_COLORS,
				&pprint.DEFAULT_LIGHTMODE_PRINT_COLORS,
			),
			Compact:                     false,
			Indent:                      []byte{' ', ' '},
			PrintDecodedTopLevelStrings: true,
		},
	}

	STR_CONVERSION_PRETTY_PRINT_CONFIG = &core.PrettyPrintConfig{
		PrettyPrintConfig: pprint.PrettyPrintConfig{
			MaxDepth: 10,
			Colorize: false,
			Compact:  true,
		},
	}

	_ = []core.GoValue{
		&_html.HTMLNode{}, &core.GoFunction{}, &_http.HttpServer{}, &_net.TcpConn{}, &_net.WebsocketConnection{}, &_http.HttpRequest{}, &_http.HttpResponseWriter{},
		&_fs.File{},
	}
)

func init() {
	//set initial working directory on unix, on WASM it's done by the main package
	targetSpecificInit()
	registerHelp()

	_shell.SetNewDefaultGlobalState(func(ctx *core.Context, envPattern *core.ObjectPattern, out io.Writer) *core.GlobalState {
		return utils.Must(NewDefaultGlobalState(ctx, DefaultGlobalStateConfig{
			EnvPattern: envPattern,
			Out:        out,
		}))
	})
}

type DefaultGlobalStateConfig struct {
	EnvPattern          *core.ObjectPattern
	AllowMissingEnvVars bool
	Out                 io.Writer
	LogOut              io.Writer
}

// NewDefaultGlobalState creates a new GlobalState with the default globals.
func NewDefaultGlobalState(ctx *core.Context, conf DefaultGlobalStateConfig) (*core.GlobalState, error) {
	logOut := conf.LogOut
	var logger zerolog.Logger
	if logOut == nil { //if there is not writer for logs we log to conf.Out
		logOut = conf.Out

		consoleLogger := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.Out = logOut
			w.NoColor = !config.SHOULD_COLORIZE
			w.TimeFormat = "15:04:05"
			w.FieldsExclude = []string{"src"}
		})
		logger = zerolog.New(consoleLogger)
	} else {
		logger = zerolog.New(logOut)
	}

	logger = logger.With().Timestamp().Logger().Level(zerolog.InfoLevel)

	envNamespace, err := _env.NewEnvNamespace(ctx, conf.EnvPattern, conf.AllowMissingEnvVars)
	if err != nil {
		return nil, err
	}

	constants := map[string]core.Value{
		// constants
		core.INITIAL_WORKING_DIR_VARNAME:        core.INITIAL_WORKING_DIR_PATH,
		core.INITIAL_WORKING_DIR_PREFIX_VARNAME: core.INITIAL_WORKING_DIR_PATH_PATTERN,

		// namespaces
		"fs":       _fs.NewFsNamespace(),
		"http":     _http.NewHttpNamespace(),
		"tcp":      _net.NewTcpNamespace(),
		"dns":      _net.NewDNSnamespace(),
		"ws":       _net.NewWebsocketNamespace(),
		"s3":       _s3.NewS3namespace(),
		"chrome":   _chrome.NewChromeNamespace(),
		"localdb":  _locdb.NewLocalDbNamespace(),
		"env":      envNamespace,
		"html":     _html.NewHTMLNamespace(),
		"dom":      _dom.NewDomNamespace(),
		"sql":      _sql.NewSQLNamespace(),
		"inox":     NewInoxNamespace(),
		"inoxsh":   _shell.NewInoxshNamespace(),
		"strmanip": _strmanip.NewStrManipNnamespace(),
		"rsa":      newRSANamespace(),

		"ls": core.WrapGoFunction(_fs.ListFiles),

		// transaction
		"get_current_tx": core.ValOf(_get_current_tx),
		"Tx":             core.ValOf(core.NewTransaction),
		"start_tx":       core.ValOf(core.StartNewTransaction),

		"Error": core.ValOf(_Error),

		// resource
		"read":   core.ValOf(_readResource),
		"get":    core.ValOf(_getResource),
		"create": core.ValOf(_createResource),
		"update": core.ValOf(_updateResource),
		"delete": core.ValOf(_deleteResource),

		"serve": core.ValOf(_serve),

		// events
		"Event":       core.ValOf(_Event),
		"EventSource": core.ValOf(core.NewEventSource),

		// watch
		"watch_received_messages": core.ValOf(core.WatchReceivedMessages),
		"ValueHistory":            core.WrapGoFunction(core.NewValueHistory),
		"dynif":                   core.WrapGoFunction(core.NewDynamicIf),
		"dyncall":                 core.WrapGoFunction(core.NewDynamicCall),
		"get_system_graph":        core.WrapGoFunction(_get_system_graph),

		// send & receive values
		"sendval": core.ValOf(core.SendVal),

		// crypto
		"insecure":       newInsecure(),
		"sha256":         core.ValOf(_sha256),
		"sha384":         core.ValOf(_sha384),
		"sha512":         core.ValOf(_sha512),
		"hash_password":  core.ValOf(_hashPassword),
		"check_password": core.ValOf(_checkPassword),
		"rand":           core.ValOf(_rand),

		//encodings
		"b64":  core.ValOf(encodeBase64),
		"db64": core.ValOf(decodeBase64),

		"hex":   core.ValOf(encodeHex),
		"unhex": core.ValOf(decodeHex),

		// conversion
		"tostr":      core.ValOf(_tostr),
		"torune":     core.ValOf(_torune),
		"tobyte":     core.ValOf(_tobyte),
		"tofloat":    core.ValOf(_tofloat),
		"toint":      core.ValOf(_toint),
		"torstream":  core.ValOf(_torstream),
		"tojson":     core.ValOf(core.ToJSON),
		"topjson":    core.ValOf(core.ToPrettyJSON),
		"repr":       core.ValOf(_repr),
		"parse_repr": core.ValOf(_parse_repr),
		"parse":      core.ValOf(_parse),
		"split":      core.ValOf(_split),

		// time
		"ago":   core.ValOf(_ago),
		"now":   core.ValOf(_now),
		"sleep": core.ValOf(core.Sleep),

		// printing
		"logvals":       core.ValOf(_logvals),
		"log":           core.ValOf(_log),
		"print":         core.ValOf(_print),
		"printvals":     core.ValOf(_printvals),
		"fprint":        core.ValOf(_fprint),
		"stringify_ast": core.ValOf(_stringify_ast),
		"fmt":           core.ValOf(core.Fmt),

		// bytes & string
		"mkbytes":       core.ValOf(_mkbytes),
		"Runes":         core.ValOf(_Runes),
		"Bytes":         core.ValOf(_Bytes),
		"is_rune_space": core.ValOf(_is_rune_space),
		"Reader":        core.ValOf(_Reader),
		"RingBuffer":    core.ValOf(core.NewRingBuffer),

		// functional
		"idt":     core.WrapGoFunction(_idt),
		"map":     core.WrapGoFunction(core.Map),
		"filter":  core.WrapGoFunction(core.Filter),
		"some":    core.WrapGoFunction(core.Some),
		"all":     core.WrapGoFunction(core.All),
		"none":    core.WrapGoFunction(core.None),
		"replace": core.WrapGoFunction(_replace),
		"find":    core.WrapGoFunction(_find),
		"sort":    core.WrapGoFunction(core.Sort),

		// concurrency & execution
		"RoutineGroup": core.ValOf(core.NewRoutineGroup),
		"dynimport":    core.ValOf(_dynimport),
		"run":          core.ValOf(_run),
		"ex":           core.ValOf(_execute),
		"cancel_exec":  core.ValOf(_cancel_exec),

		// integer
		"is_even": core.ValOf(_is_even),
		"is_odd":  core.ValOf(_is_odd),

		// protocol
		"set_client_for_url":  core.ValOf(setClientForURL),
		"set_client_for_host": core.ValOf(setClientForHost),

		// other functions
		"add_ctx_data": core.ValOf(_add_ctx_data),
		"ctx_data":     core.ValOf(_ctx_data),
		"clone_val":    core.ValOf(_clone_val),
		"propnames":    core.WrapGoFunction(_propnames),

		"List":   core.ValOf(_List),
		"append": core.ValOf(core.Append),

		"typeof":    core.ValOf(_typeof),
		"url_of":    core.ValOf(_url_of),
		"id_of":     core.ValOf(core.IdOf),
		"len":       core.ValOf(_len),
		"len_range": core.ValOf(_len_range),

		"sum_options": core.ValOf(core.SumOptions),
		"mime":        core.ValOf(_http.Mime_),

		"Color": core.WrapGoFunction(_Color),

		"help": core.ValOf(_help.Help),
	}

	for k, v := range _containers.NewContainersNamespace().EntryMap() {
		constants[k] = v
	}

	state := core.NewGlobalState(ctx, constants)
	state.Out = conf.Out
	state.Logger = logger
	state.GetBaseGlobalsForImportedModule = func(ctx *core.Context, manifest *core.Manifest) (core.GlobalVariables, error) {
		importedModuleGlobals := utils.CopyMap(constants)
		env, err := _env.NewEnvNamespace(ctx, nil, conf.AllowMissingEnvVars)
		if err != nil {
			return core.GlobalVariables{}, err
		}

		importedModuleGlobals["env"] = env
		baseGlobalKeys := utils.GetMapKeys(importedModuleGlobals)
		return core.GlobalVariablesFromMap(importedModuleGlobals, baseGlobalKeys), nil
	}
	state.GetBasePatternsForImportedModule = func() (map[string]core.Pattern, map[string]*core.PatternNamespace) {
		return utils.CopyMap(core.DEFAULT_NAMED_PATTERNS), utils.CopyMap(core.DEFAULT_PATTERN_NAMESPACES)
	}

	return state, nil
}

type DefaultContextConfig struct {
	Permissions          []core.Permission
	ForbiddenPermissions []core.Permission
	Limitations          []core.Limitation
	HostResolutions      map[core.Host]core.Value
	ParentContext        *core.Context  //optional
	Filesystem           afs.Filesystem //if nil the OS filesystem is used
}

// NewDefaultState creates a new Context with the default patterns.
func NewDefaultContext(config DefaultContextConfig) (*core.Context, error) {

	ctxConfig := core.ContextConfig{
		Permissions:          config.Permissions,
		ForbiddenPermissions: config.ForbiddenPermissions,
		Limitations:          config.Limitations,
		HostResolutions:      config.HostResolutions,
		ParentContext:        config.ParentContext,
		Filesystem:           config.Filesystem,
	}

	if ctxConfig.Filesystem == nil {
		ctxConfig.Filesystem = _fs.GetOsFilesystem()
	}

	if ctxConfig.ParentContext != nil {
		if err, _ := ctxConfig.HasParentRequiredPermissions(); err != nil {
			return nil, err
		}
	}

	ctx := core.NewContext(ctxConfig)

	for k, v := range core.DEFAULT_NAMED_PATTERNS {
		ctx.AddNamedPattern(k, v)
	}

	for k, v := range core.DEFAULT_PATTERN_NAMESPACES {
		ctx.AddPatternNamespace(k, v)
	}

	return ctx, nil
}
