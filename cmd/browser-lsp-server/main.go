//go:build js

package main

import (
	"fmt"
	"io"
	"syscall/js"
	"time"

	core "github.com/inoxlang/inox/internal/core"
	lsp "github.com/inoxlang/inox/internal/lsp"
	"github.com/inoxlang/inox/internal/utils"
)

const (
	LSP_INPUT_BUFFER_SIZE  = 5_000_000
	LSP_OUTPUT_BUFFER_SIZE = 5_000_000
	OUT_PREFIX             = "[lsp server module]"
)

func main() {
	fmt.Println(OUT_PREFIX, "start")
	ctx := core.NewContext(core.ContextConfig{})
	pauseChan := make(chan struct{})

	lspInput := core.NewRingBuffer(ctx, LSP_INPUT_BUFFER_SIZE)
	lspInputWriter := utils.FnReaderWriter{
		WriteFn: func(p []byte) (n int, err error) {
			fmt.Println(OUT_PREFIX, "resume reading because we are going to write")
			select {
			case <-pauseChan:
			case <-time.After(100 * time.Millisecond):
			}
			fmt.Println(OUT_PREFIX, "write LSP input")
			return lspInput.Write(p)
		},
		ReadFn: func(p []byte) (n int, err error) {
			if lspInput.ReadableCount(ctx) == 0 {
				fmt.Println(OUT_PREFIX, "pause read call because there is nothing to read")
				pauseChan <- struct{}{}
			}
			fmt.Println(OUT_PREFIX, "read LSP input")
			return lspInput.Read(p)
		},
	}

	lspOuput := core.NewRingBuffer(ctx, LSP_OUTPUT_BUFFER_SIZE)
	registerCallbacks(lspInputWriter, lspOuput)

	fmt.Println(OUT_PREFIX, "start server")

	go lsp.StartLSPServer(lsp.LSPServerOptions{
		WASM: &lsp.WasmOptions{
			StdioInput:  lspInputWriter,
			StdioOutput: lspOuput,
			LogOutput: utils.FnWriter{
				WriteFn: func(p []byte) (n int, err error) {
					fmt.Println(OUT_PREFIX, utils.BytesAsString(p))
					return len(p), nil
				},
			},
		},
	})

	fmt.Println(OUT_PREFIX, "end of main: block with channel")

	channel := make(chan struct{})
	<-channel
}

func registerCallbacks(lspInput io.ReadWriter, lspOutput *core.RingBuffer) {
	exports := js.Global().Get("exports")
	exports.Set("write_lsp_input", js.FuncOf(func(this js.Value, args []js.Value) any {
		fmt.Println(OUT_PREFIX, "write_lsp_input() called by JS")

		s := args[0].String()
		lspInput.Write(utils.StringAsBytes(s))
		return js.ValueOf(nil)
	}))

	exports.Set("read_lsp_output", js.FuncOf(func(this js.Value, args []js.Value) any {
		fmt.Println(OUT_PREFIX, "read_lsp_output() called by JS")

		b := lspOutput.ReadableBytesCopy()
		return js.ValueOf(string(b))
	}))

	exports.Set("setup", js.FuncOf(func(this js.Value, args []js.Value) any {
		fmt.Println(OUT_PREFIX, "setup() called by JS")

		IWD := args[0].Get(core.INITIAL_WORKING_DIR_VARNAME).String()

		core.SetInitialWorkingDir(func() (string, error) {
			return IWD, nil
		})

		b := lspOutput.ReadableBytesCopy()
		return js.ValueOf(string(b))
	}))

	fmt.Println(OUT_PREFIX, "exports registered")
}
