package main

import (
	"fmt"
	"sort"
	"strings"
)

func main() {
	example := []string{"пятак", "пятка", "тяпка", "листок", "слиток", "столик", "стол"}
	fmt.Println(searchDict(example))
}

func searchDict(words []string) map[string][]string {
	result := make(map[string][]string)
	anagramGroups := make(map[string][]string)
	seen := make(map[string]bool)

	for _, word := range words {
		word = strings.ToLower(word)
		if seen[word] {
			continue
		}
		seen[word] = true

		runes := []rune(word)
		sort.Slice(runes, func(i, j int) bool {
			return runes[i] < runes[j]
		})
		key := string(runes)
		anagramGroups[key] = append(anagramGroups[key], word)
	}

	for _, group := range anagramGroups {
		if len(group) > 1 {
			sort.Strings(group)
			result[group[0]] = group
		}
	}

	return result
}
