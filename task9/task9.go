package main

import (
	"errors"
	"fmt"
	"strconv"
	"unicode"
)

func main() {
	example1 := "a4bc2d5e"
	example2 := "abcd"
	example3 := "45"
	example4 := ""
	example5 := "qwe\\4\\5"
	example6 := "qwe\\45"

	fmt.Println(unpackingString(example1))
	fmt.Println(unpackingString(example2))
	fmt.Println(unpackingString(example3))
	fmt.Println(unpackingString(example4))
	fmt.Println(unpackingString(example5))
	fmt.Println(unpackingString(example6))

}

func unpackingString(s string) (string, error) {
	r := []rune(s)
	result := make([]rune, 0, 2*len(r))

	if len(r) == 0 {
		return "", nil
	}
	if unicode.IsDigit(r[0]) {
		return "", errors.New("invalid string")
	}

	escaped := false
	numCount := ""
	for i := 0; i < len(r); i++ {

		if escaped {
			result = append(result, r[i])
			escaped = false
			numCount = ""
			continue
		}

		if r[i] == '\\' {
			escaped = true
			continue
		}

		if unicode.IsLetter(r[i]) {
			if numCount != "" {
				last := result[len(result)-1]
				count, _ := strconv.Atoi(numCount)
				if count == 0 {
					result = result[:len(result)-1]
				} else {
					for j := 1; j < count; j++ {
						result = append(result, last)
					}
				}
				numCount = ""
			}
			result = append(result, r[i])
			continue
		}

		if unicode.IsDigit(r[i]) {
			numCount += string(r[i])
			continue
		}

		return "", errors.New("invalid string: unsupported symbol")
	}

	if numCount != "" {
		if len(result) == 0 {
			return "", errors.New("invalid string: digit with no previous symbol")
		}
		last := result[len(result)-1]
		count, _ := strconv.Atoi(numCount)
		if count == 0 {
			result = result[:len(result)-1]
		} else {
			for j := 1; j < count; j++ {
				result = append(result, last)
			}
		}
	}

	if escaped {
		return "", errors.New("invalid string: ends with backslash")
	}

	return string(result), nil
}
