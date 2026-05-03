package debug

var errMemoryLimitNotSupported = newPkgError("memory limit not supported")

func getMemoryLimit() (uint64, error) {
	return 0, errMemoryLimitNotSupported
}
