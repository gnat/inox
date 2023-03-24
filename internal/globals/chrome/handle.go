package internal

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"

	core "github.com/inox-project/inox/internal/core"
)

const (
	DEFAULT_SINGLE_ACTION_TIMEOUT = 15 * time.Second
)

type Handle struct {
	core.NoReprMixin
	allocCtx       context.Context
	cancelAllocCtx context.CancelFunc

	chromedpContext       context.Context
	cancelChromedpContext context.CancelFunc
}

func NewHandle(ctx *core.Context) (*Handle, error) {

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.Flag("headless", true),
		chromedp.UserDataDir(string(core.CreateTempdir("chrome"))),
		//chromedp.Headless,
	)

	allocCtx, cancelAllocCtx := chromedp.NewExecAllocator(ctx, opts...)

	chromedpCtx, cancel := chromedp.NewContext(
		allocCtx,
		//chromedp.WithDebugf(ctx.GetState().Logger.Printf),
	)

	handle := &Handle{
		allocCtx:       allocCtx,
		cancelAllocCtx: cancelAllocCtx,

		chromedpContext:       chromedpCtx,
		cancelChromedpContext: cancel,
	}

	if err := handle.do(ctx, chromedp.EmulateViewport(1920, 1080)); err != nil {
		return nil, err
	}

	return handle, nil
}

func (h *Handle) do(ctx *core.Context, action chromedp.Action) error {
	return chromedp.Run(h.chromedpContext,
		action,
	)
}

func (h *Handle) Nav(ctx *core.Context, u core.URL) error {
	action := chromedp.Navigate(string(u))
	return h.do(ctx, action)
}

func (h *Handle) WaitVisible(ctx *core.Context, s core.Str) error {
	action := chromedp.WaitVisible(string(s))
	return h.do(ctx, action)
}

func (h *Handle) Click(ctx *core.Context, s core.Str) error {
	action := chromedp.Click(string(s))
	return h.do(ctx, action)
}

func (h *Handle) ScreenshotPage(ctx *core.Context) (*core.ByteSlice, error) {
	var b []byte

	action := chromedp.CaptureScreenshot(&b)
	if err := h.do(ctx, action); err != nil {
		return nil, err
	}

	return &core.ByteSlice{Bytes: b, IsDataMutable: true}, nil
}

func (h *Handle) Screenshot(ctx *core.Context, sel core.Str) (*core.ByteSlice, error) {
	var b []byte

	action := chromedp.Screenshot(sel, &b)
	if err := h.do(ctx, action); err != nil {
		return nil, err
	}

	return &core.ByteSlice{Bytes: b, IsDataMutable: true}, nil
}

func (h *Handle) Close(ctx *core.Context) {
	h.cancelChromedpContext()
	h.cancelAllocCtx()
}

func (h *Handle) Prop(ctx *core.Context, name string) core.Value {
	method, ok := h.GetGoMethod(name)
	if !ok {
		panic(core.FormatErrPropertyDoesNotExist(name, h))
	}
	return method
}

func (*Handle) SetProp(ctx *core.Context, name string, value core.Value) error {
	return core.ErrCannotSetProp
}

func (h *Handle) GetGoMethod(name string) (*core.GoFunction, bool) {
	switch name {
	case "nav":
		return core.WrapGoMethod(h.Nav), true
	case "waitVisible":
		return core.WrapGoMethod(h.WaitVisible), true
	case "click":
		return core.WrapGoMethod(h.Click), true
	case "screenshot":
		return core.WrapGoMethod(h.Screenshot), true
	case "screenshotPage":
		return core.WrapGoMethod(h.ScreenshotPage), true
	case "close":
		return core.WrapGoMethod(h.Close), true
	}
	return nil, false
}

func (h *Handle) PropertyNames(ctx *core.Context) []string {
	return []string{"nav", "waitVisible", "click", "screenshotPage", "close"}
}
