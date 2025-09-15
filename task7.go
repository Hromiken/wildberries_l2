package main

import (
	"fmt"
	"math/rand"
	"time"
)

func asChan(vs ...int) <-chan int {
	c := make(chan int)
	go func() {
		for _, v := range vs {
			c <- v
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		}
		close(c)
	}()
	return c
}

func merge(a, b <-chan int) <-chan int {
	c := make(chan int)
	go func() {
		for {
			select {
			case v, ok := <-a:
				if ok {
					c <- v
				} else {
					a = nil
				}
			case v, ok := <-b:
				if ok {
					c <- v
				} else {
					b = nil
				}
			}

			if a == nil && b == nil {
				close(c)
				return
			}
		}
	}()
	return c
}

func main() {
	rand.Seed(time.Now().UnixNano())

	a := asChan(1, 3, 5, 7)
	b := asChan(2, 4, 6, 8)

	c := merge(a, b)

	for v := range c {
		fmt.Print(v)
	}
}

/*
Программа выводит все числа 1…8 в случайном порядке,

Работа конвейера с использованием select
Конвейер — это цепочка горутин, где каждая горутина выполняет свою работу и передаёт данные дальше по каналам.

Использование select:
Позволяет горутине одновременно слушать несколько каналов.
Когда значение доступно в любом из каналов, select выбирает его и обрабатывает.
Если канал закрыт, мы можем присвоить ему nil, чтобы больше его не читать.

Fan-in (слияние каналов) как часть конвейера:
Несколько источников данных (а,b) в один канал (c) .
select обеспечивает асинхронное чтение и объединение значений в один поток.

*/
