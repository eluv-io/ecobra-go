package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/eluv-io/log-go"
	"github.com/thecodeteam/goodbye"
)

var stacktrace []byte

func init() {
	// reserve buffer for stacktrace on startup so we don't run into OOM errors
	// when generating it on shutdown...
	stacktrace = make([]byte, 1024*1024)
}

func registerSignalHandler(onExitFn func(), dumpStackOnExit bool) func() {
	ctx := context.Background()
	// begin trapping signals
	goodbye.Notify(ctx,
		syscall.SIGKILL, 1,
		syscall.SIGHUP, 0,
		syscall.SIGINT, 0,
		// syscall.SIGQUIT: 0, ==> separate handler below to print stacktrace
		// syscall.SIGTERM, 0, ==> normal exit: don't interfere
	)
	// Register shutdown handler
	goodbye.Register(func(ctx context.Context, sig os.Signal) {
		log.Warn("unexpected shutdown", "signal", sig.String(), "signal#", fmt.Sprintf("%d", sig))
		if dumpStackOnExit {
			dumpAllStacktraces()
		}
		onExitFn()
		log.Debug("shutdown hook completed")
	})

	registerStacktraceHandler()

	return func() {
		goodbye.Exit(ctx, -1)
	}
}

func registerStacktraceHandler() {
	// register specific handler for SIGQUIT to print out stacktrace, but
	// continue running.
	// Unfortunately, goodbye does not provide the possibility to continue
	// execution after the handler has executed, hence we register it directly
	// with the go runtime.
	sigc := make(chan os.Signal)
	signal.Notify(sigc, syscall.SIGQUIT)
	go func() {
		for sig := range sigc {
			log.Warn("stacktrace dump requested", "signal", sig.String(), "signal#", fmt.Sprintf("%d", sig))
			dumpAllStacktraces()
		}
	}()
}

func dumpAllStacktraces() {
	stacktrace[0] = '\n'
	length := runtime.Stack(stacktrace[1:], true)
	log.Warn("full stack dump", "stacktraces", string(stacktrace[:length]))
}
