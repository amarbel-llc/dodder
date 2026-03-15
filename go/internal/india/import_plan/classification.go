package import_plan

type Classification string

const (
	ClassificationImport             Classification = "import"
	ClassificationSkipExists         Classification = "skip-exists"
	ClassificationSkipDedup          Classification = "skip-dedup"
	ClassificationSkipBloblessType   Classification = "skip-blobless-type"
	ClassificationResolveTaiReassign Classification = "resolve-tai-reassign"
	ClassificationErrorMissingBlob   Classification = "error-missing-blob"
)

func (c Classification) IsCommittable() bool {
	return c == ClassificationImport || c == ClassificationResolveTaiReassign
}

func (c Classification) IsError() bool {
	return c == ClassificationErrorMissingBlob
}

func (c Classification) IsSkip() bool {
	return c == ClassificationSkipExists ||
		c == ClassificationSkipDedup ||
		c == ClassificationSkipBloblessType
}
