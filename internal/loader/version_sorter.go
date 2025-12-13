package loader

import (
	"sort"
	"strconv"
	"strings"
)

type VersionSorter []string

func (s VersionSorter) Len() int {
	return len(s)
}

func (s VersionSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s VersionSorter) Less(i, j int) bool {
	parts1 := strings.Split(s[i], ".")
	parts2 := strings.Split(s[j], ".")

	len1 := len(parts1)
	len2 := len(parts2)

	maxLen := len1
	if len2 > maxLen {
		maxLen = len2
	}

	for k := 0; k < maxLen; k++ {
		var p1, p2 string
		if k < len1 {
			p1 = parts1[k]
		}
		if k < len2 {
			p2 = parts2[k]
		}

		n1, err1 := strconv.Atoi(p1)
		n2, err2 := strconv.Atoi(p2)

		if err1 == nil && err2 == nil {
			if n1 != n2 {
				return n1 < n2
			}
		} else {
			if p1 != p2 {
				return p1 < p2
			}
		}
	}

	return len1 < len2
}

func SortVersions(versions []string) {
	sort.Sort(sort.Reverse(VersionSorter(versions)))
}
