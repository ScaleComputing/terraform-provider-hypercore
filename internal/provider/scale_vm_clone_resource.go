// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

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
	DiskSize    types.Int64  `tfsdk:"disk_size"`
	Nics        types.List   `tfsdk:"nics"`
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
				MarkdownDescription: "Number of CPUs on this VM. If the cloned VM was already created and it's VCPU was modified, the cloned VM will be rebooted (either gracefully or forcefully)",
				Optional:            true,
			},
			"memory": schema.Int64Attribute{
				MarkdownDescription: "Memory (RAM) size in MiB: If the cloned VM was already created and it's memory was modified, the cloned VM will be rebooted (either gracefully or forcefully)",
				Optional:            true,
			},
			"disk_size": schema.Int64Attribute{
				MarkdownDescription: "Disk size in GB: If the cloned VM doesn't have a disk already, a new one will be created, otherwise the current disk will be updated with the preferred size. Note that if the cloned VM already has N disks, the first match (by slot or type) will be replaced",
				Optional:            true,
			},
			"nics": schema.ListNestedAttribute{
				MarkdownDescription: "NICs for this VM",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							MarkdownDescription: "NIC type",
							Required:            true,
						},
						"vlan": schema.Int32Attribute{
							MarkdownDescription: "Specific VLAN to use",
							Optional:            true,
						},
					},
				},
				Required: true,
			},
			"power_state": schema.StringAttribute{
				MarkdownDescription: "Initial power state on create: If not provided, it will default to `stop`. Available power states are: start, started, stop, shutdown, reboot, reset. Power state can be modified on the cloned VM even after the cloning process.",
				Optional:            true,
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
	var data ScaleVMCloneResourceModel

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

	// Disk size modifications
	// set the disk size of the new VM
	vmCloneDisk, _ := utils.NewVMDiskForClone(data.DiskSize.ValueInt64Pointer())
	changed, vmWasRebooted, vmDiff := vmCloneDisk.SetSize(vmClone, *r.client, ctx)
	tflog.Info(ctx, fmt.Sprintf("Changed: %t, Was VM Rebooted: %t, Diff: %v", changed, vmWasRebooted, vmDiff))

	// [ ] TODO: 2. set the NICs of the new VM

	// General parametrization
	// set: description, group, vcpu, memory, power_state
	changed, vmWasRebooted, vmDiff = vmClone.SetVMParams(*r.client, ctx)
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

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVMCloneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVMCloneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
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
}

func (r *ScaleVMCloneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
