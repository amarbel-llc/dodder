package import_plan

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

func (plan *Plan) BloblessTypes() []string {
	seen := make(map[string]struct{})
	var result []string

	for i := range plan.Entries {
		entry := &plan.Entries[i]

		if entry.Classification != ClassificationSkipBloblessType {
			continue
		}

		typeString := entry.object.GetObjectId().String()

		if _, ok := seen[typeString]; ok {
			continue
		}

		seen[typeString] = struct{}{}
		result = append(result, typeString)
	}

	return result
}

func (plan *Plan) ResolveBloblessTypes(remapping map[string]string) {
	for i := range plan.Entries {
		entry := &plan.Entries[i]

		if entry.Classification != ClassificationErrorMissingBlob {
			continue
		}

		replacement, ok := remapping[entry.ErrorCause]
		if !ok {
			continue
		}

		if err := entry.object.GetMetadataMutable().GetTypeMutable().SetType(
			replacement,
		); err != nil {
			errors.PanicIfError(err)
		}

		if !entry.OriginalTai.Equals(entry.object.GetTai()) {
			entry.Classification = ClassificationResolveTaiReassign
		} else {
			entry.Classification = ClassificationImport
		}

		entry.ErrorCause = ""
	}

	plan.HasErrors = false

	for i := range plan.Entries {
		if plan.Entries[i].Classification.IsError() {
			plan.HasErrors = true
			break
		}
	}
}
