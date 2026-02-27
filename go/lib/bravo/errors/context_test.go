package errors

import (
	ConTeXT "context"
	"syscall"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

func TestContextCancelled(t *testing.T) {
	ctx := MakeContext(ConTeXT.Background())

	var must1, must2, after1 bool

	if err := ctx.Run(
		func(ctx Context) {
			defer func() {
				t.Log("defer1")

				if actual := recover(); actual != nil {
					t.Log("recover")

					expected := interfaces.ContextStateSucceeded

					if actual != expected {
						t.Errorf("expected recover to be %q, but was %q", expected, actual)
					}
				}
			}()

			defer ctx.Must(func(interfaces.ActiveContext) error {
				t.Log("must1")
				must1 = true
				return nil
			})

			defer ctx.Must(func(interfaces.ActiveContext) error {
				t.Log("must2")
				must2 = true
				return nil
			})

			ctx.After(MakeFuncContextFromFuncErr(func() error {
				after1 = true
				return nil
			}))

			ctx.Cancel(nil)
			t.Errorf("expected to not get here")
		},
	); err != nil {
		t.Errorf("expected no error but got: %s", err)
	}

	if !must1 || !must2 || !after1 {
		t.Errorf("expected all must and after functions to execute")
	}
}

type errTestRecover struct {
	shouldFail bool
}

func (errTestRecover) Error() string {
	return "test recover error"
}

func (err errTestRecover) Recover(
	ctx interfaces.ActiveContext,
	retry interfaces.FuncRetry,
	abort interfaces.FuncRetryAborted,
) {
	if err.shouldFail {
		abort(BadRequestf("bad request"))
	} else {
		retry()
	}
}

func TestContextCancelledRetry(t *testing.T) {
	ctx := MakeContext(ConTeXT.Background())

	tryCount := 0

	if err := ctx.Run(
		func(ctx Context) {
			t.Logf("try count: %d", tryCount)

			if tryCount == 0 {
				tryCount++
				ctx.Cancel(errTestRecover{})
			}

			tryCount++
		},
	); err != nil {
		t.Errorf("expected no error but got: %s", err)
	}

	if tryCount != 2 {
		t.Errorf("expected try count 2 but got: %d", tryCount)
	}
}

func TestContextCancelledRetryFailed(t *testing.T) {
	ctx := MakeContext(ConTeXT.Background())

	tryCount := 0

	expected := errContextRetryAborted{}

	if err := ctx.Run(
		func(ctx Context) {
			t.Logf("try count: %d", tryCount)

			if tryCount == 0 {
				tryCount++
				ctx.Cancel(errTestRecover{shouldFail: true})
			}

			tryCount++
		},
	); err == nil {
		t.Errorf("expected an error")
	} else if !Is(err, expected) {
		t.Errorf("expected %T but got %q", expected, err)
	}

	if tryCount != 1 {
		t.Errorf("expected try count 2 but got: %d", tryCount)
	}
}

func TestContextSignal(t *testing.T) {
	ctx := MakeContext(ConTeXT.Background())
	ContextSetCancelOnSIGHUP(ctx)

	cont := make(chan struct{})

	go func() {
		if err := ctx.Run(
			func(ctx Context) {
				child := MakeContext(ctx)

				if err := child.Run(
					func(ctx Context) {
						<-ctx.Done()
						cont <- struct{}{}
					},
				); err == nil {
					t.Errorf("expected signal error but got none")
				}
			},
		); err == nil {
			t.Errorf("expected signal error but got none")
		}
	}()

	ctx.signals <- syscall.SIGHUP
	<-cont
}
