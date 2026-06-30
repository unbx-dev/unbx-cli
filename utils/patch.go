package utils

import (
	"strconv"
	"strings"
)

// StripPatch removes diff markers from a unified diff patch.
// Returns the clean source (added + context lines only, prefixes stripped)
// and a slice mapping each 0-indexed clean-source row to its 1-indexed file line number.
func StripPatch(patch string) (string, []int) {
	var lines []string
	var mapping []int
	fileLine := 0

	for _, raw := range strings.Split(patch, "\n") {
		if strings.HasPrefix(raw, "@@ ") {
			fileLine = parseHunkStart(raw)
			continue
		}
		if fileLine == 0 {
			continue
		}
		if strings.HasPrefix(raw, "-") {
			continue // removed line: not in new file
		}
		if strings.HasPrefix(raw, "+") || strings.HasPrefix(raw, " ") {
			lines = append(lines, raw[1:])
			mapping = append(mapping, fileLine)
			fileLine++
		}
	}

	return strings.Join(lines, "\n"), mapping
}

// ParseDiffValidLines returns the set of new-file line numbers in the diff.
func ParseDiffValidLines(patch string) map[int]bool {
	_, mapping := StripPatch(patch)
	valid := make(map[int]bool, len(mapping))
	for _, n := range mapping {
		valid[n] = true
	}
	return valid
}

// PatchRowToFileLine converts a 1-indexed scan row to the actual file line number.
// Returns 0 if out of range.
func PatchRowToFileLine(row int, mapping []int) int {
	i := row - 1
	if i < 0 || i >= len(mapping) {
		return 0
	}
	return mapping[i]
}

func parseHunkStart(line string) int {
	for _, field := range strings.Fields(line) {
		if !strings.HasPrefix(field, "+") {
			continue
		}
		s := strings.TrimPrefix(field, "+")
		if idx := strings.Index(s, ","); idx >= 0 {
			s = s[:idx]
		}
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return 0
}
