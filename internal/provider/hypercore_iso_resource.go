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
var _ resource.Resource = &HypercoreISOResource{}
var _ resource.ResourceWithImportState = &HypercoreISOResource{}

func NewHypercoreISOResource() resource.Resource {
	return &HypercoreISOResource{}
}

// HypercoreNicResource defines the resource implementation.
type HypercoreISOResource struct {
	client *utils.RestClient
}

// HypercoreNicResourceModel describes the resource data model.
type HypercoreISOResourceModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	SourceURL types.String `tfsdk:"source_url"`
}

func (r *HypercoreISOResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iso"
}

func (r *HypercoreISOResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "" +
			"Hypercore ISO resource to manage ISO images. <br><br>" +
			"To use this resource, it's recommended to set the environment variable `TF_CLI_ARGS_apply=\"-parallelism=1\"` or pass the `-parallelism` parameter to the `terraform apply`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ISO identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Desired name of the ISO to upload. ISO name must end with '.iso'.",
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

func (r *HypercoreISOResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Info(ctx, "TTRT HypercoreISOResource CONFIGURE")
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

func (r *HypercoreISOResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "TTRT HypercoreNicResource CREATE")
	var data HypercoreISOResourceModel

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

	isoName := data.Name.ValueString()
	isoSourceURL := data.SourceURL.ValueString()

	// STEPS:
	// 1. Create ISO resource (with readForInsert = False)

	// Validate ISO name
	nameDiag := utils.ValidateISOName(isoName)
	if nameDiag != nil {
		resp.Diagnostics.AddError(nameDiag.Summary(), nameDiag.Detail())
		return
	}

	// Validate ISO SourceURL
	sourceURLDiag := utils.ValidateISOSourceURL(isoSourceURL)
	if sourceURLDiag != nil {
		resp.Diagnostics.AddError(sourceURLDiag.Summary(), sourceURLDiag.Detail())
		return
	}

	// Read binary
	isoBinaryData, binDiag := utils.ReadISOBinary(isoSourceURL)
	if binDiag != nil {
		resp.Diagnostics.AddError(binDiag.Summary(), binDiag.Detail())
		return
	}

	// Create
	tflog.Info(ctx, fmt.Sprintf("TTRT Create: name=%s", data.Name.ValueString()))
	isoUUID, iso := utils.CreateISO(*r.client, isoName, false, isoBinaryData, ctx)
	tflog.Info(ctx, fmt.Sprintf("TTRT Created: name=%s, iso_uuid=%s, iso=%v", data.Name.ValueString(), isoUUID, iso))

	// 2. Upload ISO file
	fileSize := len(isoBinaryData)
	tflog.Debug(ctx, fmt.Sprintf("TTRT ISO Upload: source_url=%s, file_size=%d (Bytes)", isoSourceURL, fileSize))
	_, uploadDiag := utils.UploadISO(*r.client, isoUUID, isoBinaryData, ctx)
	if uploadDiag != nil {
		resp.Diagnostics.AddWarning(uploadDiag.Summary(), uploadDiag.Detail())
	}

	// 3. Update ISO resource (change readForInsert = True)
	payload := map[string]any{
		"name":           data.Name.ValueString(),
		"size":           len(isoBinaryData),
		"readyForInsert": true,
	}
	updateDiag := utils.UpdateISO(*r.client, isoUUID, payload, ctx)
	if updateDiag != nil {
		resp.Diagnostics.AddWarning(updateDiag.Summary(), updateDiag.Detail())
	}

	// TODO: Check if HC3 matches TF
	// save into the Terraform state.
	data.Id = types.StringValue(isoUUID)
	// TODO MAC, IP address etc

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource ISO")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreISOResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "TTRT HypercoreISOResource READ")
	var data HypercoreISOResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// ISO read ======================================================================
	restClient := *r.client
	name := data.Name.ValueString()
	isoUUID := data.Id.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("TTRT HypercoreISOResource Read oldState name=%s and id=%s\n", name, isoUUID))

	pISO := utils.GetISOByUUID(restClient, isoUUID)
	if pISO == nil {
		msg := fmt.Sprintf("ISO not found - isoUUID=%s, name=%s.\n", isoUUID, name)
		resp.Diagnostics.AddError("ISO not found\n", msg)
		return
	}
	iso := *pISO
	//
	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreISOResource: name=%s, iso_uuid=%s, iso=%v\n", name, isoUUID, iso))
	// save into the Terraform state.
	data.Id = types.StringValue(isoUUID)
	data.Name = types.StringValue(utils.AnyToString(iso["name"]))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreISOResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "TTRT HypercoreISOResource UPDATE")
	var data_state HypercoreISOResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data_state)...)
	var data HypercoreISOResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	restClient := *r.client
	isoUUID := data.Id.ValueString()
	name := data.Name.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("TTRT HypercoreISOResource Update name=%s iso_uuid=%s REQUESTED", name, isoUUID))
	tflog.Debug(ctx, fmt.Sprintf("TTRT HypercoreISOResource Update name=%s iso_uuid=%s STATE", name, isoUUID))

	updatePayload := map[string]any{
		"name": name,
	}
	diag := utils.UpdateISO(restClient, isoUUID, updatePayload, ctx)
	if diag != nil {
		resp.Diagnostics.AddWarning(diag.Summary(), diag.Detail())
	}

	// TODO: Check if HC3 matches TF
	// Do not trust UpdateNic made what we asked for. Read new NIC state from HC3.
	pISO := utils.GetISOByUUID(restClient, isoUUID)
	if pISO == nil {
		msg := fmt.Sprintf("ISO not found - isoUUID=%s, name=%s.", isoUUID, name)
		resp.Diagnostics.AddError("ISO not found", msg)
		return
	}
	iso := *pISO

	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreISOResource: name=%s, iso_uuid=%s, iso=%v", name, isoUUID, iso))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreISOResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "TTRT HypercoreISOResource DELETE")
	var data HypercoreISOResourceModel

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
	isoUUID := data.Id.ValueString()
	taskTag := restClient.DeleteRecord(
		fmt.Sprintf("/rest/v1/ISO/%s", isoUUID),
		-1,
		ctx,
	)
	taskTag.WaitTask(restClient, ctx)
}

func (r *HypercoreISOResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "TTRT HypercoreISOResource IMPORT_STATE")

	vdUUID := req.ID
	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreISOResource: iso_uuid=%s", vdUUID))

	restClient := *r.client
	hc3ISO := utils.GetISOByUUID(restClient, vdUUID)

	if hc3ISO == nil {
		msg := fmt.Sprintf("ISO import, ISO not found -  'iso_uuid'='%s'.", req.ID)
		resp.Diagnostics.AddError("ISO import error, ISO not found", msg)
		return
	}

	name := utils.AnyToString((*hc3ISO)["name"])
	tflog.Info(ctx, fmt.Sprintf("TTRT uuid=%v, name=%v\n", vdUUID, name))

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), vdUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
}
