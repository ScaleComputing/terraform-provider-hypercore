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
	"reflect"
	"time"
)

type EnvConfig struct {
	SourceVmUUID      string
	ExistingVdiskUUID string
	SourceVmName      string
	SourceDiskUUID    string
	SourceNicUUID     string
}

const (
	VirDomainEndpoint       = "/rest/v1/VirDomain/"
	VirtualDiskEndpoint     = "/rest/v1/VirtualDisk/"
	VirDomainActionEndpoint = "/rest/v1/VirDomain/action"
)

func LoadEnv() EnvConfig {
	return EnvConfig{
		SourceVmUUID:      os.Getenv("SOURCE_VM_UUID"),
		ExistingVdiskUUID: os.Getenv("EXISTING_VDISK_UUID"),
		SourceVmName:      os.Getenv("SOURCE_VM_NAME"),
		SourceDiskUUID:    os.Getenv("SOURCE_DISK_UUID"),
		SourceNicUUID:     os.Getenv("SOURCE_NIC_UUID"),
	}
}

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
func SendHTTPRequest(request *http.Request, client *http.Client) (*http.Response, []byte) {
	resp, err := client.Do(request)
	if err != nil {
		log.Fatalf("Sending request failed with %v", err)
	}
	defer resp.Body.Close()

	// Read and print the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Reading request response body failed with %v", err)
	}

	fmt.Println("Response Status:", resp.Status)
	fmt.Println("Response Body:", string(body))

	return resp, body
}
func SetHTTPRequest(method string, url string, data []byte) *http.Request {
	var req *http.Request
	var err error

	// Set request method and body
	if method == "GET" {
		req, err = http.NewRequest("GET", url, bytes.NewBuffer(nil))
	} else {
		req, err = http.NewRequest("POST", url, bytes.NewBuffer(data))
	}

	// Handle any errors that occur
	if err != nil {
		log.Fatalf("%s request to url: %s failed with error: %v", method, url, err)
	}

	return req
}

func AreEnvVariablesLoaded(env EnvConfig) bool {
	if env.SourceVmUUID == "" || env.ExistingVdiskUUID == "" || env.SourceVmName == "" || env.SourceDiskUUID == "" || env.SourceNicUUID == "" {
		return false
	}
	return true
}
func DoesTestVMExist(host string, client *http.Client, env EnvConfig) bool {
	url := fmt.Sprintf("%s%s%s", host, VirDomainEndpoint, env.SourceVmUUID)

	req := SetHTTPRequest("GET", url, nil)
	req = SetHTTPHeader(req)

	resp, _ := SendHTTPRequest(req, client)

	return resp.StatusCode == http.StatusOK
}
func IsTestVMRunning(host string, client *http.Client, env EnvConfig) bool {
	url := fmt.Sprintf("%s%s%s", host, VirDomainEndpoint, env.SourceVmUUID)

	req := SetHTTPRequest("GET", url, nil)
	req = SetHTTPHeader(req)

	_, body := SendHTTPRequest(req, client)

	var result []map[string]interface{}
	err := json.Unmarshal(body, &result)
	if err != nil {
		log.Fatal(err)
	}
	return result[0]["state"] != "SHUTOFF"
}
func DoesVirtualDiskExist(host string, client *http.Client, env EnvConfig) bool {
	url := fmt.Sprintf("%s%s%s", host, VirtualDiskEndpoint, env.ExistingVdiskUUID)

	req := SetHTTPRequest("GET", url, nil)
	req = SetHTTPHeader(req)

	resp, _ := SendHTTPRequest(req, client)

	return resp.StatusCode == http.StatusOK
}
func IsBootOrderCorrect(host string, client *http.Client, env EnvConfig) bool {
	expectedBootOrder := []string{env.SourceDiskUUID, env.SourceNicUUID}
	url := fmt.Sprintf("%s%s%s", host, VirDomainEndpoint, env.SourceVmUUID)

	req := SetHTTPRequest("GET", url, nil)
	req = SetHTTPHeader(req)

	_, body := SendHTTPRequest(req, client)

	var result []map[string]interface{}
	err := json.Unmarshal(body, &result)
	if err != nil {
		log.Fatal(err)
	}
	return reflect.DeepEqual(result[0]["bootDevices"], expectedBootOrder)
}
func PrepareEnv(host string, client *http.Client, env EnvConfig) {
	// We are doing env prepare here, make sure all the necessary entities are setup and present
	if !AreEnvVariablesLoaded(env) {
		log.Fatal("Environment variables aren't loaded, check env file in /acceptance/setup directory")
	} else {
		fmt.Println("Environment variables are loaded correctly")
	}
	if !DoesTestVMExist(host, client, env) {
		log.Fatal("Acceptance test VM is missing in your testing environment")
	} else {
		fmt.Println("Acceptance test VM is present in the testing environment")
	}
	if IsTestVMRunning(host, client, env) {
		log.Fatal("Acceptance test VM is RUNNING and should be turned off before the testing begins")
	} else {
		fmt.Println("Acceptance test VM is in the correct SHUTOFF state")
	}
	if !DoesVirtualDiskExist(host, client, env) {
		log.Fatal("Acceptance test Virtual disk is missing in your testing environment")
	} else {
		fmt.Println("Acceptance test Virtual disk is present in your testing environment")
	}
	if IsBootOrderCorrect(host, client, env) {
		log.Fatal("Acceptance test Boot order is incorrect on the test VM, should be disk followed by network interface")
	} else {
		fmt.Println("Acceptance test Boot order is in correct order")
	}
}

func CleanUpPowerState(host string, client *http.Client, env EnvConfig) {
	data := []byte(fmt.Sprintf(`[{"virDomainUUID": "%s", "actionType": "STOP", "cause": "INTERNAL"}]`, env.SourceVmUUID))
	url := fmt.Sprintf("%s%s", host, VirDomainActionEndpoint)
	req := SetHTTPRequest("POST", url, data)
	req = SetHTTPHeader(req)
	SendHTTPRequest(req, client)
	// wait 30 seconds for VM to shutdown and then proceed with other cleanup tasks
	time.Sleep(30 * time.Second)
}
func CleanUpBootOrder(host string, client *http.Client, env EnvConfig) {
	bootOrder := []string{env.SourceDiskUUID, env.SourceNicUUID}
	payload := map[string]interface{}{
		"bootDevices": bootOrder,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}
	url := fmt.Sprintf("%s%s%s", host, VirDomainEndpoint, env.SourceVmUUID)
	req := SetHTTPRequest("POST", url, data)
	req = SetHTTPHeader(req)
	SendHTTPRequest(req, client)
}
func CleanupEnv(host string, client *http.Client, env EnvConfig) {
	CleanUpPowerState(host, client, env)
	CleanUpBootOrder(host, client, env)
}

func main() {
	/*
		We are running env setup here based on the arguments passed into GO program it's either going to:
			1. Prepare environment
			2. Cleanup environment
		Argument we are looking to pass is "cleanup" see test.yml workflow file for more information
	*/
	env := LoadEnv()
	host := os.Getenv("HC_HOST")
	client := SetHTTPClient()
	isCleanup := len(os.Args) > 1 && os.Args[1] == "cleanup"
	fmt.Println("Are we doing Cleanup:", isCleanup)

	if isCleanup {
		CleanupEnv(host, client, env)
	} else {
		PrepareEnv(host, client, env)
	}
}
