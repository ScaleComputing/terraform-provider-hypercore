// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// var source_vm_name = "integration-test-vm"
var source_vm_uuid = "97904009-1878-4881-b6df-83c85ab7dc1a"
var existing_vdisk_uuid = "33c78baf-c3c6-4600-8432-9c7a2a3008ab"

func SetHeader(req *http.Request) *http.Request {
	user := os.Getenv("HC_USERNAME")
	pass := os.Getenv("HC_PASSWORD")

	// Create the Basic Authentication string
	auth := user + ":" + pass
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

	// Set the Content-Type header
	req.Header.Set("Content-Type", "application/json")

	// Set the Content-Length header (not required, it's usually set automatically)
	// req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))

	// Set Basic Authentication header
	req.Header.Set("Authorization", authHeader)
	return req
}

func SetClient() *http.Client {
	// Create a custom HTTP client with insecure transport
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Disable certificate verification
		},
	}
	client := &http.Client{Transport: tr}

	return client
}

func DoesTestVMExist(host string) bool {
	client := SetClient()
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/v1/VirDomain/%s", host, source_vm_uuid), bytes.NewBuffer(nil))
	if err != nil {
		log.Fatal(err)
	}
	req = SetHeader(req)

	// Execute the request
	resp, err := client.Do(req)
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
	return true
}

func IsTestVMRunning(host string) bool {
	client := SetClient()
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/v1/VirDomain/%s", host, source_vm_uuid), bytes.NewBuffer(nil))
	if err != nil {
		log.Fatal(err)
	}
	req = SetHeader(req)

	// Execute the request
	resp, err := client.Do(req)
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
	return false
}

func DoesVirtualDiskExist(host string) bool {
	client := SetClient()
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/v1/VirtualDisk/%s", host, existing_vdisk_uuid), bytes.NewBuffer(nil))
	if err != nil {
		log.Fatal(err)
	}
	req = SetHeader(req)

	// Execute the request
	resp, err := client.Do(req)
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
	return true
}

func main() {
	host := os.Getenv("HC_HOST")

	if !DoesTestVMExist(host) || IsTestVMRunning(host) || !DoesVirtualDiskExist(host) {
		log.Fatal("One or more prerequisites are missing in your test environment")
	}
}
