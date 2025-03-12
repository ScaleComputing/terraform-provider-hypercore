// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
var _ resource.Resource = &ScaleVMBootOrderResource{}
var _ resource.ResourceWithImportState = &ScaleVMBootOrderResource{}

func NewScaleVMBootOrderResource() resource.Resource {
	return &ScaleVMBootOrderResource{}
}

// ScaleVMBootOrderResource defines the resource implementation.
type ScaleVMBootOrderResource struct {
	client *utils.RestClient
}

// ScaleVMBootOrderResourceModel describes the resource data model.
type ScaleVMBootOrderResourceModel struct {
	Id          types.String `tfsdk:"id"`
	VmUUID      types.String `tfsdk:"vm_uuid"`
	BootDevices types.List   `tfsdk:"boot_devices"`
}

func (r *ScaleVMBootOrderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_boot_order"
}

func (r *ScaleVMBootOrderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Scale VM boot order resource to manage VM boot devices' order",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Boot order identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vm_uuid": schema.StringAttribute{
				MarkdownDescription: "VM UUID of which we want to set the boot order.",
				Required:            true,
			},
			"boot_devices": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of UUIDs of disks and NICs, in the order that they will boot",
				Required:            true,
			},
		},
	}
}

func (r *ScaleVMBootOrderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Info(ctx, "TTRT ScaleVMBootOrderResource CONFIGURE")
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

func (r *ScaleVMBootOrderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "TTRT ScaleVMBootOrderResource CREATE")
	var data ScaleVMBootOrderResourceModel

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

	var vmBootDevices []string
	diags := data.BootDevices.ElementsAs(ctx, &vmBootDevices, false)
	if diags.HasError() {
		resp.Diagnostics.Append(diags.Errors()...)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("TTRT Create: vm_uuid=%s, boot_devices=%v", vmUUID, vmBootDevices))

	diag := utils.ModifyVMBootOrder(restClient, vmUUID, vmBootDevices, ctx)
	if diag != nil {
		resp.Diagnostics.AddWarning(diag.Summary(), diag.Detail())
	}

	tflog.Info(ctx, fmt.Sprintf("TTRT Created: vm_uuid=%s, boot_devices=%s", vmUUID, vmBootDevices))

	// TODO: Check if HC3 matches TF
	// save into the Terraform state.
	data.Id = types.StringValue(vmUUID)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "Changed the boot order")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVMBootOrderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "TTRT ScaleVMBootOrderResource READ")
	var data ScaleVMBootOrderResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Boot Order read ======================================================================
	restClient := *r.client
	vmUUID := data.VmUUID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("TTRT ScaleVMBootOrderResource Read oldState vmUUID=%s\n", vmUUID))

	pHc3VM, err := utils.GetOneVMWithError(vmUUID, restClient)
	if err != nil {
		resp.Diagnostics.AddError("VM not found", fmt.Sprintf("VM not found - vmUUID=%s", vmUUID))
		return
	}
	hc3VM := *pHc3VM

	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleVMBootOrderResource: vm_uuid=%s, boot_devices=%v\n", vmUUID, data.BootDevices.Elements()))

	// save into the Terraform state.
	data.Id = types.StringValue(vmUUID)
	data.VmUUID = types.StringValue(utils.AnyToString(hc3VM["uuid"]))

	hc3BootDevices := utils.AnyToListOfStrings(hc3VM["bootDevices"])
	bootDeviceValues := make([]attr.Value, len(hc3BootDevices))
	for i, dev := range hc3BootDevices {
		bootDeviceValues[i] = types.StringValue(dev)
	}

	var diags diag.Diagnostics
	data.BootDevices, diags = types.ListValue(types.StringType, bootDeviceValues)
	if diags.HasError() {
		resp.Diagnostics.Append(diags.Errors()...)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVMBootOrderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "TTRT ScaleVMBootOrderResource UPDATE")
	var data_state ScaleVMBootOrderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data_state)...)
	var data ScaleVMBootOrderResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	restClient := *r.client
	vmUUID := data.VmUUID.ValueString()

	var vmBootDevices []string
	diags := data.BootDevices.ElementsAs(ctx, &vmBootDevices, false)
	if diags.HasError() {
		resp.Diagnostics.Append(diags.Errors()...)
		return
	}

	diag := utils.ModifyVMBootOrder(restClient, vmUUID, vmBootDevices, ctx)
	if diag != nil {
		resp.Diagnostics.AddWarning(diag.Summary(), diag.Detail())
	}

	// TODO: Check if HC3 matches TF
	// Do not trust UpdateVMBootOrder made what we asked for. Read new power state from HC3.
	pHc3VM, err := utils.GetOneVMWithError(vmUUID, restClient)
	if err != nil {
		msg := fmt.Sprintf("VM not found - vmUUID=%s.", vmUUID)
		resp.Diagnostics.AddError("VM not found", msg)
		return
	}
	newHc3VM := *pHc3VM

	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleVMBootOrderResource: vm_uuid=%s, boot_devices=%v", vmUUID, newHc3VM["bootDevices"]))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVMBootOrderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "TTRT ScaleVMBootOrderResource DELETE")
	var data ScaleVMBootOrderResourceModel

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

func (r *ScaleVMBootOrderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "TTRT ScaleVMBootOrderResource IMPORT_STATE")

	vmUUID := req.ID
	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleVMBootOrderResource: vmUUID=%s", vmUUID))

	restClient := *r.client
	hc3VM, err := utils.GetOneVMWithError(vmUUID, restClient)

	if err != nil {
		msg := fmt.Sprintf("VM Boot Order import, VM not found -  'vm_uuid'='%s'.", req.ID)
		resp.Diagnostics.AddError("VM Boot Order import error, VM not found", msg)
		return
	}

	bootDevicesOrder := utils.AnyToListOfStrings((*hc3VM)["bootDevices"])
	tflog.Info(ctx, fmt.Sprintf("TTRT boot_devices=%v\n", bootDevicesOrder))

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), vmUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vm_uuid"), vmUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("boot_devices"), bootDevicesOrder)...)
}
