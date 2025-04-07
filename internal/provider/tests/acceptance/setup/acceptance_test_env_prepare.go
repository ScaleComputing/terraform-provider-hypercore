package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	host := os.Getenv("HC_HOST")
	// user := os.Getenv("HC_USERNAME")
	// pass := os.Getenv("HC_PASSWORD")

	resp, err := http.Get(fmt.Sprintf("%s/rest/v1/ISO", host))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Read and print the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response Status:", resp.Status)
	fmt.Println("Response Body:", string(body))
}
