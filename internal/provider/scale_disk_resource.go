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
var _ resource.Resource = &ScaleDiskResource{}
var _ resource.ResourceWithImportState = &ScaleDiskResource{}

func NewScaleDiskResource() resource.Resource {
	return &ScaleDiskResource{}
}

// ScaleDiskResource defines the resource implementation.
type ScaleDiskResource struct {
	client *utils.RestClient
}

// ScaleDiskResourceModel describes the resource data model.
type ScaleDiskResourceModel struct {
	Id     types.String  `tfsdk:"id"`
	VmUUID types.String  `tfsdk:"vm_uuid"`
	Slot   types.Int64   `tfsdk:"slot"`
	Type   types.String  `tfsdk:"type"`
	Size   types.Float64 `tfsdk:"size"`
	// MacAddress types.String `tfsdk:"type"`
}

func (r *ScaleDiskResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_disk"
}

func (r *ScaleDiskResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Scale disk resource to manage VM disks",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Disk identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vm_uuid": schema.StringAttribute{
				MarkdownDescription: "VM UUID.",
				Required:            true,
			},
			"slot": schema.Int64Attribute{
				MarkdownDescription: "Disk slot number. Will not do anything if the disk already exists, since HC3 doesn't change disk slots to existing disks.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Disk type. Can be: `IDE_DISK`, `SCSI_DISK`, `VIRTIO_DISK`, `IDE_FLOPPY`, `NVRAM`, `VTPM`",
				Optional:            true,
			},
			"size": schema.Float64Attribute{
				MarkdownDescription: "Disk size in `GB`. Must be larger than the current size of the disk if specified.",
				Optional:            true,
			},
		},
	}
}

func (r *ScaleDiskResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Info(ctx, "TTRT ScaleDiskResource CONFIGURE")
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

func (r *ScaleDiskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "TTRT ScaleDiskResource CREATE")
	var data ScaleDiskResourceModel
	// var readData ScaleDiskResourceModel

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

	tflog.Info(ctx, fmt.Sprintf("TTRT Create: vm_uuid=%s, type=%s, slot=%d, size=%d", data.VmUUID.ValueString(), data.Type.ValueString(), data.Slot.ValueInt64(), data.Slot.ValueInt64()))

	diagDiskType := utils.ValidateDiskType(data.Type.ValueString())
	if diagDiskType != nil {
		resp.Diagnostics.AddError(diagDiskType.Summary(), diagDiskType.Detail())
		return
	}

	createPayload := map[string]any{
		"virDomainUUID": data.VmUUID.ValueString(),
		"type":          data.Type.ValueString(),
		"capacity":      data.Size.ValueFloat64() * 1000 * 1000 * 1000, // GB to B
	}
	diskUUID, disk := utils.CreateDisk(*r.client, createPayload, ctx)
	tflog.Info(ctx, fmt.Sprintf("TTRT Created: vm_uuid=%s, disk_uuid=%s, disk=%v", data.VmUUID.ValueString(), diskUUID, disk))

	// TODO: Check if HC3 matches TF
	// save into the Terraform state.
	data.Id = types.StringValue(diskUUID)
	data.Slot = types.Int64Value(utils.AnyToInteger64(disk["slot"]))
	// TODO MAC, IP address etc

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource Disk")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleDiskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "TTRT ScaleDiskResource READ")
	var data ScaleDiskResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Disk read ======================================================================
	restClient := *r.client
	vmUUID := data.VmUUID.ValueString()
	diskUUID := data.Id.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("TTRT ScaleDiskResource Read oldState vmUUID=%s\n", vmUUID))

	pDisk := utils.GetDiskByUUID(restClient, diskUUID)
	if pDisk == nil {
		msg := fmt.Sprintf("Disk not found - diskUUID=%s, vmUUID=%s.\n", diskUUID, vmUUID)
		resp.Diagnostics.AddError("Disk not found\n", msg)
		return
	}
	disk := *pDisk

	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleDiskResource: vm_uuid=%s, disk_uuid=%s, disk=%v\n", vmUUID, diskUUID, disk))
	// save into the Terraform state.
	data.Id = types.StringValue(diskUUID)
	data.VmUUID = types.StringValue(utils.AnyToString(disk["virDomainUUID"]))
	data.Type = types.StringValue(utils.AnyToString(disk["type"]))
	data.Slot = types.Int64Value(utils.AnyToInteger64(disk["slot"]))
	data.Size = types.Float64Value(utils.AnyToFloat64(disk["capacity"]) / 1000 / 1000 / 1000)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleDiskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "TTRT ScaleDiskResource UPDATE")
	var data_state ScaleDiskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data_state)...)
	var data ScaleDiskResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	restClient := *r.client
	diskUUID := data.Id.ValueString()
	vmUUID := data.VmUUID.ValueString()
	tflog.Debug(
		ctx, fmt.Sprintf(
			"TTRT ScaleDiskResource Update vm_uuid=%s disk_uuid=%s REQUESTED slot=%d type=%s size=%v\n",
			vmUUID, diskUUID, data.Slot.ValueInt64(), data.Type.String(), data.Size.ValueFloat64()),
	)
	tflog.Debug(ctx, fmt.Sprintf(
		"TTRT ScaleDiskResource Update vm_uuid=%s disk_uuid=%s STATE     slot=%d type=%s size=%v\n",
		vmUUID, diskUUID, data_state.Slot.ValueInt64(), data_state.Type.String(), data.Size.ValueFloat64()),
	)

	// Get the disk before update
	pDisk := utils.GetDiskByUUID(restClient, diskUUID)
	if pDisk == nil {
		msg := fmt.Sprintf("Disk not found - diskUUID=%s, vmUUID=%s.", diskUUID, vmUUID)
		resp.Diagnostics.AddError("Disk not found", msg)
		return
	}
	oldHc3Disk := *pDisk

	// Validate the size
	oldDiskSize := utils.AnyToFloat64(oldHc3Disk["capacity"]) / 1000 / 1000 / 1000 // B to GB
	wantedDiskSize := data.Size.ValueFloat64()
	diagDiskSize := utils.ValidateDiskSize(data.Id.ValueString(), oldDiskSize, wantedDiskSize)

	if diagDiskSize != nil {
		resp.Diagnostics.AddError(diagDiskSize.Summary(), diagDiskSize.Detail())
		return
	}

	// Validate the type
	diagDiskType := utils.ValidateDiskType(data.Type.ValueString())
	if diagDiskType != nil {
		resp.Diagnostics.AddError(diagDiskType.Summary(), diagDiskType.Detail())
		return
	}

	updatePayload := map[string]any{
		"virDomainUUID": vmUUID,
		"type":          data.Type.ValueString(),
		"capacity":      data.Size.ValueFloat64() * 1000 * 1000 * 1000, // GB to B
	}
	diag := utils.UpdateDisk(restClient, diskUUID, updatePayload, ctx)
	if diag != nil {
		resp.Diagnostics.AddWarning(diag.Summary(), diag.Detail())
	}

	// TODO: Check if HC3 matches TF
	// Do not trust UpdateDisk made what we asked for. Read new Disk state from HC3.
	pDisk = utils.GetDiskByUUID(restClient, diskUUID)
	if pDisk == nil {
		msg := fmt.Sprintf("Disk not found - diskUUID=%s, vmUUID=%s.", diskUUID, vmUUID)
		resp.Diagnostics.AddError("Disk not found", msg)
		return
	}
	newHc3Disk := *pDisk

	data.Slot = types.Int64Value(utils.AnyToInteger64(newHc3Disk["slot"]))
	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleDiskResource: vm_uuid=%s, disk_uuid=%s, disk=%v", vmUUID, diskUUID, newHc3Disk))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScaleDiskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "TTRT ScaleDiskResource DELETE")
	var data ScaleDiskResourceModel

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
	diskUUID := data.Id.ValueString()
	taskTag := restClient.DeleteRecord(
		fmt.Sprintf("/rest/v1/VirDomainBlockDevice/%s", diskUUID),
		-1,
		ctx,
	)
	taskTag.WaitTask(restClient, ctx)
}

func (r *ScaleDiskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "TTRT ScaleDiskResource IMPORT_STATE")
	idParts := strings.Split(req.ID, ":")
	if len(idParts) != 3 {
		msg := fmt.Sprintf("Disk import composite ID format is 'vm_uuid:disk_type:disk_slot'. ID='%s' is invalid.", req.ID)
		resp.Diagnostics.AddError("Disk import requires a composite ID", msg)
		return
	}
	vmUUID := idParts[0]
	diskType := idParts[1]
	slot := utils.AnyToInteger64(idParts[2])
	tflog.Info(ctx, fmt.Sprintf("TTRT ScaleDiskResource: vmUUID=%s, type=%s, slot=%d", vmUUID, diskType, slot))

	restClient := *r.client
	hc3VM := utils.GetOneVM(vmUUID, restClient)
	hc3Disks := utils.AnyToListOfMap(hc3VM["blockDevs"])
	tflog.Info(ctx, fmt.Sprintf("TTRT hc3Disks=%v\n", hc3Disks))

	var diskUUID string
	var size float64
	for _, disk := range hc3Disks {
		if utils.AnyToInteger64(disk["slot"]) == slot &&
			utils.AnyToString(disk["type"]) == diskType {
			diskUUID = utils.AnyToString(disk["uuid"])
			size = utils.AnyToFloat64(disk["capacity"]) / 1000 / 1000 / 1000 // hc3 has B, so convert to GB
			break
		}
	}
	if diskUUID == "" {
		msg := fmt.Sprintf("Disk import, Disk not found -  'vm_uuid:disk_type:disk_slot'='%s'.", req.ID)
		resp.Diagnostics.AddError("Disk import error, Disk not found", msg)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), diskUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vm_uuid"), vmUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), diskType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slot"), slot)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("size"), size)...)
}
