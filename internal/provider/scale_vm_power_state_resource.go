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
	"github.com/hashicorp/terraform-provider-scale/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ScaleVMPowerStateResource{}
var _ resource.ResourceWithImportState = &ScaleVMPowerStateResource{}

func NewScaleVMPowerStateResource() resource.Resource {
	return &ScaleVMPowerStateResource{}
}

// ScaleVMPowerStateResource defines the resource implementation.
type ScaleVMPowerStateResource struct {
	client *utils.RestClient
}

// ScaleVMPowerStateResourceModel describes the resource data model.
type ScaleVMPowerStateResourceModel struct {
	Id     types.String `tfsdk:"id"`
	VmUUID types.String `tfsdk:"vm_uuid"`
	State  types.String `tfsdk:"state"`
}

func (r *ScaleVMPowerStateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_power_state"
}

func (r *ScaleVMPowerStateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Scale disk resource to manage VM disks",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Power state identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vm_uuid": schema.StringAttribute{
				MarkdownDescription: "VM UUID of which we want to set the power state.",
				Required:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "Desired power state of the VM. Can be: `SHUTOFF`, `RUNNING`, `PAUSED`",
				Required:            true,
			},
		},
	}
}

func (r *ScaleVMPowerStateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Info(ctx, "TTRT ScaleVMPowerStateResource CONFIGURE")
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

func (r *ScaleVMPowerStateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "TTRT ScaleVMPowerStateResource CREATE")
	var data ScaleVMPowerStateResourceModel

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

	tflog.Info(ctx, fmt.Sprintf("TTRT Create: vm_uuid=%s, state=%s", data.VmUUID.ValueString(), data.State.ValueString()))

	diagPowerState := utils.ValidatePowerState(data.State.ValueString())
	if diagPowerState != nil {
		resp.Diagnostics.AddError(diagPowerState.Summary(), diagPowerState.Detail())
		return
	}

	// Power state is not the same as action.
	// Power state is the end state of the VM that was the result of the performed action,
	// so to get what action we need to perform to get to the desired end state of the VM,
	// we need to check with the NEEDED_ACTION_FOR_POWER_STATE.
	actionType := utils.NEEDED_ACTION_FOR_POWER_STATE[data.State.ValueString()]
	createPayload := []map[string]any{
		{
			"virDomainUUID": data.VmUUID.ValueString(),
			"actionType":    actionType,
			"cause":         "INTERNAL",
		},
	}
	diag := utils.ModifyVMPowerState(*r.client, data.VmUUID.ValueString(), createPayload, ctx)
	if diag != nil {
		resp.Diagnostics.AddWarning(diag.Summary(), diag.Detail())
	}

	tflog.Info(ctx, fmt.Sprintf("TTRT Created: vm_uuid=%s, state=%s, action_performed=%s", data.VmUUID.ValueString(), data.State.ValueString(), actionType))

	// TODO: Check if HC3 matches TF
	// save into the Terraform state.
	data.Id = types.StringValue(data.VmUUID.ValueString())

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "Changed the power state")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVMPowerStateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "TTRT ScaleVMPowerStateResource READ")
	var data ScaleVMPowerStateResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Disk read ======================================================================
	restClient := *r.client
	vmUUID := data.VmUUID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("TTRT ScaleVMPowerStateResource Read oldState vmUUID=%s\n", vmUUID))

	pHc3VM, err := utils.GetOneVMWithError(vmUUID, restClient)
	if err != nil {
		resp.Diagnostics.AddError("VM not found", fmt.Sprintf("VM not found - vmUUID=%s", vmUUID))
		return
	}
	hc3VM := *pHc3VM

	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleVMPowerStateResource: vm_uuid=%s, state=%s\n", vmUUID, data.State.ValueString()))

	// save into the Terraform state.
	data.Id = types.StringValue(vmUUID)
	data.VmUUID = types.StringValue(utils.AnyToString(hc3VM["uuid"]))
	data.State = types.StringValue(utils.AnyToString(hc3VM["desiredDisposition"]))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVMPowerStateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "TTRT ScaleVMPowerStateResource UPDATE")
	var data_state ScaleVMPowerStateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data_state)...)
	var data ScaleVMPowerStateResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	restClient := *r.client
	// resourceId := data.Id.ValueString()  // this should be the same as the vmUUID
	vmUUID := data.VmUUID.ValueString()
	vmDesiredState := data.State.ValueString()
	tflog.Debug(
		ctx, fmt.Sprintf(
			"TTRT ScaleVMPowerStateResource Update vm_uuid=%s REQUESTED state=%s\n",
			vmUUID, vmDesiredState),
	)
	tflog.Debug(ctx, fmt.Sprintf(
		"TTRT ScaleVMPowerStateResource Update vm_uuid=%s STATE     state=%s\n",
		vmUUID, data_state.State.ValueString()),
	)

	// Validate the state
	diagPowerState := utils.ValidatePowerState(vmDesiredState)

	if diagPowerState != nil {
		resp.Diagnostics.AddError(diagPowerState.Summary(), diagPowerState.Detail())
		return
	}

	// Power state is not the same as action.
	// Power state is the end state of the VM that was the result of the performed action,
	// so to get what action we need to perform to get to the desired end state of the VM,
	// we need to check with the NEEDED_ACTION_FOR_POWER_STATE.
	actionType := utils.NEEDED_ACTION_FOR_POWER_STATE[vmDesiredState]
	updatePayload := []map[string]any{
		{
			"virDomainUUID": vmUUID,
			"actionType":    actionType,
			"cause":         "INTERNAL",
		},
	}
	diag := utils.ModifyVMPowerState(restClient, vmUUID, updatePayload, ctx)
	if diag != nil {
		resp.Diagnostics.AddWarning(diag.Summary(), diag.Detail())
	}

	// TODO: Check if HC3 matches TF
	// Do not trust UpdateVMPowerState made what we asked for. Read new power state from HC3.
	pHc3VM, err := utils.GetOneVMWithError(vmUUID, restClient)
	if err != nil {
		msg := fmt.Sprintf("VM not found - vmUUID=%s.", vmUUID)
		resp.Diagnostics.AddError("VM not found", msg)
		return
	}
	newHc3VM := *pHc3VM

	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleVMPowerStateResource: vm_uuid=%s, state=%s, action_performed=%s", vmUUID, newHc3VM["state"], actionType))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVMPowerStateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "TTRT ScaleVMPowerStateResource DELETE")
	var data ScaleVMPowerStateResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Extra implementation not needed

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }
}

func (r *ScaleVMPowerStateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "TTRT ScaleVMPowerStateResource IMPORT_STATE")

	vmUUID := req.ID
	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleVMPowerStateResource: vmUUID=%s", vmUUID))

	restClient := *r.client
	hc3VM, err := utils.GetOneVMWithError(vmUUID, restClient)

	if err != nil {
		msg := fmt.Sprintf("VM State import, VM not found -  'vm_uuid'='%s'.", req.ID)
		resp.Diagnostics.AddError("VM State import error, VM not found", msg)
		return
	}

	state := utils.AnyToString((*hc3VM)["desiredDisposition"])
	tflog.Info(ctx, fmt.Sprintf("TTRT state=%v\n", state))

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), vmUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vm_uuid"), vmUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("state"), state)...)
}
