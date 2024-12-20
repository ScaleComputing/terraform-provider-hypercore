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
var _ resource.Resource = &ScaleVMResource{}
var _ resource.ResourceWithImportState = &ScaleVMResource{}

func NewScaleVMResource() resource.Resource {
	return &ScaleVMResource{}
}

// ScaleVMResource defines the resource implementation.
type ScaleVMResource struct {
	client *utils.RestClient
}

// ScaleVMResourceModel describes the resource data model.
type ScaleVMResourceModel struct {
	Group        types.String `tfsdk:"group"`
	Name         types.String `tfsdk:"name"`
	SourceVMName types.String `tfsdk:"source_vm_name"`
	Description  types.String `tfsdk:"description"`
	VCPU         types.Int32  `tfsdk:"vcpu"`
	Memory       types.Int32  `tfsdk:"memory"`
	DiskSize     types.Int32  `tfsdk:"disk_size"`
	Nics         types.List   `tfsdk:"nics"`
	PowerState   types.String `tfsdk:"power_state"`
	NetworkIface types.String `tfsdk:"network_iface"`
	NetworkMode  types.String `tfsdk:"network_mode"`
	UserData     types.String `tfsdk:"user_data"`
	MetaData     types.String `tfsdk:"meta_data"`
	VMList       types.String `tfsdk:"vm_list"`
	Id           types.String `tfsdk:"id"`
}

func (r *ScaleVMResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (r *ScaleVMResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "ScaleVM resource to create a VM from a template VM",

		Attributes: map[string]schema.Attribute{
			"group": schema.StringAttribute{
				MarkdownDescription: "Group/tag to create this VM in",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of this VM",
				Required:            true,
			},
			"source_vm_name": schema.StringAttribute{
				MarkdownDescription: "Name of the template VM from which this VM will be created",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of this VM",
				Required:            true,
			},
			"vcpu": schema.Int32Attribute{
				MarkdownDescription: "Number of CPUs on this VM",
				Required:            true,
			},
			"memory": schema.Int32Attribute{
				MarkdownDescription: "Memory (RAM) size in MiB",
				Required:            true,
			},
			"disk_size": schema.Int32Attribute{
				MarkdownDescription: "Disk size in GB",
				Required:            true,
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
				MarkdownDescription: "Initial power state on create",
				Optional:            true,
			},
			"network_iface": schema.StringAttribute{
				MarkdownDescription: "Network interface of this VM",
				Required:            true,
			},
			"network_mode": schema.StringAttribute{
				MarkdownDescription: "Network mode for this VM",
				Required:            true,
			},
			"user_data": schema.StringAttribute{
				MarkdownDescription: "User data jinja2 template (.yml.j2)",
				Required:            true,
			},
			"meta_data": schema.StringAttribute{
				MarkdownDescription: "User meta data jinja2 template (.yml.j2)",
				Required:            true,
			},
			"vm_list": schema.StringAttribute{
				MarkdownDescription: "List of VM objects currently on Scale (JSON as string)",
				Computed:            true,
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

func (r *ScaleVMResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScaleVMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ScaleVMResourceModel

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

	// TODO: 0. login with SCALE credentials to be able to access the rest of the endpoints
	vmList := r.client.ListRecords("/rest/v1/VirDomain", nil, -1.0)
	data.VMList = types.StringValue(fmt.Sprintf("%+v", vmList))

	// save into the Terraform state.
	data.Id = types.StringValue("scale-id")

	// TODO: 1. clone a template VM (source VM)

	// TODO: 2. set the disk size of the new VM
	// TODO: 3. set the NICs of the new VM
	// TODO: 4. set new VM params and start it (set it's initial power state)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ScaleVMResourceModel

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

func (r *ScaleVMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ScaleVMResourceModel

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

func (r *ScaleVMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ScaleVMResourceModel

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

func (r *ScaleVMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
