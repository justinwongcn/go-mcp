package main

import (
	"fmt"
	"os"
)

func main() {
	if _, err := os.Stdin.Read(make([]byte, 1)); err != nil {
		fmt.Println(err)
		return
	}
}
