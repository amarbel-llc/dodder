package ui

import (
	"fmt"
	"strings"
)

type HumanReadableBytes string

func (h HumanReadableBytes) String() string {
	return string(h)
}

func (h *HumanReadableBytes) Set(v string) error {
	if _, err := parseHumanBytes(v); err != nil {
		return err
	}

	*h = HumanReadableBytes(v)

	return nil
}

func (h HumanReadableBytes) GetByteCount() uint64 {
	n, _ := parseHumanBytes(string(h))
	return n
}

func parseHumanBytes(s string) (uint64, error) {
	s = strings.TrimSpace(s)

	if s == "" || s == "0" {
		return 0, nil
	}

	i := 0
	hasDot := false

	for i < len(s) {
		if s[i] >= '0' && s[i] <= '9' {
			i++
		} else if s[i] == '.' && !hasDot {
			hasDot = true
			i++
		} else {
			break
		}
	}

	if i == 0 {
		return 0, fmt.Errorf("no numeric value in %q", s)
	}

	numStr := s[:i]
	suffix := strings.TrimSpace(s[i:])

	var num float64

	if _, err := fmt.Sscanf(numStr, "%f", &num); err != nil {
		return 0, fmt.Errorf("invalid number %q: %w", numStr, err)
	}

	if num < 0 {
		return 0, fmt.Errorf("negative size %q", s)
	}

	var multiplier uint64

	switch strings.ToUpper(suffix) {
	case "", "B":
		multiplier = 1
	case "K", "KB", "KIB":
		multiplier = 1024
	case "M", "MB", "MIB":
		multiplier = 1024 * 1024
	case "G", "GB", "GIB":
		multiplier = 1024 * 1024 * 1024
	case "T", "TB", "TIB":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "P", "PB", "PIB":
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown unit %q in %q", suffix, s)
	}

	return uint64(num * float64(multiplier)), nil
}

func GetHumanBytesStringOrError(bytes int64) string {
	if bytes < 0 {
		return fmt.Sprintf("%d bytes (error)", bytes)
	} else {
		return GetHumanBytesString(uint64(bytes))
	}
}

func GetHumanBytesString(bytes uint64) string {
	const unit = uint64(1024)

	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := uint64(unit), 0

	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "kMGTPE"[exp])
}
