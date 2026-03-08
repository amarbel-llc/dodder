package store

import (
	"strings"
)

type discoveredReference struct {
	ObjectId string
	Alias    string
}

func parseReferenceOutput(output string) ([]discoveredReference, error) {
	var refs []discoveredReference

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var ref discoveredReference

		if idx := strings.Index(line, " = "); idx != -1 {
			ref.Alias = strings.TrimSpace(line[:idx])
			ref.ObjectId = strings.TrimSpace(line[idx+3:])
		} else {
			ref.ObjectId = line
		}

		refs = append(refs, ref)
	}

	return refs, nil
}
