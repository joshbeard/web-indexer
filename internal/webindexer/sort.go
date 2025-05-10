package webindexer

import (
	"sort"
	"strconv"
	"unicode"
)

func (i *Indexer) sort(items *[]TemplateItem) {
	switch i.Cfg.SortByValue() {
	case SortByDate:
		orderByLastModified(items)
	case SortByName:
		orderByName(items)
	case SortByNaturalName:
		orderByNaturalName(items)
	}

	if i.Cfg.OrderByValue() == OrderDesc {
		sort.SliceStable(*items, func(i, j int) bool {
			return !cmpNatural((*items)[i].Name, (*items)[j].Name)
		})
	}

	if i.Cfg.DirsFirst {
		orderDirsFirst(items)
	}
}

func orderByName(items *[]TemplateItem) {
	sort.SliceStable(*items, func(i, j int) bool {
		return (*items)[i].Name < (*items)[j].Name
	})
}

func orderByLastModified(items *[]TemplateItem) {
	sort.SliceStable(*items, func(i, j int) bool {
		return (*items)[i].LastModified > (*items)[j].LastModified
	})
}

// orderByNaturalName sorts items by their names with numbers ordered
// naturally. e.g. 1,2,10 instead of 1,10,2 or 0.8.2 before 0.8.10
func orderByNaturalName(items *[]TemplateItem) {
	sort.SliceStable(*items, func(i, j int) bool {
		return cmpNatural((*items)[i].Name, (*items)[j].Name)
	})
}

// parseSegments splits a string into numeric and non-numeric segments.
func parseSegments(s string) []string {
	var segments []string
	var currentSegment string

	for _, char := range s {
		if len(currentSegment) == 0 || unicode.IsDigit(rune(currentSegment[0])) == unicode.IsDigit(char) {
			currentSegment += string(char)
		} else {
			segments = append(segments, currentSegment)
			currentSegment = string(char)
		}
	}
	if currentSegment != "" {
		segments = append(segments, currentSegment)
	}
	return segments
}

// cmpNatural compares two strings naturally.
func cmpNatural(a, b string) bool {
	aSegments := parseSegments(a)
	bSegments := parseSegments(b)

	for i := 0; i < len(aSegments) && i < len(bSegments); i++ {
		if aSegments[i] == bSegments[i] {
			continue
		}

		aIsDigit := unicode.IsDigit(rune(aSegments[i][0]))
		bIsDigit := unicode.IsDigit(rune(bSegments[i][0]))

		if aIsDigit && bIsDigit {
			an, _ := strconv.Atoi(aSegments[i])
			bn, _ := strconv.Atoi(bSegments[i])
			return an < bn
		}

		if aIsDigit != bIsDigit {
			return aIsDigit
		}

		return aSegments[i] < bSegments[i]
	}

	return len(aSegments) < len(bSegments)
}

func orderDirsFirst(items *[]TemplateItem) {
	sort.SliceStable(*items, func(i, j int) bool {
		if (*items)[i].IsDir && !(*items)[j].IsDir {
			return true
		}
		if !(*items)[i].IsDir && (*items)[j].IsDir {
			return false
		}

		return false
	})
}
