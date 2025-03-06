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
var _ resource.Resource = &ScaleVirtualDiskResource{}
var _ resource.ResourceWithImportState = &ScaleVirtualDiskResource{}

func NewScaleVirtualDiskResource() resource.Resource {
	return &ScaleVirtualDiskResource{}
}

// ScaleVirtualDiskResource defines the resource implementation.
type ScaleVirtualDiskResource struct {
	client *utils.RestClient
}

// ScaleVirtualDiskResourceModel describes the resource data model.
type ScaleVirtualDiskResourceModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	SourceURL types.String `tfsdk:"source_url"`
}

func (r *ScaleVirtualDiskResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_disk"
}

func (r *ScaleVirtualDiskResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Scale virtual disk resource to manage VM virtual disks",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Virtual disk identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Desired name of the virtual disk to upload.",
				Required:            true,
			},
			"source_url": schema.StringAttribute{
				MarkdownDescription: "Source URL from where to fetch that disk from. URL can start with: `http://`, `https://`, `file:///`",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ScaleVirtualDiskResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Info(ctx, "TTRT ScaleVirtualDiskResource CONFIGURE")
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

func (r *ScaleVirtualDiskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "TTRT ScaleVirtualDiskResource CREATE")
	var data ScaleVirtualDiskResourceModel

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

	// Validate the SourceURL (check if it's in the supported URL types)
	diagSourceURL := utils.ValidateVirtualDiskSourceURL(data.SourceURL.ValueString())
	if diagSourceURL != nil {
		resp.Diagnostics.AddError(diagSourceURL.Summary(), diagSourceURL.Detail())
		return
	}

	tflog.Info(ctx, fmt.Sprintf("TTRT Create: name=%s, source=%s", data.Name.ValueString(), data.SourceURL.ValueString()))

	restClient := *r.client
	vdUUID, virtualDisk, diag := utils.UploadVirtualDisk(restClient, data.Name.ValueString(), data.SourceURL.ValueString(), ctx)
	if diag != nil {
		resp.Diagnostics.AddError(diag.Summary(), diag.Detail())
		return
	}

	tflog.Info(ctx, fmt.Sprintf("TTRT Created: vd_uuid=%s, name=%s, source_url=%s, virtual_disk=%v", vdUUID, data.Name.ValueString(), data.SourceURL.ValueString(), virtualDisk))

	// TODO: Check if HC3 matches TF
	// save into the Terraform state.
	data.Id = types.StringValue(utils.AnyToString(vdUUID))

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "Uploaded virtual disk")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVirtualDiskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "TTRT ScaleVirtualDiskResource READ")
	var data ScaleVirtualDiskResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Virtual Disk read ======================================================================
	restClient := *r.client
	vdUUID := data.Id.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("TTRT ScaleVirtualDiskResource Read oldState vdUUID=%s\n", vdUUID))

	pHc3VD := utils.GetVirtualDiskByUUID(restClient, vdUUID)
	if pHc3VD == nil {
		resp.Diagnostics.AddError("Virtual Disk not found", fmt.Sprintf("Virtual Disk not found - vdUUID=%s", vdUUID))
		return
	}
	hc3VD := *pHc3VD

	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleVirtualDiskResource: vd_uuid=%s, name=%s, source_url=%s\n", vdUUID, data.Name.ValueString(), data.SourceURL.ValueString()))

	// save into the Terraform state.
	data.Id = types.StringValue(vdUUID)
	data.Name = types.StringValue(utils.AnyToString(hc3VD["name"]))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVirtualDiskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "TTRT ScaleVirtualDiskResource UPDATE")
	var data_state ScaleVirtualDiskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data_state)...)
	var data ScaleVirtualDiskResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	restClient := *r.client
	vdUUID := data.Id.ValueString()
	vdName := data.Name.ValueString()
	tflog.Debug(
		ctx, fmt.Sprintf(
			"TTRT ScaleVirtualDiskResource Update vd_uuid=%s REQUESTED name=%s\n",
			vdUUID, vdName),
	)
	tflog.Debug(ctx, fmt.Sprintf(
		"TTRT ScaleVirtualDiskResource Update vd_uuid=%s STATE     name=%s\n",
		vdUUID, data_state.Name.ValueString()),
	)

	vdHC3 := utils.GetVirtualDiskByUUID(restClient, vdUUID)

	// NOTE: If disk already exists, leave it unmodified - do not modify it, even if say file content or file length is different.
	// - In case of Update method, disk should already exist, so here, nothing will happen, but will still fetch the disk from hc3

	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleVirtualDiskResource: vd_uuid=%s, name=%s, virtual_disk=%v", vdUUID, vdName, vdHC3))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleVirtualDiskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "TTRT ScaleVirtualDiskResource DELETE")
	var data ScaleVirtualDiskResourceModel

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
	vdUUID := data.Id.ValueString()
	taskTag := restClient.DeleteRecord(
		fmt.Sprintf("/rest/v1/VirtualDisk/%s", vdUUID),
		-1,
		ctx,
	)
	taskTag.WaitTask(restClient, ctx)
}

func (r *ScaleVirtualDiskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "TTRT ScaleVirtualDiskResource IMPORT_STATE")

	vdUUID := req.ID
	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleVirtualDiskResource: vd_uuid=%s", vdUUID))

	restClient := *r.client
	hc3VD := utils.GetVirtualDiskByUUID(restClient, vdUUID)

	if hc3VD == nil {
		msg := fmt.Sprintf("Virtual Disk import, virtual disk not found -  'vd_uuid'='%s'.", req.ID)
		resp.Diagnostics.AddError("Virtual Disk import error, virtual disk not found", msg)
		return
	}

	name := utils.AnyToString((*hc3VD)["name"])
	tflog.Info(ctx, fmt.Sprintf("TTRT uuid=%v, name=%v\n", vdUUID, name))

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), vdUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)

	// In case of importing an existing state of the VirtualDisk, should the source_url be same as the name?
	// resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("source_url"), name)...)
}
