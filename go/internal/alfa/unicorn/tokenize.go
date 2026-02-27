package unicorn

import (
	"bufio"
	"sort"
	"strings"
	"unicode"
)

func ExtractUniqueComponents(lines []string) []string {
	type tokenInfo struct {
		count int
	}

	tokens := make(map[string]tokenInfo)

	for _, line := range lines {
		scanner := bufio.NewScanner(strings.NewReader(line))
		scanner.Split(bufio.ScanWords)

		var candidates []string

		for scanner.Scan() {
			word := scanner.Text()

			if len(word) <= 3 {
				continue
			}

			if strings.ContainsFunc(word, unicode.IsPunct) {
				continue
			}

			word = StripDiacritics(word)
			candidates = append(candidates, word)
		}

		var chosen string

		switch len(candidates) {
		case 0:
			continue
		case 1:
			chosen = candidates[0]
		default:
			chosen = candidates[len(candidates)-1]
		}

		ti := tokens[chosen]
		ti.count++
		tokens[chosen] = ti
	}

	var result []string

	for k, v := range tokens {
		if v.count == 1 {
			result = append(result, k)
		}
	}

	sort.Strings(result)

	if result == nil {
		result = []string{}
	}

	return result
}
