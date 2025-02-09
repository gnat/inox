package scan

import (
	"slices"
	"strings"

	"github.com/inoxlang/inox/internal/afs"
	"github.com/inoxlang/inox/internal/codebase/codebasescan"
	"github.com/inoxlang/inox/internal/core"
	"github.com/inoxlang/inox/internal/parse"
	"github.com/inoxlang/inox/internal/tailwind"
)

type Configuration struct {
	TopDirectories []string
	MaxFileSize    int64 //defaults to codebasescan.DEFAULT_MAX_SCANNED_INOX_FILE_SIZE
	Fast           bool  //if true the scan will be faster but will use more CPU and memory.
	InoxChunkCache *parse.ChunkCache
}

// ScanForTailwindRulesToInclude scans for Tailwind class names in 'class' attributes in Inox files.
func ScanForTailwindRulesToInclude(ctx *core.Context, fls afs.Filesystem, config Configuration) (rules []tailwind.Ruleset, _ error) {

	ruleSet := map[string]tailwind.Ruleset{}

	codebasescan.ScanCodebase(ctx, fls, codebasescan.Configuration{
		TopDirectories: config.TopDirectories,
		MaxFileSize:    config.MaxFileSize,
		Fast:           config.Fast,
		ChunkCache:     config.InoxChunkCache,
		FileHandlers: []codebasescan.FileHandler{
			func(path string, content string, n *parse.Chunk) error {
				for _, rule := range findTailwindRulesToInclude(n) {
					ruleSet[rule.Name] = rule
				}
				return nil
			},
		},
	})

	for _, rule := range ruleSet {
		rules = append(rules, rule)
	}

	slices.SortFunc(rules, func(a, b tailwind.Ruleset) int {
		return strings.Compare(a.Name, b.Name)
	})

	return
}

func findTailwindRulesToInclude(chunk *parse.Chunk) (rulesets []tailwind.Ruleset) {
	//Search for tailwind class names in 'class' attributes.

	parse.Walk(chunk, func(node, parent, scopeNode parse.Node, ancestorChain []parse.Node, after bool) (parse.TraversalAction, error) {
		attr, ok := node.(*parse.XMLAttribute)
		if !ok {
			return parse.ContinueTraversal, nil
		}

		ident, ok := attr.Name.(*parse.IdentifierLiteral)
		if !ok || ident.Name != "class" {
			return parse.Prune, nil
		}

		attrValue := ""

		switch v := attr.Value.(type) {
		case *parse.QuotedStringLiteral:
			attrValue = v.Value
		case *parse.MultilineStringLiteral:
			attrValue = v.Value
			//TODO: support string templates
		default:
			return parse.Prune, nil
		}

		classNames := strings.Split(attrValue, " ")
		for _, name := range classNames {
			name = strings.TrimSpace(name)
			ruleset, ok := tailwind.GetRuleset("." + name)
			if ok {
				rulesets = append(rulesets, ruleset)
			}
		}

		return parse.ContinueTraversal, nil
	}, nil)

	return
}
