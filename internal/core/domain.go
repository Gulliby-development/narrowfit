package core

import (
	"sort"
	"strconv"
	"strings"
)

// ParseInt parses a canonical decimal integer within the inclusive bounds.
// The grammar is 0 or an optional minus followed by a nonzero ASCII digit and
// zero or more ASCII digits.
func ParseInt(text string, lo, hi int) (int, bool) {
	if text == "" || lo > hi {
		return 0, false
	}

	start := 0
	negative := false
	if text[0] == '-' {
		negative = true
		start = 1
		if start == len(text) {
			return 0, false
		}
	}
	if text[start] == '0' {
		if start+1 != len(text) || negative {
			return 0, false
		}
	} else if text[start] < '1' || text[start] > '9' {
		return 0, false
	}
	for i := start + 1; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return 0, false
		}
	}

	parsed, err := strconv.ParseInt(text, 10, strconv.IntSize)
	if err != nil {
		return 0, false
	}
	if parsed < int64(lo) || parsed > int64(hi) {
		return 0, false
	}
	return int(parsed), true
}

// ParseDomain parses comma-separated integers and inclusive ranges into a
// nonempty ascending set of values, all within -1000..1000.
func ParseDomain(text string) ([]int, bool) {
	if text == "" {
		return nil, false
	}

	values := make(map[int]struct{})
	for _, item := range strings.Split(text, ",") {
		if item == "" {
			return nil, false
		}

		rangeAt := strings.Index(item, "..")
		if rangeAt < 0 {
			value, ok := ParseInt(item, -1000, 1000)
			if !ok {
				return nil, false
			}
			values[value] = struct{}{}
			continue
		}
		if strings.Index(item[rangeAt+2:], "..") >= 0 {
			return nil, false
		}

		lo, ok := ParseInt(item[:rangeAt], -1000, 1000)
		if !ok {
			return nil, false
		}
		hi, ok := ParseInt(item[rangeAt+2:], -1000, 1000)
		if !ok || lo > hi {
			return nil, false
		}
		for value := lo; value <= hi; value++ {
			values[value] = struct{}{}
		}
	}

	if len(values) == 0 {
		return nil, false
	}
	result := make([]int, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Ints(result)
	return result, true
}

// DomainText renders an ascending unique domain, compressing each consecutive
// run of two or more values as LO..HI.
func DomainText(values []int) string {
	var text strings.Builder
	for i := 0; i < len(values); {
		end := i
		for end+1 < len(values) && values[end] < int(^uint(0)>>1) && values[end+1] == values[end]+1 {
			end++
		}
		if text.Len() > 0 {
			text.WriteByte(',')
		}
		text.WriteString(strconv.Itoa(values[i]))
		if end > i {
			text.WriteString("..")
			text.WriteString(strconv.Itoa(values[end]))
		}
		i = end + 1
	}
	return text.String()
}
