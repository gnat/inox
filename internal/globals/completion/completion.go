package internal

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"strings"

	core "github.com/inoxlang/inox/internal/core"
	_fs "github.com/inoxlang/inox/internal/globals/fs"
	_s3 "github.com/inoxlang/inox/internal/globals/s3"
	"github.com/inoxlang/inox/internal/lsp/lsp/defines"
	"github.com/inoxlang/inox/internal/utils"

	parse "github.com/inoxlang/inox/internal/parse"
)

type Completion struct {
	ShownString   string                    `json:"shownString"`
	Value         string                    `json:"value"`
	ReplacedRange parse.SourcePositionRange `json:"replacedRange"`
	Kind          defines.CompletionItemKind
}

var (
	CONTEXT_INDEPENDENT_STMT_STARTING_KEYWORDS = []string{"if", "drop-perms", "for", "assign", "switch", "match", "return", "assert"}
)

type CompletionSearchArgs struct {
	State       *core.TreeWalkState
	Chunk       *parse.ParsedChunk
	CursorIndex int
	Mode        CompletionMode
}

type CompletionMode int

const (
	ShellCompletions CompletionMode = iota
	LspCompletions
)

func FindCompletions(args CompletionSearchArgs) []Completion {

	state := args.State
	chunk := args.Chunk
	cursorIndex := args.CursorIndex
	mode := args.Mode

	var completions []Completion
	var nodeAtCursor parse.Node
	var _parent parse.Node
	var deepestCall *parse.CallExpression
	var _ancestorChain []parse.Node

	parse.Walk(chunk.Node, func(node, parent, scopeNode parse.Node, ancestorChain []parse.Node, _ bool) (parse.TraversalAction, error) {
		span := node.Base().Span

		//if the cursor is not in the node's span we don't check the descendants of the node
		if int(span.Start) > cursorIndex || int(span.End) < cursorIndex {
			return parse.Prune, nil
		}

		if nodeAtCursor == nil || node.Base().IncludedIn(nodeAtCursor) {
			nodeAtCursor = node

			switch parent.(type) {
			case *parse.MemberExpression, *parse.IdentifierMemberExpression:
				nodeAtCursor = parent
				if len(ancestorChain) > 1 {
					_parent = ancestorChain[len(ancestorChain)-2]
				}
				_ancestorChain = utils.CopySlice(ancestorChain[:len(ancestorChain)-1])
			case *parse.PatternNamespaceMemberExpression:
				nodeAtCursor = parent
				if len(ancestorChain) > 1 {
					_parent = ancestorChain[len(ancestorChain)-2]
				}
				_ancestorChain = utils.CopySlice(ancestorChain[:len(ancestorChain)-1])
			default:
				_parent = parent
				_ancestorChain = utils.CopySlice(ancestorChain)
			}

			switch n := nodeAtCursor.(type) {
			case *parse.CallExpression:
				deepestCall = n
			}

		}

		return parse.Continue, nil
	}, nil)

	if nodeAtCursor == nil {
		return nil
	}

	switch n := nodeAtCursor.(type) {
	case *parse.PatternIdentifierLiteral:
		for name := range state.Global.Ctx.GetNamedPatterns() {
			if strings.HasPrefix(name, n.Name) {
				s := "%" + name
				completions = append(completions, Completion{
					ShownString: s,
					Value:       s,
					Kind:        defines.CompletionItemKindInterface,
				})
			}
		}
		for name := range state.Global.Ctx.GetPatternNamespaces() {
			if strings.HasPrefix(name, n.Name) {
				s := "%" + name + "."
				completions = append(completions, Completion{
					ShownString: s,
					Value:       s,
					Kind:        defines.CompletionItemKindInterface,
				})
			}
		}
	case *parse.PatternNamespaceIdentifierLiteral:
		namespace := state.Global.Ctx.ResolvePatternNamespace(n.Name)
		if namespace == nil {
			return nil
		}
		for patternName := range namespace.Patterns {
			s := "%" + n.Name + "." + patternName

			completions = append(completions, Completion{
				ShownString: s,
				Value:       s,
				Kind:        defines.CompletionItemKindInterface,
			})
		}
	case *parse.PatternNamespaceMemberExpression:
		namespace := state.Global.Ctx.ResolvePatternNamespace(n.Namespace.Name)
		if namespace == nil {
			return nil
		}
		for patternName := range namespace.Patterns {
			if strings.HasPrefix(patternName, n.MemberName.Name) {
				s := "%" + n.Namespace.Name + "." + patternName

				completions = append(completions, Completion{
					ShownString: s,
					Value:       s,
					Kind:        defines.CompletionItemKindInterface,
				})
			}
		}
	case *parse.Variable:
		var names []string
		if args.Mode == ShellCompletions {
			for name := range state.CurrentLocalScope() {
				if strings.HasPrefix(name, n.Name) {
					names = append(names, name)
				}
			}
		} else {
			scopeData, _ := state.Global.SymbolicData.GetLocalScopeData(n, _ancestorChain)
			for _, varData := range scopeData.Variables {
				if strings.HasPrefix(varData.Name, n.Name) {
					names = append(names, varData.Name)
				}
			}
		}

		for _, name := range names {
			completions = append(completions, Completion{
				ShownString: name,
				Value:       "$" + name,
				Kind:        defines.CompletionItemKindVariable,
			})
		}
	case *parse.GlobalVariable:
		state.Global.Globals.Foreach(func(name string, _ core.Value) {
			if strings.HasPrefix(name, n.Name) {
				completions = append(completions, Completion{
					ShownString: name,
					Value:       "$$" + name,
					Kind:        defines.CompletionItemKindVariable,
				})
			}
		})
	case *parse.IdentifierLiteral:
		completions = handleIdentifierAndKeywordCompletions(mode, n, deepestCall, _ancestorChain, state)
	case *parse.IdentifierMemberExpression:
		completions = handleIdentifierMemberCompletions(n, state)
	case *parse.MemberExpression:
		completions = handleMemberExpressionCompletions(n, state)
	case *parse.CallExpression: //if a call is the deepest node at cursor it means we are not in an argument
		completions = handleNewCallArgumentCompletions(n, cursorIndex, state, chunk)
	case *parse.RelativePathLiteral:
		completions = findPathCompletions(state.Global.Ctx, n.Raw)
	case *parse.AbsolutePathLiteral:
		completions = findPathCompletions(state.Global.Ctx, n.Raw)
	case *parse.URLLiteral:
		completions = findURLCompletions(state.Global.Ctx, core.URL(n.Value), _parent)
	case *parse.HostLiteral:
		completions = findHostCompletions(state.Global.Ctx, n.Value, _parent)
	case *parse.SchemeLiteral:
		completions = findHostCompletions(state.Global.Ctx, n.Name, _parent)
	}

	for i, completion := range completions {
		if completion.ReplacedRange.Span == (parse.NodeSpan{}) {
			span := nodeAtCursor.Base().Span
			completion.ReplacedRange = chunk.GetSourcePosition(span)
		}
		if completion.Kind == 0 {
			completion.Kind = defines.CompletionItemKindText
		}
		completions[i] = completion
	}

	return completions
}

func handleIdentifierAndKeywordCompletions(
	mode CompletionMode, ident *parse.IdentifierLiteral, deepestCall *parse.CallExpression,
	ancestors []parse.Node, state *core.TreeWalkState,
) []Completion {

	var completions []Completion

	if deepestCall != nil { //subcommand completions
		argIndex := -1

		for i, arg := range deepestCall.Arguments {
			if core.SamePointer(ident, arg) {
				argIndex = i
				break
			}
		}

		if argIndex >= 0 {
			calleeIdent, ok := deepestCall.Callee.(*parse.IdentifierLiteral)
			if !ok {
				return nil
			}

			subcommandIdentChain := make([]*parse.IdentifierLiteral, 0)
			for _, arg := range deepestCall.Arguments {
				idnt, ok := arg.(*parse.IdentifierLiteral)
				if !ok {
					break
				}
				subcommandIdentChain = append(subcommandIdentChain, idnt)
			}

			completionSet := make(map[Completion]bool)

			for _, perm := range state.Global.Ctx.GetGrantedPermissions() {
				cmdPerm, ok := perm.(core.CommandPermission)
				if !ok ||
					cmdPerm.CommandName.UnderlyingString() != calleeIdent.Name ||
					len(subcommandIdentChain) > len(cmdPerm.SubcommandNameChain) ||
					len(cmdPerm.SubcommandNameChain) == 0 ||
					!strings.HasPrefix(cmdPerm.SubcommandNameChain[argIndex], ident.Name) {
					continue
				}

				subcommandName := cmdPerm.SubcommandNameChain[argIndex]
				completion := Completion{
					ShownString: subcommandName,
					Value:       subcommandName,
					Kind:        defines.CompletionItemKindEnum,
				}
				if !completionSet[completion] {
					completions = append(completions, completion)
					completionSet[completion] = true
				}
			}
		}
	}

	//suggest local variables

	if mode == ShellCompletions {
		for name := range state.CurrentLocalScope() {
			if strings.HasPrefix(name, ident.Name) {
				completions = append(completions, Completion{
					ShownString: name,
					Value:       name,
					Kind:        defines.CompletionItemKindVariable,
				})
			}
		}
	} else {
		scopeData, _ := state.Global.SymbolicData.GetLocalScopeData(ident, ancestors)
		for _, varData := range scopeData.Variables {
			if strings.HasPrefix(varData.Name, ident.Name) {
				completions = append(completions, Completion{
					ShownString: varData.Name,
					Value:       varData.Name,
					Kind:        defines.CompletionItemKindVariable,
				})
			}
		}
	}

	//suggest global variables

	state.Global.Globals.Foreach(func(name string, _ core.Value) {
		if strings.HasPrefix(name, ident.Name) {
			completions = append(completions, Completion{
				ShownString: name,
				Value:       name,
				Kind:        defines.CompletionItemKindVariable,
			})
		}
	})

	parent := ancestors[len(ancestors)-1]

	//suggest context dependent keywords

	for i := len(ancestors) - 1; i >= 0; i-- {
		if parse.IsScopeContainerNode(ancestors[i]) {
			break
		}
		switch ancestors[i].(type) {
		case *parse.ForStatement:

			switch parent.(type) {
			case *parse.Block:
				for _, keyword := range []string{"break", "continue"} {
					if strings.HasPrefix(keyword, ident.Name) {
						completions = append(completions, Completion{
							ShownString: keyword,
							Value:       keyword,
							Kind:        defines.CompletionItemKindKeyword,
						})
					}
				}
			}
		case *parse.WalkStatement:

			switch parent.(type) {
			case *parse.Block:
				if strings.HasPrefix("prune", ident.Name) {
					completions = append(completions, Completion{
						ShownString: "prune",
						Value:       "prune",
						Kind:        defines.CompletionItemKindKeyword,
					})
				}
			}
		}
	}

	//suggest context independent keywords starting statements

	for _, keyword := range CONTEXT_INDEPENDENT_STMT_STARTING_KEYWORDS {

		if strings.HasPrefix(keyword, ident.Name) {
			switch parent.(type) {
			case *parse.Block, *parse.InitializationBlock, *parse.EmbeddedModule, *parse.Chunk:
				completions = append(completions, Completion{
					ShownString: keyword,
					Value:       keyword,
					Kind:        defines.CompletionItemKindKeyword,
				})
			}
		}
	}

	//suggest some keywords starting expressions

	for _, keyword := range []string{"udata", "Mapping", "concat"} {
		if strings.HasPrefix(keyword, ident.Name) {
			completions = append(completions, Completion{
				ShownString: keyword,
				Value:       keyword,
				Kind:        defines.CompletionItemKindKeyword,
			})
		}
	}

	return completions
}

func handleIdentifierMemberCompletions(n *parse.IdentifierMemberExpression, state *core.TreeWalkState) []Completion {

	curr, ok := state.Get(n.Left.Name)
	if !ok {
		return nil
	}

	buff := bytes.NewBufferString(n.Left.Name)

	//we get the next property until we reach the last property's name
	for i, propName := range n.PropertyNames {
		iprops, ok := curr.(core.IProps)
		if !ok {
			return nil
		}

		found := false
		for _, name := range iprops.PropertyNames(state.Global.Ctx) {
			if name == propName.Name {
				if i == len(n.PropertyNames)-1 { //if last
					return nil
				}
				buff.WriteRune('.')
				buff.WriteString(propName.Name)
				curr = iprops.Prop(state.Global.Ctx, name)
				found = true
				break
			}
		}

		if !found && i < len(n.PropertyNames)-1 { //if not last
			return nil
		}
	}

	s := buff.String()

	return suggestPropertyNames(s, curr, n.PropertyNames, state.Global)
}

func handleMemberExpressionCompletions(n *parse.MemberExpression, state *core.TreeWalkState) []Completion {
	ok := true
	buff := bytes.NewBufferString("")

	var exprPropertyNames = []*parse.IdentifierLiteral{n.PropertyName}
	left := n.Left

loop:
	for {
		switch l := left.(type) {
		case *parse.MemberExpression:
			left = l.Left
			exprPropertyNames = append([]*parse.IdentifierLiteral{l.PropertyName}, exprPropertyNames...)
		case *parse.GlobalVariable:
			buff.WriteString(l.Str())
			break loop
		case *parse.Variable:
			buff.WriteString(l.Str())
			break loop
		default:
			return nil
		}
	}
	var curr core.Value

	switch left := left.(type) {
	case *parse.GlobalVariable:
		if curr, ok = state.Global.Globals.CheckedGet(left.Name); !ok {
			return nil
		}
	case *parse.Variable:
		if curr, ok = state.Get(left.Name); !ok {
			return nil
		}
	}

	for i, propName := range exprPropertyNames {
		if propName == nil {
			break
		}
		iprops, ok := curr.(core.IProps)
		if !ok {
			return nil
		}
		found := false
		for _, name := range iprops.PropertyNames(state.Global.Ctx) {
			if name == propName.Name {
				buff.WriteRune('.')
				buff.WriteString(propName.Name)
				curr = iprops.Prop(state.Global.Ctx, name)
				found = true
				break
			}
		}
		if !found && i < len(exprPropertyNames)-1 { //if not last
			return nil
		}
	}

	return suggestPropertyNames(buff.String(), curr, exprPropertyNames, state.Global)
}

func suggestPropertyNames(s string, curr interface{}, exprPropertyNames []*parse.IdentifierLiteral, state *core.GlobalState) []Completion {
	var completions []Completion
	var propNames []string

	//we get all property names
	switch v := curr.(type) {
	case core.IProps:
		propNames = v.PropertyNames(state.Ctx)
	}

	isLastPropPresent := len(exprPropertyNames) > 0 && exprPropertyNames[len(exprPropertyNames)-1] != nil

	if !isLastPropPresent {
		//we suggest all property names

		for _, propName := range propNames {
			completions = append(completions, Completion{
				ShownString: s + "." + propName,
				Value:       s + "." + propName,
				Kind:        defines.CompletionItemKindProperty,
			})
		}
	} else {
		//we suggest all property names which start with the last name in the member expression

		propNamePrefix := exprPropertyNames[len(exprPropertyNames)-1].Name

		for _, propName := range propNames {

			if !strings.HasPrefix(propName, propNamePrefix) {
				continue
			}

			completions = append(completions, Completion{
				ShownString: s + "." + propName,
				Value:       s + "." + propName,
				Kind:        defines.CompletionItemKindProperty,
			})
		}
	}
	return completions
}

func handleNewCallArgumentCompletions(n *parse.CallExpression, cursorIndex int, state *core.TreeWalkState, chunk *parse.ParsedChunk) []Completion {
	var completions []Completion
	calleeIdent, ok := n.Callee.(*parse.IdentifierLiteral)
	if !ok {
		return nil
	}

	subcommandIdentChain := make([]*parse.IdentifierLiteral, 0)
	for _, arg := range n.Arguments {
		idnt, ok := arg.(*parse.IdentifierLiteral)
		if !ok {
			break
		}
		subcommandIdentChain = append(subcommandIdentChain, idnt)
	}

	completionSet := make(map[Completion]bool)

top_loop:
	for _, perm := range state.Global.Ctx.GetGrantedPermissions() {
		cmdPerm, ok := perm.(core.CommandPermission)
		if !ok ||
			cmdPerm.CommandName.UnderlyingString() != calleeIdent.Name ||
			len(subcommandIdentChain) >= len(cmdPerm.SubcommandNameChain) ||
			len(cmdPerm.SubcommandNameChain) == 0 {
			continue
		}

		if len(subcommandIdentChain) == 0 {
			name := cmdPerm.SubcommandNameChain[0]
			span := parse.NodeSpan{Start: int32(cursorIndex), End: int32(cursorIndex + 1)}

			completion := Completion{
				ShownString:   name,
				Value:         name,
				ReplacedRange: chunk.GetSourcePosition(span),
				Kind:          defines.CompletionItemKindEnum,
			}
			if !completionSet[completion] {
				completions = append(completions, completion)
				completionSet[completion] = true
			}
			continue
		}

		holeIndex := -1
		identIndex := 0

		for i, name := range cmdPerm.SubcommandNameChain {
			if name != subcommandIdentChain[identIndex].Name {
				if holeIndex >= 0 {
					continue top_loop
				}
				holeIndex = i
			} else {
				if identIndex == len(subcommandIdentChain)-1 {
					if holeIndex < 0 {
						holeIndex = i + 1
					}
					break
				}
				identIndex++
			}
		}
		subcommandName := cmdPerm.SubcommandNameChain[holeIndex]
		span := parse.NodeSpan{Start: int32(cursorIndex), End: int32(cursorIndex + 1)}

		completion := Completion{
			ShownString:   subcommandName,
			Value:         subcommandName,
			ReplacedRange: chunk.GetSourcePosition(span),
			Kind:          defines.CompletionItemKindEnum,
		}
		if !completionSet[completion] {
			completions = append(completions, completion)
			completionSet[completion] = true
		}
	}
	return completions
}

func findPathCompletions(ctx *core.Context, pth string) []Completion {
	var completions []Completion

	dir := path.Dir(pth)
	base := path.Base(pth)

	if core.Path(pth).IsDirPath() {
		base = ""
	}

	entries, err := _fs.ListFiles(ctx, core.Path(dir+"/"))
	if err != nil {
		return nil
	}

	for _, e := range entries {
		name := string(e.Name)
		if strings.HasPrefix(name, base) {
			pth := path.Join(dir, name)

			if !parse.HasPathLikeStart(pth) {
				pth = "./" + pth
			}

			stat, _ := os.Stat(pth)
			if stat.IsDir() {
				pth += "/"
			}

			completions = append(completions, Completion{
				ShownString: name,
				Value:       pth,
				Kind:        defines.CompletionItemKindConstant,
			})
		}
	}

	return completions
}

func findURLCompletions(ctx *core.Context, u core.URL, parent parse.Node) []Completion {
	var completions []Completion

	urlString := string(u)

	if call, ok := parent.(*parse.CallExpression); ok {

		var S3_FNS = []string{"get", "delete", "ls"}

		if memb, ok := call.Callee.(*parse.IdentifierMemberExpression); ok &&
			memb.Left.Name == "s3" &&
			len(memb.PropertyNames) == 1 &&
			utils.SliceContains(S3_FNS, memb.PropertyNames[0].Name) &&
			strings.Contains(urlString, "/") {

			objects, err := _s3.S3List(ctx, u)
			if err == nil {
				prefix := urlString[:strings.LastIndex(urlString, "/")+1]
				for _, obj := range objects {

					val := prefix + filepath.Base(obj.Key)
					if strings.HasSuffix(obj.Key, "/") {
						val += "/"
					}

					completions = append(completions, Completion{
						ShownString: obj.Key,
						Value:       val,
						Kind:        defines.CompletionItemKindConstant,
					})
				}
			}
		}
	}

	return completions
}

func findHostCompletions(ctx *core.Context, prefix string, parent parse.Node) []Completion {
	var completions []Completion

	allData := ctx.GetAllHostResolutionData()

	for host := range allData {
		hostStr := string(host)
		if strings.HasPrefix(hostStr, prefix) {
			completions = append(completions, Completion{
				ShownString: hostStr,
				Value:       hostStr,
				Kind:        defines.CompletionItemKindConstant,
			})
		}
	}

	{ //localhost
		scheme, realHost, ok := strings.Cut(prefix, "://")

		var schemes = []string{"http", "https", "file", "ws", "wss"}

		if ok && utils.SliceContains(schemes, scheme) && len(realHost) > 0 && strings.HasPrefix("localhost", realHost) {
			s := strings.Replace(prefix, realHost, "localhost", 1)
			completions = append(completions, Completion{
				ShownString: s,
				Value:       s,
				Kind:        defines.CompletionItemKindConstant,
			})
		}

	}

	return completions
}
