package interfaces

type LockSmith interface {
	IsAcquired() bool
	Lock() error
	Unlock() error
}

type LockSmithGetter interface {
	GetLockSmith() LockSmith
}
