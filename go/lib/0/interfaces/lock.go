package interfaces

// LockSmith provides repo-wide filesystem mutex locking to prevent concurrent
// store mutations across processes. Not related to content locks (type/tag
// version pinning in markl.Lock).
type LockSmith interface {
	IsAcquired() bool
	Lock() error
	Unlock() error
}

type LockSmithGetter interface {
	GetLockSmith() LockSmith
}
