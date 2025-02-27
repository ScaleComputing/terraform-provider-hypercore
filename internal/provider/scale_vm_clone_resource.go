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
	Nics        types.List   `tfsdk:"nics"`
	PowerState  types.String `tfsdk:"power_state"`
	Clone       CloneModel   `tfsdk:"clone"`
	Id          types.String `tfsdk:"id"`
	Disks       []DiskModel  `tfsdk:"disks"`
}

type CloneModel struct {
	SourceVMUUID types.String `tfsdk:"source_vm_uuid"`
	UserData     types.String `tfsdk:"user_data"`
	MetaData     types.String `tfsdk:"meta_data"`
	// Disks        types.List   `tfsdk:"disks"`
	Disk0Label types.String `tfsdk:"disk_0_label"`
	Disk0Type  types.String `tfsdk:"disk_0_type"`
	Disk0Slot  types.Int64  `tfsdk:"disk_0_slot"`
	Disk1Label types.String `tfsdk:"disk_1_label"`
	Disk1Type  types.String `tfsdk:"disk_1_type"`
	Disk1Slot  types.Int64  `tfsdk:"disk_1_slot"`
}

type DiskModel struct {
	Label types.String  `tfsdk:"label"`
	UUID  types.String  `tfsdk:"uuid"`
	Type  types.String  `tfsdk:"type"`
	Slot  types.Int64   `tfsdk:"slot"`
	Size  types.Float64 `tfsdk:"size"`
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
			"disks": schema.ListNestedAttribute{
				MarkdownDescription: "" +
					"A list of disks' configs for the cloned VM. If a disk with the same `type` and `slot` <br>" +
					"already exists, that disk will be replaced with the new one if the `size` is larger (existing <br>" +
					"disks can only be expanded and not shrunk). In the opposite case, a new disk will created <br>" +
					"and added to the cloned VM.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"label": schema.StringAttribute{
							MarkdownDescription: "Fictitious disk label. NB - there is no disk label attribute on the HypeCore.",
							// Computed:            true,
							Required: true,
						},
						"uuid": schema.StringAttribute{
							MarkdownDescription: "Disk's `UUID`, which is known after the disk has already been created.",
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Disk type. Can be: `IDE_DISK`, `SCSI_DISK`, `VIRTIO_DISK`, `IDE_FLOPPY`, `NVRAM`, `VTPM`",
							Required:            true,
						},
						"slot": schema.Int64Attribute{
							MarkdownDescription: "Disk slot number.",
							Required:            true,
						},
						"size": schema.Float64Attribute{
							MarkdownDescription: "Disk size in `GB`.",
							Required:            true,
						},
					},
				},
				Optional: true,
				//Computed: true, // TODO - just to make it "known after apply". Is "Computed=True" wrong choice?
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

					"disk_0_label": types.StringType, // TODO Optional=True, use "" instead
					"disk_0_type":  types.StringType,
					"disk_0_slot":  types.Int64Type,
					"disk_1_label": types.StringType,
					"disk_1_type":  types.StringType,
					"disk_1_slot":  types.Int64Type,
					// "disks":          types.ListType{},
					//====================================================================
					// TODO https://terrateam.io/blog/terraform-types/
					// "disks": schema.SetNestedAttribute{
					// 	NestedObject: schema.NestedAttributeObject{
					// 		Attributes: map[string]schema.Attribute{
					// 			"computed_attr": schema.StringAttribute{
					// 				Computed: true,
					// 				PlanModifiers: []planmodifier.String{
					// 					// Preserve this computed value during updates.
					// 					stringplanmodifier.MatchElementStateForUnknown(
					// 						// Identify matching prior state value based on configurable_attr
					// 						path.MatchRelative().AtParent().AtName("configurable_attr"),
					// 						// ... potentially others ...
					// 					),
					// 				},
					// 			},
					// 			"configurable_attr": schema.StringAttribute{
					// 				Required: true,
					// 			},
					// 		},
					// 	},
					// 	Optional: true,
					// },
					//====================================================================

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

	// shrani v vmDisks se uuid od diskov ravnokar-klonirane VM - isci po (type,slot)
	// shrani v diske nove VM se disk_name. disk_name gre tudi v tfstate
	// tfDisks := make([]DiskModel, 0, len(data.Disks.Elements()))
	tflog.Info(ctx, fmt.Sprintf("TTRT Create: data.Disks=%v\n", data.Disks))
	tflog.Info(ctx, fmt.Sprintf("TTRT Create: data.Clone.Disk0Label=%v data.Clone.Disk1Label=%v\n", data.Clone.Disk0Label, data.Clone.Disk1Label))
	tflog.Info(ctx, fmt.Sprintf("TTRT Create: data.Clone.Disk0Label=%v data.Clone.Disk1Label=%v\n", data.Clone.Disk0Label.ValueString(), data.Clone.Disk1Label.ValueString()))
	var tfCloneDisks []DiskModel
	if "" != data.Clone.Disk0Label.ValueString() {
		tfCloneDisk := DiskModel{
			Label: data.Clone.Disk0Label,
			// UUID: "",
			Type: data.Clone.Disk0Type,
			Slot: data.Clone.Disk0Slot,
			// Size: 0.0,
		}
		tfCloneDisks = append(tfCloneDisks, tfCloneDisk)
	}
	if "" != data.Clone.Disk1Label.ValueString() {
		tfCloneDisk := DiskModel{
			Label: data.Clone.Disk1Label,
			// UUID: "",
			Type: data.Clone.Disk1Type,
			Slot: data.Clone.Disk1Slot,
			// Size: 0.0,
		}
		tfCloneDisks = append(tfCloneDisks, tfCloneDisk)
	}
	tflog.Info(ctx, fmt.Sprintf("TTRT tfCloneDisks=%v\n", tfCloneDisks))

	vm_uuid := vmClone.UUID
	restClient := *r.client
	hc3VM := utils.GetOne(vm_uuid, restClient)
	hc3Disks := utils.AnyToListOfMap(hc3VM["blockDevs"])
	tflog.Info(ctx, fmt.Sprintf("TTRT hc3Disks=%v\n", hc3Disks))

	var vmDisks []utils.VMDisk
	for _, hc3Disk := range hc3Disks {
		// store disk from newly created VM
		size_B := utils.AnyToFloat64(hc3Disk["capacity"])
		size_GB := size_B / 1000 / 1000 / 1000
		vmDisk, err := utils.NewVMDisk(
			"",
			utils.AnyToInteger64(hc3Disk["slot"]),
			utils.AnyToString(hc3Disk["type"]),
			&size_GB,
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid configuration for 'disks'",
				err.Error(),
			)
			return
		}
		vmDisk.UUID = utils.AnyToString(hc3Disk["uuid"])
		tflog.Info(ctx, fmt.Sprintf("TTRT vmDisk uuid=%v type=%v slot=%v", vmDisk.UUID, vmDisk.Type, vmDisk.Slot))

		// Find corresponding disk from source VM, to add label to new VM disk
		// Some disks in source VM might not have a label.
		// Some disks in destination VM might not have a label.
		for _, tfCloneDisk := range tfCloneDisks {
			tflog.Info(ctx, fmt.Sprintf("TTRT         tfCloneDisk label=%v type=%v slot=%v", tfCloneDisk.Label.ValueString(), tfCloneDisk.Type.ValueString(), tfCloneDisk.Slot))
			if tfCloneDisk.Type.ValueString() == vmDisk.Type &&
				tfCloneDisk.Slot.ValueInt64() == vmDisk.Slot {
				vmDisk.Label = tfCloneDisk.Label.String()
				tflog.Info(ctx, fmt.Sprintf("TTRT Disk match: type=%s, slot=%d, label=%s", vmDisk.Type, vmDisk.Slot, vmDisk.Label))
			}
		}

		vmDisks = append(vmDisks, *vmDisk)
	}

	// TODO
	// ustvari manjkajoce diske
	// shrani v vmDisks se uuid, disk_name od ravnokar ustvarjenih diskov
	// Update each disk - size, slot, type

	// data.Disks = vmDisks
	// Update the Disks state in tfstate -- to compute the disks' UUIDs

	// ======================================================================================================
	// Disks modifications
	if true /* && !data.Disks.IsNull() */ {
		tfDisks := data.Disks

		// Fetch the new disks to create or update
		var vmDisks []utils.VMDisk
		for _, disk := range tfDisks {
			var err error
			var vmDisk *utils.VMDisk

			// If a disk already exists
			if disk.UUID.ValueString() != "" {
				tflog.Debug(ctx, "Disk UUID is KNOWN, updating existing one!\n")
				vmDisk, err = utils.UpdateVMDisk(
					disk.UUID.ValueString(),
					disk.Label.ValueString(),
					disk.Slot.ValueInt64(),
					disk.Type.ValueString(),
					disk.Size.ValueFloat64Pointer(),
				)
			} else { // If a disk needs to be created
				tflog.Debug(ctx, "Disk UUID is NOT KNOWN, creating new disk!\n")
				vmDisk, err = utils.NewVMDisk(
					disk.Label.ValueString(),
					disk.Slot.ValueInt64(),
					disk.Type.ValueString(),
					disk.Size.ValueFloat64Pointer(),
				)
			}

			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid configuration for 'disks'",
					err.Error(),
				)
				return
			}
			vmDisks = append(vmDisks, *vmDisk)
		}

		// shrani v vmDisks se uuid od diskov ravnokar-klonirane VM - isci po (type,slot)
		// shrani v diske nove VM se disk_name. disk_name gre tudi v tfstate
		// ustvari manjkajoce diske
		// shrani v vmDisks se uuid, disk_name od ravnokar ustvarjenih diskov
		// Update each disk - size, slot, type

		tflog.Debug(ctx, "Modifying disks\n")
		tfDisks = []DiskModel{}
		for i, vmDisk := range vmDisks {
			changed, vmWasRebooted, _, err := vmDisk.CreateOrUpdate(vmClone, *r.client, ctx)
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid configuration for 'disks'",
					err.Error(),
				)
				return
			}

			tflog.Info(ctx, fmt.Sprintf("Disk %d - Changed: %t, Was VM Rebooted: %t", i, changed, vmWasRebooted))

			tfDisks = append(tfDisks, DiskModel{
				Label: types.StringValue(vmDisk.Label),
				UUID:  types.StringValue(vmDisk.UUID),
				Slot:  types.Int64Value(vmDisk.Slot),
				Type:  types.StringValue(vmDisk.Type),
				Size:  types.Float64Value(*vmDisk.Size / 1000 / 1000 / 1000),
			})
		}

		// Update the Disks state in tfstate -- to compute the disks' UUIDs
		for _, tfDisk := range tfDisks {
			// find data.Disk
			disk_ii := -1
			for ii, data_disk := range data.Disks {
				if data_disk.Type == tfDisk.Type &&
					data_disk.Slot == tfDisk.Slot {
					tflog.Info(ctx, fmt.Sprintf("TTRT Disk match: type=%s, slot=%d, label=%s, uuid=%v uuid=%v", data_disk.Type, data_disk.Slot, data_disk.Label, data_disk.UUID, tfDisk.UUID))
					disk_ii = ii
					break
				}
			}
			data.Disks[disk_ii] = tfDisk
		}
	}
	// ======================================================================================================

	// [ ] TODO: 2. set the NICs of the new VM

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
	hc3_vm := utils.GetOne(vm_uuid, restClient)
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

	// Disks Read
	hc3_disks := utils.AnyToListOfMap(hc3_vm["blockDevs"])
	tflog.Debug(ctx, fmt.Sprintf("TTRT ScaleVMCloneResource Read vm_uuid=%s hc3_disks=(disks=%v)\n", vm_uuid, hc3_disks))

	diskValues := make([]attr.Value, 0, len(hc3_disks))

	// Map previous state disks by slot number
	oldDiskMap := make(map[int64]string) // TODO - we need to map by slot and type
	for _, oldDisk := range data.Disks {
		oldSlot := oldDisk.Slot.ValueInt64()
		oldUUID := oldDisk.UUID.ValueString()
		oldDiskMap[oldSlot] = oldUUID
	}

	for ii, disk := range hc3_disks {
		old_state_label := ""

		slot := utils.AnyToInteger64(disk["slot"])
		prevUUID, hasOldUUID := oldDiskMap[slot]
		newUUID := utils.AnyToString(disk["uuid"])
		finalUUID := newUUID
		if hasOldUUID && prevUUID != "" {
			finalUUID = prevUUID // ? ampak mi UUID-ja ne spreminjamo
		}

		objVal, objDiag := types.ObjectValue(map[string]attr.Type{
			"label": types.StringType,
			"uuid":  types.StringType,
			"slot":  types.Int64Type,
			"type":  types.StringType,
			"size":  types.Float64Type,
		},
			map[string]attr.Value{
				"label": types.StringValue(old_state_label),
				"uuid":  types.StringValue(finalUUID),
				"slot":  types.Int64Value(utils.AnyToInteger64(disk["slot"])),
				"type":  types.StringValue(utils.AnyToString(disk["type"])),
				"size":  types.Float64Value(utils.AnyToFloat64(disk["capacity"]) / 1000 / 1000 / 1000), // convert to GB
			},
		)
		if objDiag.HasError() {
			resp.Diagnostics.Append(objDiag...)
			return
		}
		diskValues = append(diskValues, objVal)

		size_GB := utils.AnyToFloat64(disk["capacity"]) / 1000 / 1000 / 1000 // convert to GB
		data.Disks[ii].UUID = types.StringValue(utils.AnyToString(disk["uuid"]))
		data.Disks[ii].Type = types.StringValue(utils.AnyToString(disk["type"]))
		data.Disks[ii].Slot = types.Int64Value(utils.AnyToInteger64(disk["slot"]))
		data.Disks[ii].Size = types.Float64Value(size_GB)
	}

	// data.nics TODO
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
