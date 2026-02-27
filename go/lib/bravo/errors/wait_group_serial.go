package errors

import "sync"

func MakeWaitGroupSerial() WaitGroup {
	wg := &waitGroupSerial{
		do:      make([]FuncErr, 0),
		doAfter: make([]FuncWithStackInfo, 0),
	}

	return wg
}

type waitGroupSerial struct {
	lock    sync.Mutex
	do      []FuncErr
	doAfter []FuncWithStackInfo
	isDone  bool
}

func (wg *waitGroupSerial) GetError() (err error) {
	wg.lock.Lock()
	defer wg.lock.Unlock()

	wg.isDone = true

	me := MakeGroupBuilder()

	for _, f := range wg.do {
		if err = f(); err != nil {
			me.Add(Wrap(err))
			break
		}
	}

	for i := len(wg.doAfter) - 1; i >= 0; i-- {
		doAfter := wg.doAfter[i]
		err := doAfter.FuncErr()
		if err != nil {
			me.Add(doAfter.Wrap(err))
		}
	}

	err = me.GetError()

	return err
}

func (wg *waitGroupSerial) Do(f FuncErr) (d bool) {
	wg.lock.Lock()
	defer wg.lock.Unlock()

	if wg.isDone {
		return false
	}

	wg.do = append(wg.do, f)

	return true
}

func (wg *waitGroupSerial) DoAfter(f FuncErr) {
	wg.lock.Lock()
	defer wg.lock.Unlock()

	wg.do = append(wg.do, f)

	return
}
