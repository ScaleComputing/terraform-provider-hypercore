// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-hypercore/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &HypercoreVMReplicationResource{}
var _ resource.ResourceWithImportState = &HypercoreVMReplicationResource{}

func NewHypercoreVMReplicationResource() resource.Resource {
	return &HypercoreVMReplicationResource{}
}

// HypercoreVMReplicationResource defines the resource implementation.
type HypercoreVMReplicationResource struct {
	client *utils.RestClient
}

// HypercoreVMReplicationResourceModel describes the resource data model.
type HypercoreVMReplicationResourceModel struct {
	Id             types.String `tfsdk:"id"`
	VmUUID         types.String `tfsdk:"vm_uuid"`
	Label          types.String `tfsdk:"label"`
	ConnectionUUID types.String `tfsdk:"connection_uuid"`
	Enable         types.Bool   `tfsdk:"enable"`
	TargetVmUUID   types.String `tfsdk:"target_vm_uuid"`
}

func (r *HypercoreVMReplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_replication"
}

func (r *HypercoreVMReplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Hypercore VM replication resource to manage VM boot devices' order",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Replication identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"target_vm_uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Remote target VM UUID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vm_uuid": schema.StringAttribute{
				MarkdownDescription: "VM UUID of which we want to make a replication",
				Required:            true,
			},
			"label": schema.StringAttribute{
				MarkdownDescription: "Human-readable label describing the replication purpose",
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"connection_uuid": schema.StringAttribute{
				MarkdownDescription: "Remote connection UUID",
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enable": schema.BoolAttribute{
				MarkdownDescription: "Enable or disable replication",
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *HypercoreVMReplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Info(ctx, "TTRT HypercoreVMReplicationResource CONFIGURE")
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

func (r *HypercoreVMReplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "TTRT HypercoreVMReplicationResource CREATE")
	var data HypercoreVMReplicationResourceModel

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
	vmUUID := data.VmUUID.ValueString()                 // is required
	connectionUUID := data.ConnectionUUID.ValueString() // should be required
	label := data.Label.ValueString()                   // default empty string ""

	enable := data.Enable.ValueBool()
	if data.Enable.IsUnknown() || data.Enable.IsNull() {
		enable = true // default it to true, like in the API
	}

	if connectionUUID == "" {
		resp.Diagnostics.AddError(
			"Missing connection_uuid",
			"Parameter 'connection_uuid' is required for creating a VM replication",
		)
	}

	tflog.Info(ctx, fmt.Sprintf("TTRT Create: vm_uuid=%s, connection_uuid=%s, label=%s, enable=%t", vmUUID, connectionUUID, label, enable))

	replicationUUID, replication, _diag := utils.CreateVMReplication(restClient, vmUUID, connectionUUID, label, enable, ctx)
	if _diag != nil {
		if _diag.Severity() == 0 { // if is error
			resp.Diagnostics.AddError(_diag.Summary(), _diag.Detail())
			return
		}

		// otherwise it's a warning
		resp.Diagnostics.AddWarning(_diag.Summary(), _diag.Detail())
	}

	tflog.Info(ctx, fmt.Sprintf("TTRT Created: vm_uuid=%s, connection_uuid=%s, label=%s, enable=%t, replication=%v", vmUUID, connectionUUID, label, enable, replication))

	targetVmUUID := ""
	if replication["targetDomainUUID"] != nil {
		targetVmUUID = utils.AnyToString(targetVmUUID)
	}

	// TODO: Check if HC3 matches TF
	// save into the Terraform state.
	data.Id = types.StringValue(replicationUUID)
	data.TargetVmUUID = types.StringValue(targetVmUUID)
	data.Enable = types.BoolValue(enable)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "Created a replication")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreVMReplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "TTRT HypercoreVMReplicationResource READ")
	var data HypercoreVMReplicationResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Boot Order read ======================================================================
	restClient := *r.client
	replicationUUID := data.Id.ValueString()
	vmUUID := data.VmUUID.ValueString()

	tflog.Debug(ctx, fmt.Sprintf("TTRT HypercoreVMReplicationResource Read oldState replicationUUID=%s\n", replicationUUID))

	pHc3Replication := utils.GetVMReplicationByUUID(restClient, replicationUUID)
	if pHc3Replication == nil {
		resp.Diagnostics.AddError("VM not found", fmt.Sprintf("VM replication not found - replicationUUID=%s", replicationUUID))
		return
	}
	hc3Replication := *pHc3Replication

	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreVMReplicationResource: vm_uuid=%s, replication_uuid=%s", vmUUID, replicationUUID))

	// save into the Terraform state.
	data.Id = types.StringValue(replicationUUID)
	data.TargetVmUUID = types.StringValue(utils.AnyToString(hc3Replication["targetDomainUUID"]))
	data.VmUUID = types.StringValue(utils.AnyToString(hc3Replication["sourceDomainUUID"]))
	data.ConnectionUUID = types.StringValue(utils.AnyToString(hc3Replication["connectionUUID"]))
	data.Label = types.StringValue(utils.AnyToString(hc3Replication["label"]))
	data.Enable = types.BoolValue(utils.AnyToBool(hc3Replication["enable"]))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreVMReplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "TTRT HypercoreVMReplicationResource UPDATE")
	var data_state HypercoreVMReplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data_state)...)
	var data HypercoreVMReplicationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	restClient := *r.client
	replicationUUID := data.Id.ValueString()
	vmUUID := data.VmUUID.ValueString()
	connectionUUID := data.ConnectionUUID.ValueString()
	label := data.Label.ValueString()

	enable := data.Enable.ValueBool()
	if data.Enable.IsUnknown() || data.Enable.IsNull() {
		enable = true // default it to true, like in the API
	}
	data.Enable = types.BoolValue(enable)

	if connectionUUID == "" {
		resp.Diagnostics.AddError(
			"Missing connection_uuid",
			"Parameter 'connection_uuid' is required for updating a VM replication",
		)
	}

	diag := utils.UpdateVMReplication(restClient, replicationUUID, vmUUID, connectionUUID, label, enable, ctx)
	if diag != nil {
		resp.Diagnostics.AddWarning(diag.Summary(), diag.Detail())
	}

	// TODO: Check if HC3 matches TF
	// Do not trust UpdateVMReplication made what we asked for. Read new power state from HC3.
	pHc3Replication := utils.GetVMReplicationByUUID(restClient, replicationUUID)
	if pHc3Replication == nil {
		msg := fmt.Sprintf("VM replication not found - replicationUUID=%s.", replicationUUID)
		resp.Diagnostics.AddError("VM replication not found", msg)
		return
	}
	newHc3Replication := *pHc3Replication

	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreVMReplicationResource: replication_uuid=%s, replication=%v", replicationUUID, newHc3Replication))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreVMReplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "TTRT HypercoreVMReplicationResource DELETE")
	var data HypercoreVMReplicationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Extra implementation not needed - VirDomainReplication doesn't have a DELETE endpoint

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }
}

func (r *HypercoreVMReplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "TTRT HypercoreVMReplicationResource IMPORT_STATE")

	replicationUUID := req.ID
	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreVMReplicationResource: replicationUUID=%s", replicationUUID))

	restClient := *r.client
	hc3Replication := utils.GetVMReplicationByUUID(restClient, replicationUUID)

	if hc3Replication == nil {
		msg := fmt.Sprintf("VM Replication import, VM not found -  'replication_uuid'='%s'.", req.ID)
		resp.Diagnostics.AddError("VM Replication import error, VM not found", msg)
		return
	}

	vmUUID := utils.AnyToString((*hc3Replication)["sourceDomainUUID"])
	targetVmUUID := utils.AnyToString((*hc3Replication)["targetDomainUUID"])
	connectionUUID := utils.AnyToString((*hc3Replication)["connectionUUID"])
	label := utils.AnyToString((*hc3Replication)["label"])
	enable := utils.AnyToBool((*hc3Replication)["enable"])

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), replicationUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("target_vm_uuid"), targetVmUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vm_uuid"), vmUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("connection_uuid"), connectionUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("label"), label)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("enable"), enable)...)
}
