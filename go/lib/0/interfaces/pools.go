package interfaces

type FuncRepool func()

type Pool[T any] interface {
	GetWithRepool() (T, FuncRepool)
}

type PoolPtr[T any, TPtr Ptr[T]] interface {
	Pool[TPtr]
}
