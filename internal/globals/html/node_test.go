package internal

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	core "github.com/inox-project/inox/internal/core"
)

func TestHTMLRender(t *testing.T) {
	nodeHtml := "<div><span>a</span></div>"
	node, _ := ParseSingleNodeHTML(nodeHtml)
	ctx := core.NewContext(core.ContextConfig{})
	buf := bytes.NewBuffer(nil)
	n, err := node.Render(ctx, buf, core.RenderingInput{Mime: core.HTML_CTYPE})
	assert.NoError(t, err)
	assert.Equal(t, len(nodeHtml), n)

	assert.Equal(t, nodeHtml, buf.String())
}
