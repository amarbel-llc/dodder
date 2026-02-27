package sku

type CheckedOutWithDeletionInfo struct {
	*Transacted
	DryRun bool
}
