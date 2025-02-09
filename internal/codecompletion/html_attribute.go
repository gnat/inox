package codecompletion

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"

	"github.com/inoxlang/inox/internal/core"
	"github.com/inoxlang/inox/internal/globals/html_ns"
	"github.com/inoxlang/inox/internal/globals/http_ns/spec"
	"github.com/inoxlang/inox/internal/mimeconsts"
	parse "github.com/inoxlang/inox/internal/parse"
	"github.com/inoxlang/inox/internal/projectserver/lsp/defines"
	"github.com/inoxlang/inox/internal/utils"
)

// This file contains completion logic for HTML and HTMX.

func findHtmlAttributeNameCompletions(ident *parse.IdentifierLiteral, parent *parse.XMLAttribute, tagName string, ancestors []parse.Node) (completions []Completion) {
	attributes, ok := html_ns.GetAllTagAttributes(tagName)
	if !ok {
		return
	}

	attrName := ident.Name

	for _, attr := range attributes {
		if !strings.HasPrefix(attr.Name, attrName) {
			continue
		}

		completions = append(completions, Completion{
			ShownString:           attr.Name,
			Value:                 attr.Name,
			Kind:                  defines.CompletionItemKindProperty,
			LabelDetail:           attr.DescriptionText(),
			MarkdownDocumentation: attr.DescriptionContent(),
		})
	}

	return
}

func findHtmlAttributeValueCompletions(
	str *parse.QuotedStringLiteral,
	parent *parse.XMLAttribute,
	tagName string,
	search completionSearch,
) (completions []Completion) {
	attrIdent, ok := parent.Name.(*parse.IdentifierLiteral)
	if !ok {
		return
	}

	attrName := attrIdent.Name
	attrValue := str.Value
	inputData := search.inputData

	set, ok := html_ns.GetAttributeValueSet(attrName, tagName)
	if ok {
		for _, attrValueData := range set.Values {
			if !strings.HasPrefix(attrValueData.Name, str.Value) {
				continue
			}

			s := string(utils.Must(json.Marshal(attrValueData.Name)))

			completions = append(completions, Completion{
				ShownString: s,
				Value:       s,
				Kind:        defines.CompletionItemKindProperty,
				LabelDetail: attrValueData.DescriptionText(),
			})
		}
		return
	}

	switch tagName {
	case "link":
		if attrName != "href" {
			break
		}
		//TODO: only add completions if rel=stylesheet

		for _, path := range inputData.StaticFileURLPaths {
			if !strings.HasSuffix(path, ".css") || !strings.HasPrefix(path, attrValue) {
				continue
			}
			completions = append(completions, Completion{
				ShownString: path,
				Value:       `"` + path + `"`,
				Kind:        defines.CompletionItemKindProperty,
			})
		}
	case "script":
		if attrName != "src" {
			break
		}
		for _, path := range inputData.StaticFileURLPaths {
			if !strings.HasSuffix(path, ".js") || !strings.HasPrefix(path, attrValue) {
				continue
			}
			completions = append(completions, Completion{
				ShownString: path,
				Value:       `"` + path + `"`,
				Kind:        defines.CompletionItemKindProperty,
			})
		}
	case "img":
		if attrName != "src" {
			break
		}
		for _, path := range inputData.StaticFileURLPaths {
			if !strings.HasPrefix(path, attrValue) {
				continue
			}

			ext := filepath.Ext(path)
			mimetype := mimeconsts.TypeByExtensionWithoutParams(ext)

			if !slices.Contains(mimeconsts.COMMON_IMAGE_CTYPES, mimetype) {
				continue
			}

			completions = append(completions, Completion{
				ShownString: path,
				Value:       `"` + path + `"`,
				Kind:        defines.CompletionItemKindProperty,
			})
		}
	}

	if inputData.ServerAPI != nil && strings.HasPrefix(attrValue, "/") && (strings.HasPrefix(attrName, "hx-") || attrName == "href") {
		//local server

		api := inputData.ServerAPI

		var endpointPaths []string
		api.ForEachHandlerModule(func(mod *core.ModulePreparationCache, endpoint *spec.ApiEndpoint, operation spec.ApiOperation) error {
			addEndpoint := false

			switch attrName {
			case "href":
				if tagName == "a" {
					addEndpoint = endpoint.HasMethodAgnosticHandler() || operation.HttpMethod() == "GET"
				}
			case "hx-get":
				addEndpoint = endpoint.HasMethodAgnosticHandler() || operation.HttpMethod() == "GET"
			case "hx-post-json":
				addEndpoint = endpoint.HasMethodAgnosticHandler() || operation.HttpMethod() == "POST"
			case "hx-patch-json":
				addEndpoint = endpoint.HasMethodAgnosticHandler() || operation.HttpMethod() == "PATCH"
			case "hx-put-json":
				addEndpoint = endpoint.HasMethodAgnosticHandler() || operation.HttpMethod() == "PUT"
			case "hx-delete":
				addEndpoint = endpoint.HasMethodAgnosticHandler() || operation.HttpMethod() == "DELETE"
			}

			if addEndpoint {
				endpointPaths = append(endpointPaths, endpoint.PathWithParams())
			}
			return nil
		})

		for _, path := range endpointPaths {
			completions = append(completions, Completion{
				ShownString: path,
				Value:       `"` + path + `"`,
				Kind:        defines.CompletionItemKindProperty,
			})
		}
	}

	switch attrName {
	case "class":
		completions = append(completions, findTailwindClassNameSuggestions(str, search)...)
	}

	return
}
