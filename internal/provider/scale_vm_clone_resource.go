// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
var _ resource.Resource = &ScaleVMCloneResource{}
var _ resource.ResourceWithImportState = &ScaleVMCloneResource{}

func NewScaleVMCloneResource() resource.Resource {
	return &ScaleVMCloneResource{}
}

// ScaleVMCloneResource defines the resource implementation.
type ScaleVMCloneResource struct {
	client *utils.RestClient
}

// ScaleVMCloneResourceModel describes the resource data model.
type ScaleVMCloneResourceModel struct {
	Group       types.String `tfsdk:"group"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	VCPU        types.Int32  `tfsdk:"vcpu"`
	Memory      types.Int64  `tfsdk:"memory"`
	PowerState  types.String `tfsdk:"power_state"`
	Clone       CloneModel   `tfsdk:"clone"`
	Id          types.String `tfsdk:"id"`
}

type CloneModel struct {
	SourceVMUUID types.String `tfsdk:"source_vm_uuid"`
	UserData     types.String `tfsdk:"user_data"`
	MetaData     types.String `tfsdk:"meta_data"`
}

func (r *ScaleVMCloneResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_clone"
}

func (r *ScaleVMCloneResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "ScaleVM resource to create a VM from a template VM",

		Attributes: map[string]schema.Attribute{
			"group": schema.StringAttribute{
				MarkdownDescription: "Group/tag to create this VM in",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of this VM",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of this VM",
				Optional:            true,
			},
			"vcpu": schema.Int32Attribute{
				MarkdownDescription: "" +
					"Number of CPUs on this VM. If the cloned VM was already created and it's <br>" +
					"`VCPU` was modified, the cloned VM will be rebooted (either gracefully or forcefully)",
				Optional: true,
			},
			"memory": schema.Int64Attribute{
				MarkdownDescription: "" +
					"Memory (RAM) size in `MiB`: If the cloned VM was already created <br>" +
					"and it's memory was modified, the cloned VM will be rebooted (either gracefully or forcefully)",
				Optional: true,
			},
			"power_state": schema.StringAttribute{
				MarkdownDescription: "" +
					"Initial power state on create: If not provided, it will default to `stop`. <br>" +
					"Available power states are: start, started, stop, shutdown, reboot, reset. <br>" +
					"Power state can be modified on the cloned VM even after the cloning process.",
				Optional: true,
			},
			"clone": schema.ObjectAttribute{
				Optional: true,
				AttributeTypes: map[string]attr.Type{
					"source_vm_uuid": types.StringType,
					"user_data":      types.StringType,
					"meta_data":      types.StringType,
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ScaleVM identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ScaleVMCloneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Info(ctx, "TTRT ScaleVMCloneResource CONFIGURE")
	// Prevent panic if the provider has not been configured.
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

func (r *ScaleVMCloneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "TTRT ScaleVMCloneResource CREATE")
	var data ScaleVMCloneResourceModel
	// var readData ScaleVMCloneResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	// resp.State.Get(ctx, &readData)
	//
	// tflog.Debug(ctx, fmt.Sprintf("STATE IS: %v\n", readData.Disks))

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

	var tags *[]string
	var description *string
	var powerState string

	if data.Group.ValueString() == "" {
		tags = nil
	} else {
		tags = &[]string{data.Group.ValueString()}
	}

	if data.Description.ValueString() == "" {
		description = nil
	} else {
		description = data.Description.ValueStringPointer()
	}

	if data.PowerState.ValueString() == "" {
		powerState = "stop"
	} else {
		powerState = data.PowerState.ValueString()
	}

	tflog.Info(ctx, fmt.Sprintf("TTRT Create: name=%s, source_uuid=%s", data.Name.ValueString(), data.Clone.SourceVMUUID.ValueString()))

	vmClone, _ := utils.NewVMClone(
		data.Name.ValueString(),
		data.Clone.SourceVMUUID.ValueString(),
		data.Clone.UserData.ValueString(),
		data.Clone.MetaData.ValueString(),
		description,
		tags,
		data.VCPU.ValueInt32Pointer(),
		data.Memory.ValueInt64Pointer(),
		&powerState,
	)
	changed, msg := vmClone.Create(*r.client, ctx)
	tflog.Info(ctx, fmt.Sprintf("Changed: %t, Message: %s\n", changed, msg))

	// General parametrization
	// set: description, group, vcpu, memory, power_state
	changed, vmWasRebooted, vmDiff := vmClone.SetVMParams(*r.client, ctx)
	tflog.Info(ctx, fmt.Sprintf("Changed: %t, Was VM Rebooted: %t, Diff: %v", changed, vmWasRebooted, vmDiff))

	// save into the Terraform state.
	data.Id = types.StringValue(vmClone.UUID)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVMCloneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "TTRT ScaleVMCloneResource READ")
	var data ScaleVMCloneResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

	// VM read ======================================================================
	restClient := *r.client
	vm_uuid := data.Id.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("TTRT ScaleVMCloneResource Read oldState vm_uuid=%s\n", vm_uuid))
	hc3_vm := utils.GetOneVM(vm_uuid, restClient)
	tflog.Debug(ctx, fmt.Sprintf("TTRT ScaleVMCloneResource Read vmhc3_vm=%s\n", hc3_vm))
	hc3_vm_name := utils.AnyToString(hc3_vm["name"])
	tflog.Debug(ctx, fmt.Sprintf("TTRT ScaleVMCloneResource Read vm_uuid=%s hc3_vm=(name=%s)\n", vm_uuid, hc3_vm_name))

	data.Name = types.StringValue(utils.AnyToString(hc3_vm["name"]))
	data.Description = types.StringValue(utils.AnyToString(hc3_vm["description"]))
	// data.Group TODO - replace "group" string with "tags" list of strings

	hc3_power_state := utils.AnyToString(hc3_vm["state"])
	// line below look like correct thing to do. But "terraform plan -refresh-only"
	// complains about change 'power_state = "stop" -> "stopped"
	tf_power_state := types.StringValue(utils.FromHypercoreToTerraformPowerState[hc3_power_state])
	// TEMP make "terraform plan -refresh-only" report "nothing changed"
	hc3_stopped_states := []string{"SHUTOFF", "CRASHED"}
	if slices.Contains(hc3_stopped_states, hc3_power_state) {
		tf_power_state = types.StringValue("stop")
	}
	data.PowerState = tf_power_state

	// desiredDisposition TODO
	// uiState TODO
	data.VCPU = types.Int32Value(int32(utils.AnyToInteger64(hc3_vm["numVCPU"])))
	data.Memory = types.Int64Value(utils.AnyToInteger64(hc3_vm["mem"]) / 1024 / 1024)

	// ==============================================================================

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVMCloneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "TTRT ScaleVMCloneResource UPDATE")
	var data_state ScaleVMCloneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data_state)...)
	var data ScaleVMCloneResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	// data.PowerState
	// ======================================================================
	restClient := *r.client
	vm_uuid := data.Id.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("TTRT ScaleVMCloneResource Update vm_uuid=%s REQ   vcpu=%d description=%s", vm_uuid, data.VCPU.ValueInt32(), data.Description.String()))
	tflog.Debug(ctx, fmt.Sprintf("TTRT ScaleVMCloneResource Update vm_uuid=%s STATE vcpu=%d description=%s", vm_uuid, data_state.VCPU.ValueInt32(), data_state.Description.String()))

	updatePayload := map[string]any{}
	if data_state.Name != data.Name {
		updatePayload["name"] = data.Name.String()
	}
	if data_state.Description != data.Description {
		updatePayload["description"] = data.Description.String()
	}
	// if changed, ok := changedParams["tags"]; ok && changed {
	// 	updatePayload["tags"] = tagsListToCommaString(*vc.tags)
	// }
	// updatePayload["tags"] = "ananas,aaa,bbb"
	if data_state.Memory != data.Memory {
		vcMemoryBytes := data.Memory.ValueInt64() * 1024 * 1024 // MB to B
		updatePayload["mem"] = vcMemoryBytes
	}
	if data_state.VCPU != data.VCPU {
		updatePayload["numVCPU"] = data.VCPU.ValueInt32()
	}

	taskTag, _ := restClient.UpdateRecord( /**/
		fmt.Sprintf("/rest/v1/VirDomain/%s", vm_uuid),
		updatePayload,
		-1,
		ctx,
	)
	taskTag.WaitTask(restClient, ctx)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVMCloneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "TTRT ScaleVMCloneResource DELETE")
	var data ScaleVMCloneResourceModel

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
	vm_uuid := data.Id.ValueString()
	taskTag := restClient.DeleteRecord(
		fmt.Sprintf("/rest/v1/VirDomain/%s", vm_uuid),
		-1,
		ctx,
	)
	taskTag.WaitTask(restClient, ctx)
}

func (r *ScaleVMCloneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "TTRT ScaleVMCloneResource IMPORT_STATE")
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
