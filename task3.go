package main

import (
	"fmt"
	"os"
)

func Foo() error {
	var err *os.PathError = nil
	return err
}

func main() {
	err := Foo()
	fmt.Println(err)
	fmt.Println(err == nil)
}

/*
Программа выведет <nil> и false.
В функции возвращается тип *os.PathError со значением nil.
Интерфейс в Go хранит пару (type, value), поэтому он не равен nil, так как type ≠ nil.
Интерфейс равен nil только если и type, и value равны nil.
*/
