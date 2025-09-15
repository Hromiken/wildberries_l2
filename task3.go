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
nil - При печати fmt.Println(err) значение nil не отображается, поэтому строка пустая.
false При сравнении err == nil — интерфейс не равен nil, потому что его type не пустой (*os.PathError), хоть data и nil → получаем false.
*/
