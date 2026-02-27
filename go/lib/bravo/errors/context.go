package errors

import (
	ConTeXT "context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/_/stack_frame"
	"golang.org/x/xerrors"
)

type Context interface {
	interfaces.ActiveContext
	CauseWithStackFrames() (error, []stack_frame.Frame)
	Run(func(Context)) error

	// TODO extricate from *context and turn into generic function
	SetCancelOnSignals(signals ...os.Signal)
}

// TODO maybe consider adding a target error that is used to determine whether a
// stack trace is printed?
type context struct {
	ConTeXT.Context

	stateLock sync.Mutex
	state     interfaces.ContextState

	stackFramesCancel []stack_frame.Frame
	lockCancel        sync.Mutex

	funcCancel ConTeXT.CancelCauseFunc
	funcRun    func(Context)

	signals chan os.Signal

	lockRun       sync.Mutex
	lockConc      sync.Mutex
	doAfter       []FuncWithStackInfo
	doAfterErrors []error // TODO expose and use

	retriesDisabled bool
}

func MakeContextDefault() *context {
	return MakeContext(ConTeXT.Background())
}

func MakeContext(in ConTeXT.Context) *context {
	ctx, cancel := ConTeXT.WithCancelCause(in)

	return &context{
		Context:    ctx,
		funcCancel: cancel,
		signals:    make(chan os.Signal, 1),
	}
}

func (ctx *context) Cause() error {
	if err := ConTeXT.Cause(ctx.Context); err != nil {
		switch err {
		case interfaces.ContextStateSucceeded, interfaces.ContextStateAborted:
			return nil

		default:
			return err
		}
	}

	return nil
}

func (ctx *context) CauseWithStackFrames() (error, []stack_frame.Frame) {
	if err := ctx.Cause(); err != nil {
		return err, ctx.stackFramesCancel
	} else {
		return nil, nil
	}
}

func (ctx *context) GetState() interfaces.ContextState {
	select {
	default:
		return interfaces.ContextStateStarted

	case <-ctx.Done():
		// TODO identify the right terminal state
		return interfaces.ContextStateSucceeded
	}
}

// TODO extricate from *context and turn into generic function
func (ctx *context) SetCancelOnSignals(signals ...os.Signal) {
	signal.Notify(ctx.signals, signals...)
}

func (ctx *context) Run(funcRun func(Context)) error {
	if !ctx.lockRun.TryLock() {
		return ErrorWithStackf(
			"Context.Run called before previous run completed.",
		)
	}

	defer ctx.lockRun.Unlock()

	defer ctx.runAfter()

	ctx.funcRun = funcRun

	// TODO extricate from *context and turn into generic function
	go func() {
		defer signal.Stop(ctx.signals)

		select {
		case <-ctx.Done():
		case sig := <-ctx.signals:
			ctx.cancel(Signal{Signal: sig})
		}
	}()

	for ctx.runRetry() {
	}

	return ctx.Cause()
}

func (ctx *context) runRetry() (shouldRetry bool) {
	defer func() {
		if r := recover(); r != nil {
			if r == errContextRetry {
				shouldRetry = true
				return
			}

			switch err := r.(type) {
			case interfaces.ContextState:

			case string:
				{
					err := xerrors.Errorf("%s", err)
					ctx.captureCancelStackFramesIfNecessary(2, err)
					ctx.cancel(err)
				}

			case error:
				var runtimeError runtime.Error
				var stackFrameError *stack_frame.ErrorTree

				if As(err, &runtimeError) {
					ctx.captureCancelStackFramesIfNecessary(2, err)
					ctx.cancel(err)
				} else if As(err, &stackFrameError) {
					ctx.captureCancelStackFramesIfNecessary(2, err)
					ctx.cancel(err)
				} else {
					ctx.cancel(err)
				}

			default:
				ctx.captureCancelStackFramesIfNecessary(2, nil)
				ctx.cancel(xerrors.Errorf("context failed: %w", err))
			}
		} else {
			ctx.cancel(interfaces.ContextStateSucceeded)
		}
	}()

	ctx.funcRun(ctx)

	return shouldRetry
}

func (ctx *context) runAfter() {
	for i := len(ctx.doAfter) - 1; i >= 0; i-- {
		doAfter := ctx.doAfter[i]
		err := doAfter.FuncErr()
		if err != nil {
			ctx.doAfterErrors = append(
				ctx.doAfterErrors,
				doAfter.Wrap(err),
			)
		}
	}
}

func (ctx *context) retry() {
	panic(errContextRetry)
}

func (ctx *context) cancel(err error) {
	var retryable interfaces.ErrorRetryable

	if !ctx.retriesDisabled && As(err, &retryable) {
		retryable.Recover(
			ctx,
			ctx.retry,
			func(err error) {
				ctx.Cancel(errContextRetryAborted{underlying: err})
			})
	} else {
		ctx.funcCancel(err)
	}
}

//go:noinline
func (ctx *context) after(skip int, f func() error) {
	ctx.lockConc.Lock()
	defer ctx.lockConc.Unlock()

	frame, _ := stack_frame.MakeFrame(skip + 1)

	ctx.doAfter = append(
		ctx.doAfter,
		FuncWithStackInfo{
			FuncErr: f,
			Frame:   frame,
		},
	)
}

//go:noinline
func (ctx *context) After(f interfaces.FuncActiveContext) {
	ctx.after(1, func() error { return f(ctx) })
}

//go:noinline
func ContextCloseAfter(ctx interfaces.ActiveContext, closer io.Closer) {
	ctx.After(MakeFuncContextFromFuncErr(closer.Close))
}

//go:noinline
func ContextFlushAfter(ctx interfaces.ActiveContext, closer io.Closer) {
	ctx.After(MakeFuncContextFromFuncErr(closer.Close))
}

// `Must` executes a function even if the context has been cancelled. If the
// function returns an error, `Must` cancels the context and offers a heartbeat
// to
// panic. It is meant for defers that must be executed, like closing files,
// flushing buffers, releasing locks.
//
//go:noinline
func (ctx *context) Must(f interfaces.FuncActiveContext) {
	defer ContextContinueOrPanic(ctx)

	if err := f(ctx); err != nil {
		ctx.cancel(WrapSkip(1, err))
	}
}

//go:noinline
func (ctx *context) Cancel(err error) {
	defer ContextContinueOrPanic(ctx)

	if err == nil {
		ctx.cancel(interfaces.ContextStateAborted)
		return
	}

	// TODO figure out why this needs to be 2
	ctx.captureCancelStackFramesIfNecessary(2, err)
	ctx.cancel(err)
}

// TODO add interface for adding stack frames to the cancellation error
//
//go:noinline
func (ctx *context) captureCancelStackFramesIfNecessary(skip int, err error) {
	if !debugBuild {
		return
	}

	switch err {
	case interfaces.ContextStateSucceeded, interfaces.ContextStateAborted:
		return
	}

	ctx.lockCancel.Lock()
	defer ctx.lockCancel.Unlock()

	defer func() {
		// if stack_frame.MakeFrames panics, we don't want that to take anything
		// else down
		recover()
	}()

	ctx.stackFramesCancel = stack_frame.MakeFrames(1+skip, 16)
}

//   __  __           _
//  |  \/  |_   _ ___| |_
//  | |\/| | | | / __| __|
//  | |  | | |_| \__ \ |_
//  |_|  |_|\__,_|___/\__|
//

func ContextMustClose(ctx interfaces.ActiveContext, closer io.Closer) {
	defer ContextContinueOrPanic(ctx)
	ctx.Must(MakeFuncContextFromFuncErr(closer.Close))
}

func ContextMustFlush(ctx interfaces.ActiveContext, flusher Flusher) {
	defer ContextContinueOrPanic(ctx)
	ctx.Must(MakeFuncContextFromFuncErr(flusher.Flush))
}

func ContextContinueOrPanic(ctx interfaces.ActiveContext) {
	if state := ctx.GetState(); state.IsComplete() {
		panic(state)
	}
}

//   ____  _                   _
//  / ___|(_) __ _ _ __   __ _| |___
//  \___ \| |/ _` | '_ \ / _` | / __|
//   ___) | | (_| | | | | (_| | \__ \
//  |____/|_|\__, |_| |_|\__,_|_|___/
//           |___/

func ContextSetCancelOnSIGTERM(ctx Context) {
	ctx.SetCancelOnSignals(syscall.SIGTERM)
}

func ContextSetCancelOnSIGINT(ctx Context) {
	ctx.SetCancelOnSignals(syscall.SIGINT)
}

func ContextSetCancelOnSIGHUP(ctx Context) {
	ctx.SetCancelOnSignals(syscall.SIGHUP)
}

//    ____                     _
//   / ___|__ _ _ __   ___ ___| |___
//  | |   / _` | '_ \ / __/ _ \ / __|
//  | |__| (_| | | | | (_|  __/ \__ \
//   \____\__,_|_| |_|\___\___|_|___/
//

func ContextCancelWith499ClientClosedRequest(ctx interfaces.ActiveContext) {
	ctx.Cancel(Err499ClientClosedRequest)
}

func ContextCancelWithError(ctx interfaces.ActiveContext, err error) {
	defer ContextContinueOrPanic(ctx)
	ctx.Cancel(WrapSkip(1, err))
}

func ContextCancelWithErrorAndFormat(
	ctx interfaces.ActiveContext,
	err error,
	format string,
	values ...any,
) {
	defer ContextContinueOrPanic(ctx)
	ctx.Cancel(
		WrapSkip(
			1,
			fmt.Errorf(
				"%s: "+format,
				append([]any{err.Error()}, values...)...,
			),
		),
	)
}

func ContextCancelWithErrorf(
	ctx interfaces.ActiveContext,
	format string,
	values ...any,
) {
	defer ContextContinueOrPanic(ctx)
	ctx.Cancel(WrapSkip(1, fmt.Errorf(format, values...)))
}

func ContextCancelWithBadRequestError(ctx interfaces.ActiveContext, err error) {
	defer ContextContinueOrPanic(ctx)
	ctx.Cancel(BadRequest(err))
}

func ContextCancelWithBadRequestf(
	ctx interfaces.ActiveContext,
	format string,
	values ...any,
) {
	defer ContextContinueOrPanic(ctx)
	ctx.Cancel(BadRequestf(format, values...))
}

func CancelWithNotImplemented(ctx interfaces.ActiveContext) {
	defer ContextContinueOrPanic(ctx)
	ctx.Cancel(WithoutStack(Err501NotImplemented))
}

//   ____                    _
//  |  _ \ _   _ _ __  _ __ (_)_ __   __ _
//  | |_) | | | | '_ \| '_ \| | '_ \ / _` |
//  |  _ <| |_| | | | | | | | | | | | (_| |
//  |_| \_\\__,_|_| |_|_| |_|_|_| |_|\__, |
//                                   |___/

func RunContextWithPrintTicker(
	context Context,
	runFunc func(Context),
	printFunc func(time.Time),
	duration time.Duration,
) (err error) {
	if err = context.Run(
		func(ctx Context) {
			ticker := time.NewTicker(duration)
			ctx.After(MakeFuncContextFromFuncNil(ticker.Stop))

			go func() {
				for {
					select {
					case <-ctx.Done():
						return

					case t := <-ticker.C:
						printFunc(t)
					}
				}
			}()

			runFunc(ctx)
		},
	); err != nil {
		_, frames := context.CauseWithStackFrames()
		err = stack_frame.MakeErrorTreeOrErr(err, frames...)
		return err
	}

	return err
}

func RunChildContextWithPrintTicker(
	parentContext Context,
	runFunc func(Context),
	printFunc func(time.Time),
	duration time.Duration,
) (err error) {
	return RunContextWithPrintTicker(
		MakeContext(parentContext),
		runFunc,
		printFunc,
		duration,
	)
}
