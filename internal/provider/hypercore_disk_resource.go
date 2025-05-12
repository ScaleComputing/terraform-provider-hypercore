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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-hypercore/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &HypercoreDiskResource{}
var _ resource.ResourceWithImportState = &HypercoreDiskResource{}

func NewHypercoreDiskResource() resource.Resource {
	return &HypercoreDiskResource{}
}

// HypercoreDiskResource defines the resource implementation.
type HypercoreDiskResource struct {
	client *utils.RestClient
}

// HypercoreDiskResourceModel describes the resource data model.
type HypercoreDiskResourceModel struct {
	Id                  types.String  `tfsdk:"id"`
	VmUUID              types.String  `tfsdk:"vm_uuid"`
	Slot                types.Int64   `tfsdk:"slot"`
	FlashPriority       types.Int64   `tfsdk:"flash_priority"`
	Type                types.String  `tfsdk:"type"`
	Size                types.Float64 `tfsdk:"size"`
	SourceVirtualDiskID types.String  `tfsdk:"source_virtual_disk_id"`
	IsoUUID             types.String  `tfsdk:"iso_uuid"`
}

func (r *HypercoreDiskResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_disk"
}

func (r *HypercoreDiskResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "" +
			"Hypercore disk resource to manage VM disks. <br><br>" +
			"To use this resource, it's recommended to set the environment variable `TF_CLI_ARGS_apply=\"-parallelism=1\"` or pass the `-parallelism` parameter to the `terraform apply`." +
			"<br><br> Removing disk from a running VM is (often) not possible. In this case it is required to shutdown the VM before disk removal." +
			"",
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
			"flash_priority": schema.Int64Attribute{
				MarkdownDescription: "SSD tiering priority factor for block placement. If not provided, it will default to `4`, unless imported, in which case the disk's current flash priority will be taken into account and can then be modified. This can be any **positive** value between (including) `0` and `11`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(4),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Disk type. Can be: `IDE_DISK`, `IDE_CDROM`, `SCSI_DISK`, `VIRTIO_DISK`, `IDE_FLOPPY`, `NVRAM`, `VTPM`",
				Optional:            true,
			},
			"size": schema.Float64Attribute{
				MarkdownDescription: "Disk size in `GB`. Must be larger than the current size of the disk if specified.",
				Optional:            true,
			},
			"source_virtual_disk_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the virtual disk to use to clone and attach to the VM.",
				Optional:            true,
			},
			"iso_uuid": schema.StringAttribute{
				MarkdownDescription: "ISO UUID we want to attach to the disk, only available with disk type `IDE_CDROM`.",
				Optional:            true,
			},
		},
	}
}

func (r *HypercoreDiskResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Info(ctx, "TTRT HypercoreDiskResource CONFIGURE")
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

func (r *HypercoreDiskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	tflog.Info(ctx, "TTRT HypercoreDiskResource CREATE")
	var data HypercoreDiskResourceModel
	// var readData HypercoreDiskResourceModel

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

	var diskUUID string
	var disk map[string]any
	isAttachingISO := data.IsoUUID.ValueString() != ""

	diagDiskType := utils.ValidateDiskType(data.Type.ValueString(), data.IsoUUID.ValueString())
	if diagDiskType != nil {
		resp.Diagnostics.AddError(diagDiskType.Summary(), diagDiskType.Detail())
		return
	}
	diagISOAttach, iso := utils.ValidateISOAttach(*r.client, data.IsoUUID.ValueString(), isAttachingISO)
	if diagISOAttach != nil {
		resp.Diagnostics.AddError(diagISOAttach.Summary(), diagISOAttach.Detail())
		return
	}
	diagFlashPriority := utils.ValidateDiskFlashPriority(data.FlashPriority.ValueInt64())
	if diagFlashPriority != nil {
		resp.Diagnostics.AddError(diagFlashPriority.Summary(), diagFlashPriority.Detail())
		return
	}

	createPayload := map[string]any{
		"virDomainUUID":         data.VmUUID.ValueString(),
		"type":                  data.Type.ValueString(),
		"capacity":              data.Size.ValueFloat64() * 1000 * 1000 * 1000, // GB to B
		"tieringPriorityFactor": utils.FROM_HUMAN_PRIORITY_FACTOR[data.FlashPriority.ValueInt64()],
	}
	if isAttachingISO {
		createPayload["path"] = (*iso)["path"]
	}

	sourceVirtualDiskID := data.SourceVirtualDiskID.ValueString()
	if sourceVirtualDiskID != "" {
		sourceVirtualDiskHC3 := utils.GetVirtualDiskByUUID(*r.client, sourceVirtualDiskID)
		if sourceVirtualDiskHC3 == nil {
			resp.Diagnostics.AddError("Virtual disk not found", fmt.Sprintf("Virtual disk with UUID '%s' not found. Double check your Terraform configuration.", sourceVirtualDiskID))
			return
		}

		// First attach with original size
		originalVDSizeBytes := utils.AnyToInteger64((*sourceVirtualDiskHC3)["capacityBytes"])
		attachPayload := map[string]any{
			"options": map[string]any{
				"regenerateDiskID": false,
				"readOnly":         false,
			},
			"template": map[string]any{
				"virDomainUUID":         data.VmUUID.ValueString(),
				"type":                  data.Type.ValueString(),
				"capacity":              originalVDSizeBytes,
				"tieringPriorityFactor": utils.FROM_HUMAN_PRIORITY_FACTOR[data.FlashPriority.ValueInt64()],
			},
		}

		diskUUID, disk = utils.AttachVirtualDisk(
			*r.client,
			attachPayload,
			sourceVirtualDiskID,
			ctx,
		)
		tflog.Debug(ctx, fmt.Sprintf(
			"TTRT Attach: Attached with original size - vm_uuid=%s, disk_uuid=%s, original_size=%v (GB), source_virtual_disk_uuid=%s",
			data.VmUUID.ValueString(), diskUUID, float64(originalVDSizeBytes/1000/1000/1000), sourceVirtualDiskID),
		)

		// Then resize to desired size
		diag := utils.UpdateDisk(*r.client, diskUUID, createPayload, ctx)
		if diag != nil {
			resp.Diagnostics.AddWarning(diag.Summary(), diag.Detail())
		}
		tflog.Debug(ctx, fmt.Sprintf(
			"TTRT Attach: Resized to desired size - vm_uuid=%s, disk_uuid=%s, desired_size=%v (GB), source_virtual_disk_uuid=%s",
			data.VmUUID.ValueString(), diskUUID, data.Size.ValueFloat64(), sourceVirtualDiskID),
		)

		tflog.Info(ctx, fmt.Sprintf("TTRT Created: vm_uuid=%s, disk_uuid=%s, disk=%v, source_virtual_disk_uuid=%s", data.VmUUID.ValueString(), diskUUID, disk, sourceVirtualDiskID))
	} else {
		diskUUID, disk = utils.CreateDisk(*r.client, createPayload, ctx)
		tflog.Info(ctx, fmt.Sprintf("TTRT Created: vm_uuid=%s, disk_uuid=%s, disk=%v", data.VmUUID.ValueString(), diskUUID, disk))
	}

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

func (r *HypercoreDiskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	tflog.Info(ctx, "TTRT HypercoreDiskResource READ")
	var data HypercoreDiskResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Disk read ======================================================================
	restClient := *r.client
	vmUUID := data.VmUUID.ValueString()
	diskUUID := data.Id.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("TTRT HypercoreDiskResource Read oldState vmUUID=%s\n", vmUUID))

	pDisk := utils.GetDiskByUUID(restClient, diskUUID)
	if pDisk == nil {
		msg := fmt.Sprintf("Disk not found - diskUUID=%s, vmUUID=%s.\n", diskUUID, vmUUID)
		resp.Diagnostics.AddError("Disk not found\n", msg)
		return
	}
	disk := *pDisk

	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreDiskResource: vm_uuid=%s, disk_uuid=%s, disk=%v\n", vmUUID, diskUUID, disk))
	// save into the Terraform state.
	data.Id = types.StringValue(diskUUID)
	data.VmUUID = types.StringValue(utils.AnyToString(disk["virDomainUUID"]))
	data.Type = types.StringValue(utils.AnyToString(disk["type"]))
	data.Slot = types.Int64Value(utils.AnyToInteger64(disk["slot"]))
	data.Size = types.Float64Value(utils.AnyToFloat64(disk["capacity"]) / 1000 / 1000 / 1000)

	hc3PriorityFactor := utils.AnyToInteger64(disk["tieringPriorityFactor"])
	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreDiskResource: hc3PriorityFactor = %v\n", hc3PriorityFactor))
	data.FlashPriority = types.Int64Value(utils.TO_HUMAN_PRIORITY_FACTOR[hc3PriorityFactor])

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreDiskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	tflog.Info(ctx, "TTRT HypercoreDiskResource UPDATE")
	var data_state HypercoreDiskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data_state)...)
	var data HypercoreDiskResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	restClient := *r.client
	diskUUID := data.Id.ValueString()
	vmUUID := data.VmUUID.ValueString()
	isAttachingISO := data.IsoUUID.ValueString() != ""

	tflog.Debug(
		ctx, fmt.Sprintf(
			"TTRT HypercoreDiskResource Update vm_uuid=%s disk_uuid=%s REQUESTED slot=%d type=%s size=%v\n",
			vmUUID, diskUUID, data.Slot.ValueInt64(), data.Type.String(), data.Size.ValueFloat64()),
	)
	tflog.Debug(ctx, fmt.Sprintf(
		"TTRT HypercoreDiskResource Update vm_uuid=%s disk_uuid=%s STATE     slot=%d type=%s size=%v\n",
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
	diagDiskType := utils.ValidateDiskType(data.Type.ValueString(), data.IsoUUID.ValueString())
	if diagDiskType != nil {
		resp.Diagnostics.AddError(diagDiskType.Summary(), diagDiskType.Detail())
		return
	}
	diagISOAttach, iso := utils.ValidateISOAttach(*r.client, data.IsoUUID.ValueString(), isAttachingISO)
	if diagISOAttach != nil {
		resp.Diagnostics.AddError(diagISOAttach.Summary(), diagISOAttach.Detail())
		return
	}
	diagFlashPriority := utils.ValidateDiskFlashPriority(data.FlashPriority.ValueInt64())
	if diagFlashPriority != nil {
		resp.Diagnostics.AddError(diagFlashPriority.Summary(), diagFlashPriority.Detail())
		return
	}

	isDetachingISO := oldHc3Disk["path"] != "" && data.IsoUUID.ValueString() == "" && data.Type.ValueString() == "IDE_CDROM"

	updatePayload := map[string]any{
		"virDomainUUID":         vmUUID,
		"type":                  data.Type.ValueString(),
		"capacity":              data.Size.ValueFloat64() * 1000 * 1000 * 1000, // GB to B
		"tieringPriorityFactor": utils.FROM_HUMAN_PRIORITY_FACTOR[data.FlashPriority.ValueInt64()],
	}
	if isAttachingISO {
		updatePayload["path"] = (*iso)["path"]
	} else if isDetachingISO {
		updatePayload["path"] = ""
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
	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreDiskResource: vm_uuid=%s, disk_uuid=%s, disk=%v", vmUUID, diskUUID, newHc3Disk))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreDiskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	tflog.Info(ctx, "TTRT HypercoreDiskResource DELETE")
	var data HypercoreDiskResourceModel

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

func (r *HypercoreDiskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	tflog.Info(ctx, "TTRT HypercoreDiskResource IMPORT_STATE")
	idParts := strings.Split(req.ID, ":")
	if len(idParts) != 3 {
		msg := fmt.Sprintf("Disk import composite ID format is 'vm_uuid:disk_type:disk_slot'. ID='%s' is invalid.", req.ID)
		resp.Diagnostics.AddError("Disk import requires a composite ID", msg)
		return
	}
	vmUUID := idParts[0]
	diskType := idParts[1]
	slot := utils.AnyToInteger64(idParts[2])
	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreDiskResource: vmUUID=%s, type=%s, slot=%d", vmUUID, diskType, slot))

	restClient := *r.client
	hc3VM := utils.GetOneVM(vmUUID, restClient)
	hc3Disks := utils.AnyToListOfMap(hc3VM["blockDevs"])
	tflog.Info(ctx, fmt.Sprintf("TTRT hc3Disks=%v\n", hc3Disks))

	var diskUUID string
	var size float64
	var flashPriority int64
	for _, disk := range hc3Disks {
		if utils.AnyToInteger64(disk["slot"]) == slot &&
			utils.AnyToString(disk["type"]) == diskType {
			diskUUID = utils.AnyToString(disk["uuid"])
			size = utils.AnyToFloat64(disk["capacity"]) / 1000 / 1000 / 1000 // hc3 has B, so convert to GB
			flashPriority = utils.AnyToInteger64(disk["tieringPriorityFactor"])
			tflog.Debug(ctx, fmt.Sprintf("TTRT HUMAN FLASH PRIORITY = %v", flashPriority))
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
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("flash_priority"), flashPriority)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("size"), size)...)
}
