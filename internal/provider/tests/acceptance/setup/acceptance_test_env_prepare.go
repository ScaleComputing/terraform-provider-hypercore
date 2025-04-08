// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

var source_vm_uuid = os.Getenv("SOURCE_VM_UUID")
var existing_vdisk_uuid = os.Getenv("EXISTING_VDISK_UUID")
var source_vm_name = os.Getenv("SOURCE_VM_NAME")

func SetHTTPHeader(req *http.Request) *http.Request {
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

func SetHTTPClient() *http.Client {
	// Create a custom HTTP client with insecure transport
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Disable certificate verification
		},
	}
	client := &http.Client{Transport: tr}

	return client
}

func AreEnvVariablesLoaded() bool {
	if source_vm_uuid == "" || existing_vdisk_uuid == "" || source_vm_name == "" {
		return false
	}
	return true
}

func DoesTestVMExist(host string) bool {
	client := SetHTTPClient()
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/v1/VirDomain/%s", host, source_vm_uuid), bytes.NewBuffer(nil))
	if err != nil {
		log.Fatal(err)
	}
	req = SetHTTPHeader(req)

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

	return resp.StatusCode == http.StatusOK
}

func IsTestVMRunning(host string) bool {
	client := SetHTTPClient()
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/v1/VirDomain/%s", host, source_vm_uuid), bytes.NewBuffer(nil))
	if err != nil {
		log.Fatal(err)
	}
	req = SetHTTPHeader(req)

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

	var result []map[string]interface{}
	errr := json.Unmarshal(body, &result)
	if errr != nil {
		log.Fatal(errr)
	}
	return result[0]["state"] != "SHUTOFF"
}

func DoesVirtualDiskExist(host string) bool {
	client := SetHTTPClient()
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/v1/VirtualDisk/%s", host, existing_vdisk_uuid), bytes.NewBuffer(nil))
	if err != nil {
		log.Fatal(err)
	}
	req = SetHTTPHeader(req)

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
	return resp.StatusCode == http.StatusOK
}

func CleanupEnv(host string) {
	client := SetHTTPClient()
	data := []byte(fmt.Sprintf(`[{"virDomainUUID": "%s", "actionType": "STOP", "cause": "INTERNAL"}]`, source_vm_uuid))
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/rest/v1/VirDomain/action", host), bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}
	req = SetHTTPHeader(req)

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
}

func main() {
	/*
		We are running env setup here based on the arguments passed into GO program it's either going to:
			1. Prepare environment
			2. Cleanup environment
		Argument we are looking to pass is "cleanup" see test.yml workflow file for more information
	*/
	host := os.Getenv("HC_HOST")
	isCleanup := len(os.Args) > 1 && os.Args[1] == "cleanup"
	fmt.Println("Are we doing Cleanup:", isCleanup)

	if isCleanup {
		CleanupEnv(host)
	} else {
		// We are doing env prepare here, make sure all the necessary entities are setup and present
		if !AreEnvVariablesLoaded() {
			log.Fatal("Environment variables aren't loaded, check env file in /acceptance/setup directory")
		}

		if !DoesTestVMExist(host) {
			log.Fatal("Acceptance test VM is missing in your testing environment")
		}

		if IsTestVMRunning(host) {
			log.Fatal("Acceptance test VM is RUNNING and should be turned off before the testing begins")
		}

		if !DoesVirtualDiskExist(host) {
			log.Fatal("Acceptance test Virtual disk is missing in your testing environment")
		}
	}
}
