package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: readme-merge <update|check|hook>")
		os.Exit(1)
	}
	fmt.Println("readme-merge:", os.Args[1])
}
