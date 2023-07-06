package safego

import (
	"runtime/debug"
	"strings"
	"time"

	"github.com/piyushsingariya/kaku/logger"
)

const defaultRestartTimeout = 2 * time.Second

type RecoverHandler func(value interface{})

var GlobalRecoverHandler RecoverHandler

var (
	startTime time.Time
)

type Execution struct {
	f              func()
	recoverHandler RecoverHandler
	restartTimeout time.Duration
}

// Run runs a new goroutine and add panic handler (without restart)
func Run(f func()) *Execution {
	exec := Execution{
		f:              f,
		recoverHandler: GlobalRecoverHandler,
		restartTimeout: 0,
	}
	return exec.run()
}

// RunWithRestart run a new goroutine and add panic handler:
// write logs, wait 2 seconds and restart the goroutine
func RunWithRestart(f func()) *Execution {
	exec := Execution{
		f:              f,
		recoverHandler: GlobalRecoverHandler,
		restartTimeout: defaultRestartTimeout,
	}
	return exec.run()
}

func (exec *Execution) run() *Execution {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				exec.recoverHandler(r)

				if exec.restartTimeout > 0 {
					time.Sleep(exec.restartTimeout)
					exec.run()
				}
			}
		}()
		exec.f()
	}()
	return exec
}

func (exec *Execution) WithRestartTimeout(timeout time.Duration) *Execution {
	exec.restartTimeout = timeout
	return exec
}

func init() {
	GlobalRecoverHandler = func(value interface{}) {
		logger.Error("panic")
		logger.Error(value)
		logger.Error(string(debug.Stack()))
	}
}

func Recovery() {
	err := recover()
	if err != nil {
		logger.Error(err)
		// capture stacks trace
		for _, str := range strings.Split(string(debug.Stack()), "\n") {
			logger.Error(strings.ReplaceAll(str, "\t", ""))
		}
	}
	logger.Infof("Time of execution %v", time.Since(startTime).String())
}

func Insert[T any](ch chan<- T, value T) bool {
	safeInsert := false

	Run(func() {
		ch <- value
		safeInsert = true
	})

	return safeInsert
}

func Close[T any](ch chan T) {
	Run(func() {
		close(ch)
	})
}

func init() {
	startTime = time.Now()
}
