package main

import "fmt"

func main() {
	example1 := "()"
	example2 := "({[]})"
	example3 := "(])"

	fmt.Println(isValid(example1), isValid(example2), isValid(example3))
}
func isValid(s string) bool {
	if len(s) == 0 || len(s)%2 == 1 {
		return false
	}

	stack := []rune{}
	hash := map[rune]rune{')': '(', ']': '[', '}': '{'}

	for _, char := range s {
		if match, found := hash[char]; found {
			if len(stack) > 0 && stack[len(stack)-1] == match {
				stack = stack[:len(stack)-1]
			} else {
				return false
			}
		} else {
			stack = append(stack, char)
		}
	}
	return len(stack) == 0
}
