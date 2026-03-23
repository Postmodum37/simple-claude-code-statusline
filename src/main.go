package main

import (
	"fmt"
	"os"
)

func main() {
	_, err := ParseStdin(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}
