package main

import (
	"fmt"
	"os"

	"github.com/beevik/ntp"
)

func main() {
	t, err := ntp.Time("pool.ntp.org")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Ошибка получения времени: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(t)
}
