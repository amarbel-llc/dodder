package mcp_dodder

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"code.linenisgreat.com/dodder/go/lib/charlie/expansion"
)

type tagSummary struct {
	ObjectId    string   `json:"object-id"`
	Date        string   `json:"date"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	ResourceURI string   `json:"resource-uri"`
}

type tagIndex struct {
	bridge Bridge
	once   sync.Once
	words  map[string][]tagSummary
	err    error
}

func makeTagIndex(bridge Bridge) *tagIndex {
	return &tagIndex{bridge: bridge}
}

func (idx *tagIndex) ensureBuilt() error {
	idx.once.Do(func() { idx.err = idx.build() })
	return idx.err
}

func (idx *tagIndex) build() error {
	result, err := idx.bridge.RunCommand(
		context.Background(),
		"show",
		[]string{"-format", "json", ":e"},
		500_000,
	)
	if err != nil {
		return err
	}

	idx.words = make(map[string][]tagSummary)

	for _, line := range strings.Split(result.Stdout, "\n") {
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

		// Skip repo signature artifacts
		if strings.HasPrefix(obj.ObjectId, "%-repo") {
			continue
		}

		summary := tagSummary{
			ObjectId:    obj.ObjectId,
			Date:        obj.Date,
			Description: obj.Description,
			Tags:        obj.Tags,
			ResourceURI: "dodder://tags/" + strings.TrimPrefix(obj.ObjectId, "%"),
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

		addWords(strings.TrimPrefix(obj.ObjectId, "%"))

		for _, word := range strings.Fields(obj.Description) {
			addWords(strings.ToLower(word))
		}

		for _, tag := range obj.Tags {
			if strings.HasPrefix(tag, "-repo") {
				continue
			}
			addWords(strings.TrimPrefix(tag, "%"))
		}
	}

	return nil
}

func (idx *tagIndex) sortedWords() []string {
	words := make([]string, 0, len(idx.words))
	for w := range idx.words {
		words = append(words, w)
	}
	sort.Strings(words)
	return words
}

func (idx *tagIndex) query(queryWords []string) []tagSummary {
	seen := make(map[string]bool)
	var results []tagSummary

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

func countUniqueTags(index *tagIndex) int {
	seen := make(map[string]bool)
	for _, summaries := range index.words {
		for _, s := range summaries {
			seen[s.ObjectId] = true
		}
	}
	return len(seen)
}
