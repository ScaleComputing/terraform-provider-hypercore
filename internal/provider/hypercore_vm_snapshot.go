// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-hypercore/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &HypercoreVMSnapshotResource{}
var _ resource.ResourceWithImportState = &HypercoreVMSnapshotResource{}

func NewHypercoreVMSnapshotResource() resource.Resource {
	return &HypercoreVMSnapshotResource{}
}

// HypercoreVMSnapshotResource defines the resource implementation.
type HypercoreVMSnapshotResource struct {
	client *utils.RestClient
}

// HypercoreVMSnapshotResourceModel describes the resource data model.
type HypercoreVMSnapshotResourceModel struct {
	Id     types.String `tfsdk:"id"`
	VmUUID types.String `tfsdk:"vm_uuid"`
	Type   types.String `tfsdk:"type"`
	Label  types.String `tfsdk:"label"`
}

func (r *HypercoreVMSnapshotResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_snapshot"
}

func (r *HypercoreVMSnapshotResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Hypercore VM snapshot resource to manage VM snapshots",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "VM snapshot identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vm_uuid": schema.StringAttribute{
				MarkdownDescription: "VM UUID of which we want to create a snapshot.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Snapshot type. Can be: USER, AUTOMATED, SUPPORT",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"label": schema.StringAttribute{
				MarkdownDescription: "Snapshot label.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *HypercoreVMSnapshotResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Info(ctx, "TTRT HypercoreVMSnapshotResource CONFIGURE")
	// Prevent padisk if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	restClient, ok := req.ProviderData.(*utils.RestClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = restClient
}

func (r *HypercoreVMSnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "TTRT HypercoreVMSnapshotResource CREATE")
	var data HypercoreVMSnapshotResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured HTTP Client",
			"Expected configured HTTP client. Please report this issue to the provider developers.",
		)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	restClient := *r.client
	vmUUID := data.VmUUID.ValueString()
	snapLabel := data.Label.ValueString()
	snapType := "USER"

	if snapLabel == "" || data.Label.IsUnknown() || data.Label.IsNull() {
		resp.Diagnostics.AddError(
			"Missing 'label' parameter",
			"Snapshots must be labeled",
		)
		return
	}

	// Create VM snapshot
	payload := map[string]any{
		"domainUUID": vmUUID,
		"label":      snapLabel,
		"type":       snapType,

		// These are all defaults from API and are
		// required by the API to be present
		"automatedTriggerTimestamp":      0,
		"localRetainUntilTimestamp":      0,
		"remoteRetainUntilTimestamp":     0,
		"blockCountDiffFromSerialNumber": -1,
		"replication":                    true,
	}
	snapUUID, snap, _diag := utils.CreateVMSnapshot(restClient, vmUUID, payload, ctx)
	if _diag != nil {
		resp.Diagnostics.AddWarning(_diag.Summary(), _diag.Detail())
	}
	tflog.Info(ctx, fmt.Sprintf("TTRT Created: vm_uuid=%s, label=%s, type=%s, snap=%v", vmUUID, snapLabel, snapType, snap))

	// TODO: Check if HC3 matches TF
	// save into the Terraform state.
	data.Id = types.StringValue(snapUUID)
	data.Type = types.StringValue(snapType)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "Created a VM snapshot")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreVMSnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "TTRT HypercoreVMSnapshotResource READ")
	var data HypercoreVMSnapshotResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Snapshot read ======================================================================
	restClient := *r.client
	snapUUID := data.Id.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("TTRT HypercoreSnapshot Read oldState snapUUID=%s\n", snapUUID))

	pHc3Snap := utils.GetVMSnapshotByUUID(restClient, snapUUID)
	if pHc3Snap == nil {
		resp.Diagnostics.AddError("Snapshot not found", fmt.Sprintf("Snapshot not found - snapUUID=%s", snapUUID))
		return
	}
	hc3Snap := *pHc3Snap

	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreSnapshot: snap_uuid=%s, vm_uuid=%s, label=%s, type=%s\n", snapUUID, data.VmUUID.ValueString(), data.Label.ValueString(), data.Type.ValueString()))

	vmUUID := utils.AnyToString(hc3Snap["domainUUID"])
	snapLabel := utils.AnyToString(hc3Snap["label"])
	snapType := utils.AnyToString(hc3Snap["type"])

	// save into the Terraform state.
	data.Id = types.StringValue(snapUUID)
	data.VmUUID = types.StringValue(vmUUID)
	data.Label = types.StringValue(snapLabel)
	data.Type = types.StringValue(snapType)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreVMSnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// NOTE: /rest/v1/VirDomainSnapshot has no update endpoints, so update is not needed here

	// tflog.Info(ctx, "TTRT HypercoreVMSnapshotResource UPDATE")
	// var data_state HypercoreVMSnapshotResourceModel
	// resp.Diagnostics.Append(req.State.Get(ctx, &data_state)...)
	// var data HypercoreVMSnapshotResourceModel
	//
	// // Read Terraform plan data into the model
	// resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	//
	// if resp.Diagnostics.HasError() {
	// 	return
	// }
	//
	// // Save updated data into Terraform state
	// resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreVMSnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "TTRT HypercoreVMSnapshotResource DELETE")
	var data HypercoreVMSnapshotResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }

	restClient := *r.client
	snapUUID := data.Id.ValueString()
	taskTag := restClient.DeleteRecord(
		fmt.Sprintf("/rest/v1/VirDomainSnapshot/%s", snapUUID),
		-1,
		ctx,
	)
	taskTag.WaitTask(restClient, ctx)
}

func (r *HypercoreVMSnapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "TTRT HypercoreVMSnapshotResource IMPORT_STATE")

	snapUUID := req.ID
	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreVMSnapshotResource: snapUUID=%s", snapUUID))

	restClient := *r.client
	hc3Snapshot := utils.GetVMSnapshotByUUID(restClient, snapUUID)

	if hc3Snapshot == nil {
		msg := fmt.Sprintf("VM Snapshot import, snapshot not found -  'snap_uuid'='%s'.", req.ID)
		resp.Diagnostics.AddError("VM Snapshot import error, snapshot not found", msg)
		return
	}

	snapType := utils.AnyToString((*hc3Snapshot)["type"])
	snapLabel := utils.AnyToString((*hc3Snapshot)["label"])
	vmUUID := utils.AnyToString((*hc3Snapshot)["domainUUID"])

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), snapUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vm_uuid"), vmUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), snapType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("label"), snapLabel)...)
}
