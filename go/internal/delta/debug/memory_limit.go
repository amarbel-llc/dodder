package debug

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
)

type memoryLimit struct {
	ctx errors.Context
	*time.Ticker
	memoryLimit atomic.Uint64
	memoryInUse atomic.Uint64
	sync.Once
}

func (ml *memoryLimit) Start(ctx errors.Context) (err error) {
	ml.ctx = ctx

	ml.Ticker = time.NewTicker(time.Microsecond)
	ctx.After(errors.MakeFuncContextFromFuncNil(ml.Stop))

	var memoryLimit uint64

	if memoryLimit, err = getMemoryLimit(); err != nil {
		memoryLimit = (1500 * 1024 * 1024) // 1.5 GB
		ui.Err().Printf(
			"error getting memory limit: %s. setting to %s",
			ui.GetHumanBytesString(memoryLimit),
		)

		err = nil
		// err = errors.Wrapf(err, "could not determine memory limit")
		// return
	}

	ml.memoryLimit.Swap(memoryLimit)

	go func() {
		var memStats runtime.MemStats

		for {
			select {
			case <-ctx.Done():
				return

			case <-ml.C:
				runtime.ReadMemStats(&memStats)
				ml.memoryInUse.Swap(memStats.Alloc)

				memoryInUse := ml.memoryInUse.Load()
				memoryLimit := ml.memoryLimit.Load()

				percent := float64(memoryInUse) / float64(memoryLimit) * 100

				if percent >= 90 {
					ml.Do(ml.Terminate)
				}
			}
		}
	}()

	return err
}

func (ml *memoryLimit) Terminate() {
	memoryInUse := ml.memoryInUse.Load()
	memoryLimit := ml.memoryLimit.Load()
	percent := float64(memoryInUse) / float64(memoryLimit) * 100

	ui.Err().Printf(
		"%.2f%% memory used: %s of %s",
		percent,
		ui.GetHumanBytesString(memoryInUse),
		ui.GetHumanBytesString(memoryLimit),
	)

	defer func() {
		recover()
	}()

	errors.ContextCancelWithErrorf(ml.ctx, "10%% memory remaining")
}
