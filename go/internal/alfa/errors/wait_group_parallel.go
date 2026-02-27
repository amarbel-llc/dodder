package errors

import (
	"slices"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/_/stack_frame"
)

func MakeWaitGroupParallel() WaitGroup {
	waitGroup := &waitGroupParallel{
		lock:         &sync.Mutex{},
		inner:        &sync.WaitGroup{},
		doAfter:      make([]FuncWithStackInfo, 0),
		addStackInfo: debugBuild,
	}

	return waitGroup
}

type waitGroupParallel struct {
	lock              *sync.Mutex
	inner             *sync.WaitGroup
	errorGroupBuilder GroupBuilder
	doAfter           []FuncWithStackInfo

	addStackInfo bool

	isDone bool
}

func (waitGroup *waitGroupParallel) GetError() error {
	waitGroup.wait()

	for _, doAfter := range slices.Backward(waitGroup.doAfter) {
		errAfter := doAfter.FuncErr()
		waitGroup.errorGroupBuilder.Add(doAfter.Wrap(errAfter))
	}

	err := waitGroup.errorGroupBuilder.GetError()

	return err
}

func (waitGroup *waitGroupParallel) Do(f FuncErr) (ok bool) {
	waitGroup.lock.Lock()

	if waitGroup.isDone {
		waitGroup.lock.Unlock()
		return false
	}

	waitGroup.lock.Unlock()

	waitGroup.inner.Add(1)

	var frame stack_frame.Frame

	if waitGroup.addStackInfo {
		frame, _ = stack_frame.MakeFrame(1)
	}

	go func() {
		err := f()

		waitGroup.doneWith(&frame, err)
	}()

	return true
}

func (waitGroup *waitGroupParallel) DoAfter(f FuncErr) {
	waitGroup.lock.Lock()
	defer waitGroup.lock.Unlock()

	frame, _ := stack_frame.MakeFrame(1)

	waitGroup.doAfter = append(
		waitGroup.doAfter,
		FuncWithStackInfo{
			FuncErr: f,
			Frame:   frame,
		},
	)
}

func (waitGroup *waitGroupParallel) doneWith(
	frame *stack_frame.Frame,
	err error,
) {
	waitGroup.inner.Done()
	waitGroup.errorGroupBuilder.Add(frame.Wrap(err))
}

func (waitGroup *waitGroupParallel) wait() {
	waitGroup.inner.Wait()

	waitGroup.lock.Lock()
	defer waitGroup.lock.Unlock()

	waitGroup.isDone = true
}
