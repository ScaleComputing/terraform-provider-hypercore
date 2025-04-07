package main

import (
	"fmt"
	"os"
)

func main() {
	value := os.Getenv("HC_HOST")
	if value == "" {
		fmt.Println("MY_ENV_VAR is not set")
	} else {
		fmt.Printf("MY_ENV_VAR: %s\n", value)
	}
}
