package type_blobs

import "code.linenisgreat.com/dodder/go/internal/_/equality"

type UTIGroup map[string]string

func (group UTIGroup) Map() map[string]string {
	return map[string]string(group)
}

func (group *UTIGroup) Equals(b UTIGroup) bool {
	if b == nil {
		return false
	}

	if len(group.Map()) != len(b.Map()) {
		return false
	}

	if !equality.MapsOrdered(group.Map(), b.Map()) {
		return false
	}

	return true
}

func (group *UTIGroup) Merge(ct2 UTIGroup) {
	for k, v := range ct2.Map() {
		if v != "" {
			group.Map()[k] = v
		}
	}
}
