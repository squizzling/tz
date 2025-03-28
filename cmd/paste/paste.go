package main

import (
	"fmt"
	"os"

	"github.com/squizzling/tz/internal/clipboard"
)

func main() {
	err := clipboard.Set(os.Args[1])
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}
