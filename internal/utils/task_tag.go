// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type TaskTag struct {
	CreatedUUID string
	TaskTag     string
}

func NewTaskTag(createdUUID string, taskTag string) (*TaskTag, error) {
	taskTagData := &TaskTag{
		CreatedUUID: createdUUID,
		TaskTag:     taskTag,
	}

	return taskTagData, nil
}

func (tt *TaskTag) ToMap() map[string]any {
	return map[string]any{
		"createdUUID": tt.CreatedUUID,
		"taskTag":     tt.TaskTag,
	}
}

func (tt *TaskTag) WaitTask(restClient RestClient, ctx context.Context) {
	if tt == nil || tt.TaskTag == "" {
		return
	}

	for { // while true
		select {
		case <-ctx.Done(): // take into account SIGINT (ctrl+c)
			tflog.Error(ctx, "Operation was interrupted by Terraform. Whatever request was made to host prior to cancelation, will now finish.")
			return
		default:
			taskStatus := restClient.GetRecord(
				fmt.Sprintf("/rest/v1/TaskTag/%s", tt.TaskTag),
				map[string]any{},
				false,
				-1,
			)

			if taskStatus == nil { // No such taskStatus found
				return
			}

			if state, ok := (*taskStatus)["state"]; ok {
				if state == "ERROR" || state == "UNINITIALIZED" { // Task has finished unsuccessfully or was never initialized. Both are errors.
					panic(fmt.Sprintf("Error executing task: %s, %s", state, taskStatus))
				}

				if !(state == "RUNNING" || state == "QUEUED") { // TaskTag has finished
					return
				}
			}
			time.Sleep(1 * time.Second) // sleep 1 second
		}
	}
}

func (tt *TaskTag) GetStatus(restClient RestClient) *map[string]any {
	if tt == nil || tt.TaskTag == "" {
		return nil
	}

	taskStatus := restClient.GetRecord(
		fmt.Sprintf("/rest/v1/TaskTag/%s", tt.TaskTag),
		map[string]any{},
		false,
		-1,
	)

	return taskStatus
}
