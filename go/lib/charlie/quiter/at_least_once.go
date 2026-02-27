package quiter

import "sync"

type AtLeastOnce[T any] struct {
	atLeastOnce bool
	once        sync.Once
}

func (alo *AtLeastOnce[T]) WasAtLeastOnce() bool {
	return alo.atLeastOnce
}

func (alo *AtLeastOnce[T]) Do(_ T) (err error) {
	alo.once.Do(func() { alo.atLeastOnce = true })
	return err
}
