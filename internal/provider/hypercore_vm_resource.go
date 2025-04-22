// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-hypercore/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &HypercoreVMResource{}
var _ resource.ResourceWithImportState = &HypercoreVMResource{}

func NewHypercoreVMResource() resource.Resource {
	return &HypercoreVMResource{}
}

// HypercoreVMResource defines the resource implementation.
type HypercoreVMResource struct {
	client *utils.RestClient
}

// HypercoreVMResourceModel describes the resource data model.
type HypercoreVMResourceModel struct {
	Group                types.String          `tfsdk:"group"`
	Name                 types.String          `tfsdk:"name"`
	Description          types.String          `tfsdk:"description"`
	VCPU                 types.Int32           `tfsdk:"vcpu"`
	Memory               types.Int64           `tfsdk:"memory"`
	Import               *ImportModel          `tfsdk:"import"`
	SnapshotScheduleUUID types.String          `tfsdk:"snapshot_schedule_uuid"`
	Clone                *CloneModel           `tfsdk:"clone"`
	AffinityStrategy     AffinityStrategyModel `tfsdk:"affinity_strategy"`
	Id                   types.String          `tfsdk:"id"`
}

type ImportModel struct {
	HTTPUri  types.String `tfsdk:"http_uri"`
	Server   types.String `tfsdk:"server"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Path     types.String `tfsdk:"path"`
	FileName types.String `tfsdk:"file_name"`
}

type CloneModel struct {
	SourceVMUUID types.String `tfsdk:"source_vm_uuid"`
	UserData     types.String `tfsdk:"user_data"`
	MetaData     types.String `tfsdk:"meta_data"`
}

type AffinityStrategyModel struct {
	StrictAffinity    types.Bool   `tfsdk:"strict_affinity"`
	PreferredNodeUUID types.String `tfsdk:"preferred_node_uuid"`
	BackupNodeUUID    types.String `tfsdk:"backup_node_uuid"`
}

func (r *HypercoreVMResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (r *HypercoreVMResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: `HypercoreVM resource to create a VM from a template VM

A running VM might need to shutdown to apply required changes.
In this case first a nice ACPI shutdown is tried.
If the VM does not stop, a force shutdown is tried.
In both cases provider wait up to HC_VM_SHUTDOWN_TIMEOUT seconds for VM to shutdown.
HC_VM_SHUTDOWN_TIMEOUT dafault value is 300 seconds.
HC_VM_SHUTDOWN_TIMEOUT can be changed via environ.

The provider will currently try to shutdown VM only before VM delete.`,

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
			"snapshot_schedule_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the snapshot schedule to create automatic snapshots",
				Optional:            true,
			},
			"import": schema.SingleNestedAttribute{
				MarkdownDescription: "Options for importing a VM through a SMB server or some other HTTP location. <br>" +
					"Use server, username, password for SMB or http_uri for some other HTTP location. Parameters path and file_name are always **required**",
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"http_uri": schema.StringAttribute{
						Optional: true,
					},
					"server": schema.StringAttribute{
						Optional: true,
					},
					"username": schema.StringAttribute{
						Optional: true,
					},
					"password": schema.StringAttribute{
						Optional:  true,
						Sensitive: true,
					},
					"path": schema.StringAttribute{
						Required: true,
					},
					"file_name": schema.StringAttribute{
						Required: true,
					},
				},
			},
			"clone": schema.ObjectAttribute{
				MarkdownDescription: "" +
					"Clone options if the VM is being created as a clone. The `source_vm_uuid` is the UUID of the VM used for cloning, <br>" +
					"`user_data` and `meta_data` are used for the cloud init data.",
				Optional: true,
				AttributeTypes: map[string]attr.Type{
					"source_vm_uuid": types.StringType,
					"user_data":      types.StringType,
					"meta_data":      types.StringType,
				},
			},
			"affinity_strategy": schema.ObjectAttribute{
				MarkdownDescription: "VM node affinity.",
				Optional:            true,
				AttributeTypes: map[string]attr.Type{
					"strict_affinity":     types.BoolType,
					"preferred_node_uuid": types.StringType,
					"backup_node_uuid":    types.StringType,
				},
				Computed: true,
				Default: objectdefault.StaticValue(
					types.ObjectValueMust(
						map[string]attr.Type{
							"strict_affinity":     types.BoolType,
							"preferred_node_uuid": types.StringType,
							"backup_node_uuid":    types.StringType,
						},
						map[string]attr.Value{
							"strict_affinity":     types.BoolValue(false),
							"preferred_node_uuid": types.StringValue(""),
							"backup_node_uuid":    types.StringValue(""),
						},
					),
				),
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "HypercoreVM identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *HypercoreVMResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Info(ctx, "TTRT HypercoreVMResource CONFIGURE")
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

func getVMStruct(data *HypercoreVMResourceModel, vmDescription *string, vmTags *[]string) *utils.VM {
	// Gets VM structure from Utils.VM, sends parameters based on which VM create logic is being called
	sourceVMUUID, userData, metaData := "", "", ""
	if data.Clone != nil {
		sourceVMUUID = data.Clone.SourceVMUUID.ValueString()
		userData = data.Clone.UserData.ValueString()
		metaData = data.Clone.MetaData.ValueString()
	}
	vmStruct := utils.GetVMStruct(
		data.Name.ValueString(),
		sourceVMUUID,
		userData,
		metaData,
		vmDescription,
		vmTags,
		data.VCPU.ValueInt32Pointer(),
		data.Memory.ValueInt64Pointer(),
		data.SnapshotScheduleUUID.ValueStringPointer(),
		nil,
		data.AffinityStrategy.StrictAffinity.ValueBool(),
		data.AffinityStrategy.PreferredNodeUUID.ValueString(),
		data.AffinityStrategy.BackupNodeUUID.ValueString(),
	)
	return vmStruct
}

func validateParameters(data *HypercoreVMResourceModel) (*string, *[]string) {
	var tags *[]string
	var description *string

	if data.Group.ValueString() == "" {
		tags = nil
	} else {
		t := []string{data.Group.ValueString()}
		tags = &t
	}

	if data.Description.ValueString() == "" {
		description = nil
	} else {
		description = data.Description.ValueStringPointer()
	}
	return description, tags
}
func isHTTPImport(data *HypercoreVMResourceModel) bool {
	// Check if HTTP URI is being used for VM import
	httpUri := data.Import.HTTPUri.ValueString()
	return httpUri != ""
}
func isSMBImport(data *HypercoreVMResourceModel) bool {
	smbServer := data.Import.Server.ValueString()
	smbUsername := data.Import.Username.ValueString()
	smbPassword := data.Import.Password.ValueString()

	return smbServer != "" || smbUsername != "" || smbPassword != ""
}

func (r *HypercoreVMResource) handleCreateFromScratchLogic(data *HypercoreVMResourceModel, ctx context.Context, vmNew *utils.VM) {
	changed, msg := vmNew.FromScratch(*r.client, ctx)
	tflog.Info(ctx, fmt.Sprintf("Changed: %t, Message: %s\n", changed, msg))
	vmNew.SetVMParams(*r.client, ctx)
	data.Id = types.StringValue(vmNew.UUID)
}
func (r *HypercoreVMResource) handleCloneLogic(data *HypercoreVMResourceModel, ctx context.Context, vmNew *utils.VM) {
	changed, msg := vmNew.Clone(*r.client, ctx)
	tflog.Info(ctx, fmt.Sprintf("Changed: %t, Message: %s\n", changed, msg))
	// Clone will retain setting from original VM so we call SetVMParams to change those settings based on user input before we save state
	changed, vmWasRebooted, vmDiff := vmNew.SetVMParams(*r.client, ctx)
	data.Id = types.StringValue(vmNew.UUID)
	tflog.Info(ctx, fmt.Sprintf("Changed: %t, Was VM Rebooted: %t, Diff: %v", changed, vmWasRebooted, vmDiff))
}
func (r *HypercoreVMResource) handleImportFromSMBLogic(data *HypercoreVMResourceModel, ctx context.Context, resp *resource.CreateResponse, vmNew *utils.VM, path string, fileName string) {
	smbServer, smbUsername, smbPassword := data.Import.Server.ValueString(), data.Import.Username.ValueString(), data.Import.Password.ValueString()
	errorDiagnostic := utils.ValidateSMB(smbServer, smbUsername, smbPassword)
	if errorDiagnostic != nil {
		resp.Diagnostics.AddError(errorDiagnostic.Summary(), errorDiagnostic.Detail())
		return
	}
	smbSource := utils.BuildImportSource(smbUsername, smbPassword, smbServer, path, fileName, "", true)
	vmNew.Import(*r.client, smbSource, ctx)
	// Import will retain setting from original VM so we call SetVMParams to change those settings based on user input before we save state
	changed, vmWasRebooted, vmDiff := vmNew.SetVMParams(*r.client, ctx)
	data.Id = types.StringValue(vmNew.UUID)
	tflog.Info(ctx, fmt.Sprintf("Changed: %t, Was VM Rebooted: %t, Diff: %v", changed, vmWasRebooted, vmDiff))
}
func (r *HypercoreVMResource) handleImportFromURILogic(data *HypercoreVMResourceModel, ctx context.Context, resp *resource.CreateResponse, vmNew *utils.VM, path string, fileName string) {
	httpUri := data.Import.HTTPUri.ValueString()
	errorDiagnostic := utils.ValidateHTTP(httpUri, path)
	if errorDiagnostic != nil {
		resp.Diagnostics.AddError(errorDiagnostic.Summary(), errorDiagnostic.Detail())
		return
	}
	httpSource := utils.BuildImportSource("", "", "", path, fileName, httpUri, false)
	vmNew.Import(*r.client, httpSource, ctx)
	// Import will retain setting from original VM so we call SetVMParams to change those settings based on user input before we save state
	changed, vmWasRebooted, vmDiff := vmNew.SetVMParams(*r.client, ctx)
	data.Id = types.StringValue(vmNew.UUID)
	tflog.Info(ctx, fmt.Sprintf("Changed: %t, Was VM Rebooted: %t, Diff: %v", changed, vmWasRebooted, vmDiff))
}
func (r *HypercoreVMResource) doCreateLogic(data *HypercoreVMResourceModel, ctx context.Context, resp *resource.CreateResponse, description *string, tags *[]string) {
	vmNew := getVMStruct(data, description, tags)
	// Chose which VM create logic we're going with (clone, import, from scratch)
	if data.Clone != nil {
		r.handleCloneLogic(data, ctx, vmNew)
	} else if data.Import != nil {
		path := data.Import.Path.ValueString()
		fileName := data.Import.FileName.ValueString()
		if isHTTPImport(data) && !isSMBImport(data) {
			r.handleImportFromURILogic(data, ctx, resp, vmNew, path, fileName)
		} else if isSMBImport(data) && !isHTTPImport(data) {
			r.handleImportFromSMBLogic(data, ctx, resp, vmNew, path, fileName)
		}
	} else {
		r.handleCreateFromScratchLogic(data, ctx, vmNew)
	}
}

func (r *HypercoreVMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	tflog.Info(ctx, "TTRT HypercoreVMResource CREATE")
	var data HypercoreVMResourceModel

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

	// Validate parameters TODO: Add other inputs here from schema if validation is needed
	description, tags := validateParameters(&data)

	// Right now handles import or clone TODO: Add other VM create options here
	r.doCreateLogic(&data, ctx, resp, description, tags)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreVMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	tflog.Info(ctx, "TTRT HypercoreVMResource READ")
	var data HypercoreVMResourceModel
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
	tflog.Debug(ctx, fmt.Sprintf("TTRT HypercoreVMResource Read oldState vm_uuid=%s\n", vm_uuid))
	hc3_vm := utils.GetOneVM(vm_uuid, restClient)
	tflog.Debug(ctx, fmt.Sprintf("TTRT HypercoreVMResource Read vmhc3_vm=%s\n", hc3_vm))
	hc3_vm_name := utils.AnyToString(hc3_vm["name"])
	tflog.Debug(ctx, fmt.Sprintf("TTRT HypercoreVMResource Read vm_uuid=%s hc3_vm=(name=%s)\n", vm_uuid, hc3_vm_name))

	data.Name = types.StringValue(utils.AnyToString(hc3_vm["name"]))
	data.Description = types.StringValue(utils.AnyToString(hc3_vm["description"]))
	// data.Group TODO - replace "group" string with "tags" list of strings

	// NOTE: power state not needed here anymore because of the hypercore_vm_power_state resource
	// hc3_power_state := utils.AnyToString(hc3_vm["state"])
	// // line below look like correct thing to do. But "terraform plan -refresh-only"
	// // complains about change 'power_state = "stop" -> "stopped"
	// tf_power_state := types.StringValue(utils.FromHypercoreToTerraformPowerState[hc3_power_state])
	// // TEMP make "terraform plan -refresh-only" report "nothing changed"
	// hc3_stopped_states := []string{"SHUTOFF", "CRASHED"}
	// if slices.Contains(hc3_stopped_states, hc3_power_state) {
	// 	tf_power_state = types.StringValue("stop")
	// }
	// data.PowerState = tf_power_state

	// desiredDisposition TODO
	// uiState TODO
	data.VCPU = types.Int32Value(int32(utils.AnyToInteger64(hc3_vm["numVCPU"])))
	data.Memory = types.Int64Value(utils.AnyToInteger64(hc3_vm["mem"]) / 1024 / 1024)
	data.SnapshotScheduleUUID = types.StringValue(utils.AnyToString(hc3_vm["snapshotScheduleUUID"]))

	affinityStrategy := utils.AnyToMap(hc3_vm["affinityStrategy"])
	data.AffinityStrategy.StrictAffinity = types.BoolValue(utils.AnyToBool(affinityStrategy["strictAffinity"]))
	data.AffinityStrategy.PreferredNodeUUID = types.StringValue(utils.AnyToString(affinityStrategy["preferredNodeUUID"]))
	data.AffinityStrategy.BackupNodeUUID = types.StringValue(utils.AnyToString(affinityStrategy["backupNodeUUID"]))

	// ==============================================================================

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreVMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	tflog.Info(ctx, "TTRT HypercoreVMResource UPDATE")
	var data_state HypercoreVMResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data_state)...)
	var data HypercoreVMResourceModel

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

	// ======================================================================
	restClient := *r.client
	vm_uuid := data.Id.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("TTRT HypercoreVMResource Update vm_uuid=%s REQ   vcpu=%d description=%s", vm_uuid, data.VCPU.ValueInt32(), data.Description.String()))
	tflog.Debug(ctx, fmt.Sprintf("TTRT HypercoreVMResource Update vm_uuid=%s STATE vcpu=%d description=%s", vm_uuid, data_state.VCPU.ValueInt32(), data_state.Description.String()))

	updatePayload := map[string]any{}
	if data_state.Name != data.Name {
		updatePayload["name"] = data.Name.ValueString()
	}
	if data_state.Description != data.Description {
		updatePayload["description"] = data.Description.ValueString()
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
	if data_state.SnapshotScheduleUUID != data.SnapshotScheduleUUID {
		updatePayload["snapshotScheduleUUID"] = data.SnapshotScheduleUUID.ValueString()
	}

	affinityStrategy := map[string]any{}
	if data_state.AffinityStrategy.StrictAffinity != data.AffinityStrategy.StrictAffinity {
		affinityStrategy["strictAffinity"] = data.AffinityStrategy.StrictAffinity.ValueBool()
	}
	if data_state.AffinityStrategy.PreferredNodeUUID != data.AffinityStrategy.PreferredNodeUUID {
		affinityStrategy["preferredNodeUUID"] = data.AffinityStrategy.PreferredNodeUUID.ValueString()
	}
	if data_state.AffinityStrategy.BackupNodeUUID != data.AffinityStrategy.BackupNodeUUID {
		affinityStrategy["backupNodeUUID"] = data.AffinityStrategy.BackupNodeUUID.ValueString()
	}
	if len(affinityStrategy) > 0 {
		updatePayload["affinityStrategy"] = affinityStrategy
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

func (r *HypercoreVMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	tflog.Info(ctx, "TTRT HypercoreVMResource DELETE")
	var data HypercoreVMResourceModel

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
	err := ShutdownVM(ctx, vm_uuid, &restClient)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to shutdown VM, got error: %s", err))
		return
	}

	taskTag := restClient.DeleteRecord(
		fmt.Sprintf("/rest/v1/VirDomain/%s", vm_uuid),
		-1,
		ctx,
	)
	taskTag.WaitTask(restClient, ctx)
}

func (r *HypercoreVMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "TTRT HypercoreVMResource IMPORT_STATE")
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func ShutdownVM(ctx context.Context, vmUUID string, restClient *utils.RestClient) diag.Diagnostic {
	currentState, err := utils.GetVMPowerState(vmUUID, *restClient)
	if err != nil {
		return err
	}
	if currentState == "SHUTOFF" {
		return nil
	}
	// VM needs to be shutdown

	// VM shutdown might be already initiated, but not yet fully done.
	// Send ACPI shutdown if needed.
	desiredState, err := utils.GetVMDesiredState(vmUUID, *restClient)
	if err != nil {
		return err
	}
	if desiredState != "SHUTOFF" {
		err = utils.ModifyVMPowerState(*restClient, vmUUID, "SHUTDOWN", ctx)
		if err != nil {
			return err
		}
	}
	var maxWaitTime int = 300
	envMaxWaitTime := os.Getenv("HC_VM_SHUTDOWN_TIMEOUT")
	if envMaxWaitTime != "" {
		val, err := strconv.Atoi(envMaxWaitTime)
		if err != nil {
			err_msg := fmt.Sprintf("TTRT HypercoreVMResource Destroy, environ HC_VM_SHUTDOWN_TIMEOUT=%s is not number, err=%s", envMaxWaitTime, err)
			tflog.Error(ctx, err_msg)
			var diags diag.Diagnostics
			diags.AddError(
				"Invalid environ variable HC_VM_SHUTDOWN_TIMEOUT",
				err_msg,
			)
			return diags[0]
		}
		maxWaitTime = val
	}
	var waitSleep = 5
	var waitTime = 0
	for {
		// wait on VM to stop
		currentState, err := utils.GetVMPowerState(vmUUID, *restClient)
		if err != nil {
			return err
		}
		if currentState == "SHUTOFF" {
			return nil
		}
		if waitTime > maxWaitTime {
			break
		}
		time.Sleep(time.Duration(waitSleep) * time.Second)
		waitTime += waitSleep
	}

	// force shutdown is needed
	tflog.Warn(ctx, "TTRT HypercoreVMResource Destroy, VM is still running after nice ACPI shutdown, VM will be force shutoff.")
	err = utils.ModifyVMPowerState(*restClient, vmUUID, "STOP", ctx)
	if err != nil {
		return err
	}
	waitTime = 0
	for {
		// wait on VM to stop
		currentState, err := utils.GetVMPowerState(vmUUID, *restClient)
		if err != nil {
			return err
		}
		if currentState == "SHUTOFF" {
			return nil
		}
		if waitTime > maxWaitTime {
			break
		}
		time.Sleep(time.Duration(waitSleep) * time.Second)
		waitTime += waitSleep
	}

	tflog.Error(ctx, "TTRT HypercoreVMResource Destroy, VM is still running after force shutdown")
	var diags diag.Diagnostics
	diags.AddError(
		"Error Shutting down VM",
		"Unable to shutdown VM with ACPI shutdown or force shutdown.",
	)
	return diags[0]
}
