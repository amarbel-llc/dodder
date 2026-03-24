package mcp_dodder

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"code.linenisgreat.com/dodder/go/lib/charlie/expansion"
)

type typeSummary struct {
	ObjectId    string   `json:"object-id"`
	Date        string   `json:"date"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	ResourceURI string   `json:"resource-uri"`
}

type typeIndex struct {
	bridge Bridge
	once   sync.Once
	words  map[string][]typeSummary
	err    error
}

func makeTypeIndex(bridge Bridge) *typeIndex {
	return &typeIndex{bridge: bridge}
}

func (idx *typeIndex) ensureBuilt() error {
	idx.once.Do(func() { idx.err = idx.build() })
	return idx.err
}

func (idx *typeIndex) build() error {
	result, err := idx.bridge.RunCommand(
		context.Background(),
		"show",
		[]string{"-format", "json", ":t"},
		500_000,
	)
	if err != nil {
		return err
	}

	idx.words = make(map[string][]typeSummary)

	for line := range strings.SplitSeq(result.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var obj struct {
			ObjectId    string   `json:"object-id"`
			Date        string   `json:"date"`
			Description string   `json:"description"`
			Tags        []string `json:"tags"`
		}

		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		summary := typeSummary{
			ObjectId:    obj.ObjectId,
			Date:        obj.Date,
			Description: obj.Description,
			Tags:        obj.Tags,
			ResourceURI: "dodder://types/" + strings.TrimPrefix(obj.ObjectId, "!"),
		}

		seen := make(map[string]bool)

		addWords := func(source string) {
			expansion.ExpanderRight.Expand(source)(func(word string) bool {
				word = strings.ToLower(word)
				if word == "" || seen[word] {
					return true
				}
				seen[word] = true
				idx.words[word] = append(idx.words[word], summary)
				return true
			})
		}

		addWords(strings.TrimPrefix(obj.ObjectId, "!"))

		for word := range strings.FieldsSeq(obj.Description) {
			addWords(strings.ToLower(word))
		}

		// Skip repo signature artifacts (migration data issue where
		// pubkey/sig metadata was misinterpreted as tags)
		for _, tag := range obj.Tags {
			if strings.HasPrefix(tag, "-repo") {
				continue
			}
			addWords(strings.TrimPrefix(tag, "%"))
		}
	}

	return nil
}

func (idx *typeIndex) sortedWords() []string {
	words := make([]string, 0, len(idx.words))
	for w := range idx.words {
		words = append(words, w)
	}
	sort.Strings(words)
	return words
}

func (idx *typeIndex) query(queryWords []string) []typeSummary {
	seen := make(map[string]bool)
	var results []typeSummary

	for _, qw := range queryWords {
		qw = strings.ToLower(qw)
		for _, summary := range idx.words[qw] {
			if !seen[summary.ObjectId] {
				seen[summary.ObjectId] = true
				results = append(results, summary)
			}
		}
	}

	return results
}

func countUniqueTypes(index *typeIndex) int {
	seen := make(map[string]bool)
	for _, summaries := range index.words {
		for _, s := range summaries {
			seen[s.ObjectId] = true
		}
	}
	return len(seen)
}
