package main

import "fmt"

func test() (x int) {
	defer func() {
		x++
	}()
	x = 1
	return
}

func anotherTest() int {
	var x int
	defer func() {
		x++
	}()
	x = 1
	return x
}

func main() {
	fmt.Println(test())
	fmt.Println(anotherTest())
}

/*Выведет
2
1
В первом случае результат — это именованная переменная x,
и defer выполняется до окончательного выхода, поэтому успевает изменить x.

Во втором случае результат неименованный, и выражение return x вычисляется до выполнения defer.
defer меняет только локальную переменную, но не возвращаемое значение.
*/
