package inoxd

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode"

	"github.com/gorilla/websocket"
	"github.com/inoxlang/inox/internal/inoxd/cloud/cloudproxy"
	"github.com/inoxlang/inox/internal/inoxd/crypto"
	"github.com/inoxlang/inox/internal/inoxd/systemd/unitenv"
	"github.com/inoxlang/inox/internal/project_server"
	"github.com/inoxlang/inox/internal/utils"
	"github.com/inoxlang/inox/internal/utils/processutils"
	"github.com/rs/zerolog"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/stretchr/testify/assert"
)

func TestDaemonCloudMode(t *testing.T) {
	//this test suite require inox to be in /usr/local/go/bin.

	setup := func() (context.Context, context.CancelFunc, string, io.Writer) {
		tmpDir := t.TempDir()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		keyset := crypto.GenerateRandomInoxdMasterKeyset()
		save, ok := os.LookupEnv(unitenv.INOXD_MASTER_KEYSET_ENV_VARNAME)
		if ok {
			defer os.Setenv(unitenv.INOXD_MASTER_KEYSET_ENV_VARNAME, save)
		}
		os.Setenv(unitenv.INOXD_MASTER_KEYSET_ENV_VARNAME, string(keyset))

		outputBuf := bytes.NewBuffer(nil)
		var lock sync.Mutex
		writer := utils.FnWriter{
			WriteFn: func(p []byte) (n int, err error) {
				lock.Lock()
				defer lock.Unlock()
				return outputBuf.Write(p)
			},
		}
		return ctx, cancel, tmpDir, writer
	}

	t.Run("base case", func(t *testing.T) {

		ctx, cancel, tmpDir, writer := setup()
		defer cancel()

		var helloAckReceived atomic.Bool

		//logger hook used to check that the hello ack has been received.
		hook := zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, message string) {
			//fmt.Println(message)
			if strings.Contains(message, "ack received on connection to cloud-proxy") {
				helloAckReceived.Store(true)
			}
		})
		logger := zerolog.New(writer).Hook(hook)

		go Inoxd(InoxdArgs{
			Config: DaemonConfig{
				InoxCloud:      true,
				InoxBinaryPath: "inox",
			},
			Logger: logger,
			GoCtx:  ctx,

			DoNotUseCgroups: true,
			TestOnlyProxyConfig: &cloudproxy.CloudProxyConfig{
				CloudDataDir: tmpDir,
				Port:         6000,
			},
		})

		//wait for the connection between inoxd and the cloud-proxy to be established.
		time.Sleep(1 * time.Second)

		assert.True(t, helloAckReceived.Load())
	})

	t.Run("killing the cloud proxy process once should not cause any issue", func(t *testing.T) {
		//this tests required inox to be in /usr/local/go/bin.

		ctx, cancel, tmpDir, writer := setup()
		defer cancel()

		var helloAckCount atomic.Int32
		var proxyPid atomic.Int32

		//logger hook used to check that the hello ack has been received and to get the PID.
		hook := zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, message string) {
			if strings.Contains(message, "ack received on connection to cloud-proxy") {
				helloAckCount.Add(1)
			}

			//detect logging of the PID.
			p := reflect.ValueOf(e).Elem().FieldByName("buf").Bytes()
			jsonFieldName := []byte(processutils.NEW_PROCESS_PID_LOG_FIELD_NAME + `":`)
			fieldIndex := bytes.Index(p, jsonFieldName)

			if fieldIndex >= 0 {
				pidIndex := fieldIndex + len(jsonFieldName)
				i := pidIndex

				for i < len(p) {
					r := rune(p[i])
					if !unicode.IsDigit(r) {
						break
					}

					i++
				}

				if i == len(p)-1 {
					i++
				}

				pid, err := strconv.Atoi(string(p[pidIndex:i]))
				if err != nil {
					assert.Fail(t, err.Error())
				}
				proxyPid.Store(int32(pid))
			}
		})
		logger := zerolog.New(writer).Hook(hook)

		go Inoxd(InoxdArgs{
			Config: DaemonConfig{
				InoxCloud:      true,
				InoxBinaryPath: "inox",
			},
			Logger: logger,
			GoCtx:  ctx,

			DoNotUseCgroups: true,
			TestOnlyProxyConfig: &cloudproxy.CloudProxyConfig{
				CloudDataDir: tmpDir,
				Port:         6000,
			},
		})

		//wait for the connection between inoxd and the cloud-proxy to be established.
		time.Sleep(time.Second)

		assert.EqualValues(t, 1, helloAckCount.Load())

		pid := proxyPid.Load()
		if !assert.NotZero(t, pid) {
			return
		}

		//kill the process and wait for a new connection to be established.
		process := utils.Must(process.NewProcess(pid))
		process.Kill()
		time.Sleep(time.Second)

		assert.EqualValues(t, 2, helloAckCount.Load())
	})
}

func TestDaemonDisabledCloudMode(t *testing.T) {
	//this test suite require inox to be in /usr/local/go/bin.

	dialer := *websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	setup := func() (context.Context, context.CancelFunc, string, io.Writer) {
		tmpDir := t.TempDir()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		keyset := crypto.GenerateRandomInoxdMasterKeyset()
		save, ok := os.LookupEnv(unitenv.INOXD_MASTER_KEYSET_ENV_VARNAME)
		if ok {
			defer os.Setenv(unitenv.INOXD_MASTER_KEYSET_ENV_VARNAME, save)
		}
		os.Setenv(unitenv.INOXD_MASTER_KEYSET_ENV_VARNAME, string(keyset))

		outputBuf := bytes.NewBuffer(nil)
		var lock sync.Mutex
		writer := utils.FnWriter{
			WriteFn: func(p []byte) (n int, err error) {
				lock.Lock()
				defer lock.Unlock()
				return outputBuf.Write(p)
			},
		}
		return ctx, cancel, tmpDir, writer
	}

	t.Run("base case", func(t *testing.T) {
		ctx, cancel, tmpDir, writer := setup()
		defer func() {
			x := recover()
			fmt.Println(x)

			fmt.Println("cancel")
			cancel()
			fmt.Println("cancelled")

			time.Sleep(time.Second)
		}()

		var sessionCreatedOnServer atomic.Bool

		hook := zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, message string) {
			fmt.Println(message)
			if strings.Contains(message, "new session at 127.0.0.1") {
				sessionCreatedOnServer.Store(true)
			}
		})
		logger := zerolog.New(writer).Hook(hook)

		go Inoxd(InoxdArgs{
			Config: DaemonConfig{
				InoxCloud:      false,
				InoxBinaryPath: "inox",
				Server: project_server.IndividualServerConfig{
					ProjectsDir: tmpDir,
					Port:        6000,
				},
			},
			Logger: logger,
			GoCtx:  ctx,

			DoNotUseCgroups: true,
		})

		//wait for the connection between inoxd and the cloud-proxy to be established.
		time.Sleep(time.Second)

		c, _, err := dialer.Dial("wss://localhost:6000", nil)
		if !assert.NoError(t, err, "failed to connect") {
			return
		}
		c.Close()

		//assert.True(t, sessionCreatedOnServer.Load())
		fmt.Println("ok")
	})

	t.Run("killing the project-server process once should not cause any issue", func(t *testing.T) {
		t.Skip()
		//this tests required inox to be in /usr/local/go/bin.

		ctx, cancel, tmpDir, writer := setup()
		defer func() {
			cancel()
			time.Sleep(time.Second)
		}()

		var sessionCreatedOnServer atomic.Int32
		var projectServerPid atomic.Int32

		//logger hook used to check that the hello ack has been received and to get the PID.
		hook := zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, message string) {
			if strings.Contains(message, "new session at 127.0.0.1") {
				sessionCreatedOnServer.Add(1)
			}

			//detect logging of the PID.
			p := reflect.ValueOf(e).Elem().FieldByName("buf").Bytes()
			jsonFieldName := []byte(processutils.NEW_PROCESS_PID_LOG_FIELD_NAME + `":`)
			fieldIndex := bytes.Index(p, jsonFieldName)

			if fieldIndex >= 0 {
				pidIndex := fieldIndex + len(jsonFieldName)
				i := pidIndex

				for i < len(p) {
					r := rune(p[i])
					if !unicode.IsDigit(r) {
						break
					}

					i++
				}

				if i == len(p)-1 {
					i++
				}

				pid, err := strconv.Atoi(string(p[pidIndex:i]))
				if err != nil {
					assert.Fail(t, err.Error())
				}
				projectServerPid.Store(int32(pid))
			}
		})
		logger := zerolog.New(writer).Hook(hook)

		go Inoxd(InoxdArgs{
			Config: DaemonConfig{
				InoxCloud:      false,
				InoxBinaryPath: "inox",
				Server: project_server.IndividualServerConfig{
					ProjectsDir: tmpDir,
					Port:        6000,
				},
			},
			Logger: logger,
			GoCtx:  ctx,

			DoNotUseCgroups: true,
		})

		//wait for the connection between inoxd and the cloud-proxy to be established.
		time.Sleep(time.Second)

		c, _, err := dialer.Dial("wss://localhost:6000", nil)
		if !assert.NoError(t, err) {
			return
		}
		c.Close()

		pid := projectServerPid.Load()
		if !assert.NotZero(t, pid) {
			return
		}

		//kill the process and wait for a new connection to be established.
		process := utils.Must(process.NewProcess(pid))
		process.Kill()
		time.Sleep(time.Second)

		c, _, err = dialer.Dial("wss://localhost:6000", nil)

		if !assert.NoError(t, err, "failed to connect") {
			return
		}
		c.Close()
	})
}
