// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type RestClient struct {
	HttpClient *http.Client
	Host       string
	Username   string
	Password   string
	AuthMethod string // local or oidc
	AuthHeader map[string]string
	Timeout    float64
}

func NewRestClient(
	host string,
	username string,
	password string,
	authMethod string,
	timeout float64,
) (*RestClient, error) {
	restClient := &RestClient{
		Host:       host,
		Username:   username,
		Password:   password,
		AuthMethod: authMethod,
		Timeout:    timeout,
	}

	restClient.HttpClient = &http.Client{
		Timeout: time.Duration(restClient.Timeout * float64(time.Second)),
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return restClient, nil
}

func (rc *RestClient) GetClient() *http.Client {
	return rc.HttpClient
}

func (rc *RestClient) GetAuthHeader() map[string]string {
	return rc.AuthHeader
}

func (rc *RestClient) ToJson(response *http.Response) any {
	respBytes, err := io.ReadAll(response.Body)
	if err != nil {
		panic(fmt.Errorf("Failed to read response body: %s", err.Error()))
	}

	var respJson any
	if err := json.Unmarshal(respBytes, &respJson); err != nil {
		response_str := string(respBytes)
		panic(fmt.Errorf("failed to decode response body: %s, body=%v", err.Error(), response_str))
	}
	return respJson
}

func (rc *RestClient) ToJsonObjectList(response *http.Response) []map[string]any {
	respJson := rc.ToJson(response)

	if respJsonObjectList, ok := respJson.([]any); ok {
		var result []map[string]any
		for _, item := range respJsonObjectList {
			if obj, ok := item.(map[string]any); ok {
				result = append(result, obj)
			} else {
				panic(fmt.Errorf("Unexpected item in response list: %v", item))
			}
		}
		return result
	}
	panic(fmt.Errorf("Expected a JSON list of objects, go: %v", respJson))
}

func (rc *RestClient) ToString(response *http.Response) string {
	respBytes, err := io.ReadAll(response.Body)
	if err != nil {
		panic(fmt.Errorf("Failed to read response body: %s", err.Error()))
	}
	return string(respBytes)
}

func (rc *RestClient) Request(method string, endpoint string, body map[string]any, headers map[string]string) *http.Request {
	var jsonBody []byte = nil
	var err error

	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			panic(fmt.Errorf("Failed to marshal JSON body: %s", err.Error()))
		}
	}

	req, err := http.NewRequest(
		method,
		rc.Host+endpoint,
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		panic(fmt.Errorf("Invalid request: %s", err.Error()))
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req
}

func (rc *RestClient) RequestWithList(method string, endpoint string, body []map[string]any, headers map[string]string) *http.Request {
	var jsonBody []byte
	var err error

	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			panic(fmt.Errorf("Failed to marshal JSON body: %s", err.Error()))
		}
	}

	req, err := http.NewRequest(
		method,
		rc.Host+endpoint,
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		panic(fmt.Errorf("Invalid request: %s", err.Error()))
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req
}

func (rc *RestClient) Login() {
	req := rc.Request(
		"POST",
		"/rest/v1/login",
		map[string]any{
			"username": rc.Username,
			"password": rc.Password,
			"useOIDC":  rc.AuthMethod == "oidc",
		},
		nil,
	)

	resp, err := rc.HttpClient.Do(req)
	if err != nil {
		panic(fmt.Errorf("Couldn't authenticate: %s", err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		panic(fmt.Errorf("Authentication failed with status code: %d", resp.StatusCode))
	}

	if respJson, ok := rc.ToJson(resp).(map[string]any); ok {
		rc.AuthHeader = map[string]string{
			"Cookie": fmt.Sprintf("sessionID=%s", respJson["sessionID"]),
		}
	} else {
		panic(fmt.Errorf("Session ID not found in response"))
	}
}

func (rc *RestClient) ListRecords(endpoint string, query map[string]any, timeout float64) []map[string]any {
	useTimeout := timeout
	if timeout == -1 {
		useTimeout = rc.Timeout
	}
	client := rc.HttpClient
	client.Timeout = time.Duration(useTimeout * float64(time.Second))

	req := rc.Request(
		"GET",
		endpoint,
		nil,
		rc.AuthHeader,
	)

	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Errorf("Error making a request: %s", err.Error()))
	}
	defer resp.Body.Close()

	if !(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent) {
		panic(fmt.Errorf("Unexpected response: %d - %v", resp.StatusCode, rc.ToString(resp)))
	}

	records := rc.ToJsonObjectList(resp)
	return filterResults(records, query)
}

func (rc *RestClient) GetRecord(endpoint string, query map[string]any, mustExist bool, timeout float64) *map[string]any {
	useTimeout := timeout
	if timeout == -1 {
		useTimeout = rc.Timeout
	}

	records := rc.ListRecords(endpoint, query, useTimeout)
	if len(records) > 1 {
		panic(fmt.Sprintf("%d records from endpoint %s match the %v query.", len(records), endpoint, query))
	}
	if mustExist && len(records) == 0 {
		panic(fmt.Sprintf("No records from endpoint %s match the %v query.", endpoint, query))
	}

	if len(records) > 0 {
		return &records[0]
	}
	return nil
}

func (rc *RestClient) CreateRecord(endpoint string, payload map[string]any, timeout float64) (*TaskTag, int, error) {
	useTimeout := timeout
	if timeout == -1 {
		useTimeout = rc.Timeout
	}
	client := rc.HttpClient
	client.Timeout = time.Duration(useTimeout * float64(time.Second))

	req := rc.Request(
		"POST",
		endpoint,
		payload,
		rc.AuthHeader,
	)

	resp, err := client.Do(req)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	defer resp.Body.Close()

	respJson := rc.ToJson(resp)
	if resp.StatusCode == 400 {
		respByte, ok := respJson.([]byte)
		if !ok { // this check is needed because of conversion from any to []byte
			panic(fmt.Errorf("Unexpected response body: %v", respJson))
		}
		panic(fmt.Errorf("Error making a request: Maybe the arguments passed to were incorrectly formatted: %v - response: %v", payload, string(respByte)))
	}

	if _, ok := AnyToMap(respJson)["taskTag"]; !ok {
		jsonErrorString, _ := json.Marshal(respJson)
		return nil, resp.StatusCode, fmt.Errorf("%s", string(jsonErrorString))
	}

	return jsonObjectToTaskTag(respJson), resp.StatusCode, err
}

func (rc *RestClient) CreateRecordWithList(endpoint string, payload []map[string]any, timeout float64) (*TaskTag, int, error) {
	useTimeout := timeout
	if timeout == -1 {
		useTimeout = rc.Timeout
	}
	client := rc.HttpClient
	client.Timeout = time.Duration(useTimeout * float64(time.Second))

	req := rc.RequestWithList(
		"POST",
		endpoint,
		payload,
		rc.AuthHeader,
	)

	resp, err := client.Do(req)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	defer resp.Body.Close()

	respJson := rc.ToJson(resp)
	if resp.StatusCode == 400 {
		respByte, ok := respJson.([]byte)
		if !ok { // this check is needed because of conversion from any to []byte
			panic(fmt.Errorf("Unexpected response body: %v", respJson))
		}
		panic(fmt.Errorf("Error making a request: Maybe the arguments passed were incorrectly formatted: %v - response: %v", payload, string(respByte)))
	}

	if _, ok := AnyToMap(respJson)["taskTag"]; !ok {
		jsonErrorString, _ := json.Marshal(respJson)
		return nil, resp.StatusCode, fmt.Errorf("%s", string(jsonErrorString))
	}

	return jsonObjectToTaskTag(respJson), resp.StatusCode, err
}

func (rc *RestClient) UpdateRecord(endpoint string, payload map[string]any, timeout float64, ctx context.Context) (*TaskTag, error) {
	useTimeout := timeout
	if timeout == -1 {
		useTimeout = rc.Timeout
	}
	client := rc.HttpClient
	client.Timeout = time.Duration(useTimeout * float64(time.Second))

	req := rc.Request(
		"PATCH",
		endpoint,
		payload,
		rc.AuthHeader,
	)

	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Errorf("Error making a request: %s", err.Error()))
	}
	defer resp.Body.Close()

	respJson := rc.ToJson(resp)
	if resp.StatusCode == 400 {
		respByte, ok := respJson.([]byte)
		if !ok { // this check is needed because of conversion from any to []byte
			panic(fmt.Errorf("Unexpected response body: %v", respJson))
		}
		panic(fmt.Errorf("Error making a request: Maybe the arguments passed were incorrectly formatted: %v - response: %v", payload, string(respByte)))
	}

	if _, ok := AnyToMap(respJson)["taskTag"]; !ok {
		jsonErrorString, _ := json.Marshal(respJson)
		return nil, fmt.Errorf("%s", string(jsonErrorString))
	}

	return jsonObjectToTaskTag(respJson), nil
}

func (rc *RestClient) DeleteRecord(endpoint string, timeout float64, ctx context.Context) *TaskTag {
	useTimeout := timeout
	if timeout == -1 {
		useTimeout = rc.Timeout
	}
	client := rc.HttpClient
	client.Timeout = time.Duration(useTimeout * float64(time.Second))

	req := rc.Request(
		"DELETE",
		endpoint,
		nil,
		rc.AuthHeader,
	)

	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Errorf("Error making a request: %s", err.Error()))
	}
	defer resp.Body.Close()

	respJson := rc.ToJson(resp)

	return jsonObjectToTaskTag(respJson)
}
