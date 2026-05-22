package import_plan

import (
	"regexp"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

func MakeOmitTagsTransform(
	patterns []string,
) (ObjectTransform, error) {
	compiled := make([]*regexp.Regexp, len(patterns))

	for i, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid -omit-tags pattern: %q", pattern)
		}

		compiled[i] = re
	}

	return func(object *sku.Transacted) (bool, error) {
		var kept []string

		for tag := range object.GetMetadata().AllTags() {
			tagString := tag.String()
			matched := false

			for _, re := range compiled {
				if re.MatchString(tagString) {
					matched = true
					break
				}
			}

			if !matched {
				kept = append(kept, tagString)
			}
		}

		metadata := object.GetMetadataMutable()
		metadata.ResetTags()

		for _, tagString := range kept {
			if err := metadata.AddTagString(tagString); err != nil {
				return false, errors.Wrap(err)
			}
		}

		return true, nil
	}, nil
}
