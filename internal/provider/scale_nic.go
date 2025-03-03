// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

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
var _ resource.Resource = &ScaleNicResource{}
var _ resource.ResourceWithImportState = &ScaleNicResource{}

func NewScaleNicResource() resource.Resource {
	return &ScaleNicResource{}
}

// ScaleNicResource defines the resource implementation.
type ScaleNicResource struct {
	client *utils.RestClient
}

// ScaleNicResourceModel describes the resource data model.
type ScaleNicResourceModel struct {
	Id     types.String `tfsdk:"id"`
	VmUUID types.String `tfsdk:"vm_uuid"`
	Vlan   types.Int64  `tfsdk:"vlan"`
	Type   types.String `tfsdk:"type"`
	// MacAddress types.String `tfsdk:"type"`
}

func (r *ScaleNicResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nic"
}

func (r *ScaleNicResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "" +
			"Scale NIC resource to manage VM NICs. <br><br>" +
			"To use this resource, it's recommended to set the environment variable `TF_CLI_ARGS_apply=\"-parallelism=1\"` or pass the `-parallelism` parameter to the `terraform apply`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "NIC identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vm_uuid": schema.StringAttribute{
				MarkdownDescription: "VM UUID.",
				Required:            true,
			},
			"vlan": schema.Int64Attribute{
				MarkdownDescription: "NIC VLAN.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "NIC type. Can be: `VIRTIO`, `INTEL_E1000`, `RTL8139`",
				Required:            true,
			},
		},
	}
}

func (r *ScaleNicResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Info(ctx, "TTRT ScaleNicResource CONFIGURE")
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

func (r *ScaleNicResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "TTRT ScaleNicResource CREATE")
	var data ScaleNicResourceModel
	// var readData ScaleNicResourceModel

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

	tflog.Info(ctx, fmt.Sprintf("TTRT Create: vm_uuid=%s, type=%s, vlan=%d", data.VmUUID.ValueString(), data.Type.ValueString(), data.Vlan.ValueInt64()))

	nicUUID, nic := utils.CreateNic(*r.client, data.VmUUID.ValueString(), data.Type.ValueString(), data.Vlan.ValueInt64(), ctx)
	tflog.Info(ctx, fmt.Sprintf("TTRT Created: vm_uuid=%s, nic_uuid=%s, nic=%v", data.VmUUID.ValueString(), nicUUID, nic))

	// TODO: Check if HC3 matches TF
	// save into the Terraform state.
	data.Id = types.StringValue(nicUUID)
	// TODO MAC, IP address etc

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource NIC")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleNicResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "TTRT ScaleNicResource READ")
	var data ScaleNicResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// NIC read ======================================================================
	restClient := *r.client
	vmUUID := data.VmUUID.ValueString()
	nicUUID := data.Id.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("TTRT ScaleNicResource Read oldState vmUUID=%s\n", vmUUID))

	pNic := utils.GetNic(restClient, nicUUID)
	if pNic == nil {
		msg := fmt.Sprintf("NIC not found - nicUUID=%s, vmUUID=%s.\n", nicUUID, vmUUID)
		resp.Diagnostics.AddError("NIC not found\n", msg)
		return
	}
	nic := *pNic
	//
	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleNicResource: vm_uuid=%s, nic_uuid=%s, nic=%v\n", vmUUID, nicUUID, nic))
	// save into the Terraform state.
	data.Id = types.StringValue(nicUUID)
	data.VmUUID = types.StringValue(utils.AnyToString(nic["virDomainUUID"]))
	data.Type = types.StringValue(utils.AnyToString(nic["type"]))
	data.Vlan = types.Int64Value(utils.AnyToInteger64(nic["vlan"]))
	// TODO MAC, IP address etc

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleNicResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "TTRT ScaleNicResource UPDATE")
	var data_state ScaleNicResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data_state)...)
	var data ScaleNicResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	restClient := *r.client
	nicUUID := data.Id.ValueString()
	vmUUID := data.VmUUID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("TTRT ScaleNicResource Update vm_uuid=%s nic_uuid=%s REQUESTED vlan=%d type=%s", vmUUID, nicUUID, data.Vlan.ValueInt64(), data.Type.String()))
	tflog.Debug(ctx, fmt.Sprintf("TTRT ScaleNicResource Update vm_uuid=%s nic_uuid=%s STATE     vlan=%d type=%s", vmUUID, nicUUID, data_state.Vlan.ValueInt64(), data_state.Type.String()))

	updatePayload := map[string]any{
		"virDomainUUID": vmUUID,
		"type":          data.Type.ValueString(),
		"vlan":          data.Vlan.ValueInt64(),
	}
	diag := utils.UpdateNic(restClient, nicUUID, updatePayload, ctx)
	if diag != nil {
		resp.Diagnostics.AddWarning(diag.Summary(), diag.Detail())
	}

	// TODO: Check if HC3 matches TF
	// Do not trust UpdateNic made what we asked for. Read new NIC state from HC3.
	pNic := utils.GetNic(restClient, nicUUID)
	if pNic == nil {
		msg := fmt.Sprintf("NIC not found - nicUUID=%s, vmUUID=%s.", nicUUID, vmUUID)
		resp.Diagnostics.AddError("NIC not found", msg)
		return
	}
	nic := *pNic
	//
	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleNicResource: vm_uuid=%s, nic_uuid=%s, nic=%v", vmUUID, nicUUID, nic))

	// TODO MAC, IP address etc

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleNicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "TTRT ScaleNicResource DELETE")
	var data ScaleNicResourceModel

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
	nicUUID := data.Id.ValueString()
	taskTag := restClient.DeleteRecord(
		fmt.Sprintf("/rest/v1/VirDomainNetDevice/%s", nicUUID),
		-1,
		ctx,
	)
	taskTag.WaitTask(restClient, ctx)
}

func (r *ScaleNicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "TTRT ScaleNicResource IMPORT_STATE")
	idParts := strings.Split(req.ID, ":")
	if len(idParts) != 3 {
		msg := fmt.Sprintf("NIC import composite ID format is 'vm_uuid:nic_type:nic_slot'. ID='%s' is invalid.", req.ID)
		resp.Diagnostics.AddError("NIC import requires a composite ID", msg)
		return
	}
	vmUUID := idParts[0]
	nicType := idParts[1]
	vlan := utils.AnyToInteger64(idParts[2])
	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleNicResource: vmUUID=%s, type=%s, vlan=%d", vmUUID, nicType, vlan))

	restClient := *r.client
	hc3VM := utils.GetOneVM(vmUUID, restClient)
	hc3Nics := utils.AnyToListOfMap(hc3VM["netDevs"])
	tflog.Info(ctx, fmt.Sprintf("TTRT hc3Nics=%v\n", hc3Nics))

	var nicUUID string
	for _, nic := range hc3Nics {
		if utils.AnyToInteger64(nic["vlan"]) == vlan &&
			utils.AnyToString(nic["type"]) == nicType {
			nicUUID = utils.AnyToString(nic["uuid"])
			break
		}
	}
	if nicUUID == "" {
		msg := fmt.Sprintf("NIC import, NIC not found -  'vm_uuid:nic_type:nic_slot'='%s'.", req.ID)
		resp.Diagnostics.AddError("NIC import error, NIC not found", msg)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), nicUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vm_uuid"), vmUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), nicType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vlan"), vlan)...)
}
