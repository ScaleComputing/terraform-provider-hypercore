// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-data-source-read
// https://developer.hashicorp.com/terraform/plugin/framework/migrating

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-hypercore/internal/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &hypercoreVMsDataSource{}
	_ datasource.DataSourceWithConfigure = &hypercoreVMsDataSource{}
)

// NewHypercoreVMsDataSource is a helper function to simplify the provider implementation.
func NewHypercoreVMsDataSource() datasource.DataSource {
	return &hypercoreVMsDataSource{}
}

// hypercoreVMsDataSource is the data source implementation.
type hypercoreVMsDataSource struct {
	client *utils.RestClient
}

// coffeesDataSourceModel maps the data source schema data.
type hypercoreVMsDataSourceModel struct {
	FilterName types.String       `tfsdk:"name"`
	Vms        []hypercoreVMModel `tfsdk:"vms"`
}

// hypercoreVMModel maps VM schema data.
type hypercoreVMModel struct {
	UUID                 types.String         `tfsdk:"uuid"`
	Name                 types.String         `tfsdk:"name"`
	Description          types.String         `tfsdk:"description"`
	PowerState           types.String         `tfsdk:"power_state"`
	VCPU                 types.Int32          `tfsdk:"vcpu"`
	Memory               types.Int64          `tfsdk:"memory"`
	SnapshotScheduleUUID types.String         `tfsdk:"snapshot_schedule_uuid"`
	Tags                 []types.String       `tfsdk:"tags"`
	Disks                []HypercoreDiskModel `tfsdk:"disks"`
	// TODO nics
	AffinityStrategy AffinityStrategyModel `tfsdk:"affinity_strategy"`
}

type HypercoreDiskModel struct {
	UUID types.String  `tfsdk:"uuid"`
	Type types.String  `tfsdk:"type"`
	Slot types.Int64   `tfsdk:"slot"`
	Size types.Float64 `tfsdk:"size"`
}

// Metadata returns the data source type name.
func (d *hypercoreVMsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vms"
}

// Schema defines the schema for the data source.
func (d *hypercoreVMsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional: true,
			},
			"vms": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"vcpu": schema.Int32Attribute{
							MarkdownDescription: "Number of CPUs",
							Optional:            true,
						},
						"memory": schema.Int64Attribute{
							MarkdownDescription: "Memory (RAM) size in MiB",
							Optional:            true,
						},
						"snapshot_schedule_uuid": schema.StringAttribute{
							MarkdownDescription: "UUID of the applied snapshot schedule for creating automated snapshots",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							Computed: true,
						},
						"power_state": schema.StringAttribute{
							Computed: true,
						},
						"tags": schema.ListAttribute{
							ElementType: types.StringType,
							Optional:    true,
						},
						"affinity_strategy": schema.ObjectAttribute{
							MarkdownDescription: "VM node affinity.",
							Computed:            true,
							AttributeTypes: map[string]attr.Type{
								"strict_affinity":     types.BoolType,
								"preferred_node_uuid": types.StringType,
								"backup_node_uuid":    types.StringType,
							},
						},

						"disks": schema.ListNestedAttribute{
							MarkdownDescription: "List of disks",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"uuid": schema.StringAttribute{
										MarkdownDescription: "UUID",
										Computed:            true,
									},
									"type": schema.StringAttribute{
										MarkdownDescription: "type",
										Computed:            true,
									},
									"slot": schema.Int64Attribute{
										MarkdownDescription: "slot",
										Computed:            true,
									},
									"size": schema.Float64Attribute{
										MarkdownDescription: "size",
										Computed:            true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *hypercoreVMsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	// client, ok := req.ProviderData.(*hashicups.Client)
	restClient, ok := req.ProviderData.(*utils.RestClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = restClient
}

// Read refreshes the Terraform state with the latest data.
func (d *hypercoreVMsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	var conf hypercoreVMsDataSourceModel
	req.Config.Get(ctx, &conf)
	filter_name := conf.FilterName.ValueString()

	query := map[string]any{}
	if filter_name != "" {
		query = map[string]any{"name": filter_name}
	}
	hc3_vms := d.client.ListRecords(
		"/rest/v1/VirDomain",
		query,
		-1.0,
		false,
	)
	tflog.Debug(ctx, fmt.Sprintf("TTRT: filter_name=%s vm_count=%d\n", filter_name, len(hc3_vms)))
	if filter_name != "" {
		if len(hc3_vms) == 0 {
			resp.Diagnostics.AddError("VM not found", fmt.Sprintf("No VM with name %s found.", filter_name))
			return
		}
		if len(hc3_vms) > 1 {
			resp.Diagnostics.AddError("Multiple VMs found", fmt.Sprintf("Multiple VMs with name %s found.", filter_name))
			return
		}
	}

	var state hypercoreVMsDataSourceModel
	state.FilterName = types.StringValue(filter_name)
	for _, vm := range hc3_vms {
		// tags
		tags_all_str := utils.AnyToString(vm["tags"])
		tags_string := strings.Split(tags_all_str, ",")
		tags_String := make([]types.String, 0)
		for _, tag := range tags_string {
			tags_String = append(tags_String, types.StringValue(tag))
		}
		// disks
		blockDevs, ok := vm["blockDevs"].([]interface{})
		if !ok {
			panic(fmt.Sprintf("Unexpected blockDevs field: %v", vm["blockDevs"]))
		}
		disks := make([]HypercoreDiskModel, 0)
		for _, blockDev1 := range blockDevs {
			blockDev2, ok := blockDev1.(map[string]any)
			if !ok {
				panic(fmt.Sprintf("Unexpected blockDevs field: %v", vm["blockDevs"]))
			}
			uuid := utils.AnyToString(blockDev2["uuid"])
			disk_type := utils.AnyToString(blockDev2["type"])
			slot := utils.AnyToInteger64(blockDev2["slot"])
			size_B := float64(utils.AnyToInteger64(blockDev2["capacity"]))
			size_GB := types.Float64Value(size_B / 1000 / 1000 / 1000)
			disk := HypercoreDiskModel{
				UUID: types.StringValue(uuid),
				Type: types.StringValue(disk_type), // TODO convert "VIRTIO_DISK" to "virtio_disk" - or not?
				Slot: types.Int64Value(slot),
				Size: size_GB,
			}
			disks = append(disks, disk)
		}

		hc3affinityStrategy := utils.AnyToMap(vm["affinityStrategy"])
		var affinityStrategy AffinityStrategyModel
		affinityStrategy.StrictAffinity = types.BoolValue(utils.AnyToBool(hc3affinityStrategy["strictAffinity"]))
		affinityStrategy.PreferredNodeUUID = types.StringValue(utils.AnyToString(hc3affinityStrategy["preferredNodeUUID"]))
		affinityStrategy.BackupNodeUUID = types.StringValue(utils.AnyToString(hc3affinityStrategy["backupNodeUUID"]))

		// VM
		memory_B := utils.AnyToInteger64(vm["mem"])
		memory_MiB := memory_B / 1024 / 1024
		hypercoreVMState := hypercoreVMModel{
			UUID:                 types.StringValue(utils.AnyToString(vm["uuid"])),
			Name:                 types.StringValue(utils.AnyToString(vm["name"])),
			VCPU:                 types.Int32Value(int32(utils.AnyToInteger64(vm["numVCPU"]))),
			Memory:               types.Int64Value(memory_MiB),
			SnapshotScheduleUUID: types.StringValue(utils.AnyToString(vm["snapshotScheduleUUID"])),
			Description:          types.StringValue(utils.AnyToString(vm["description"])),
			PowerState:           types.StringValue(utils.AnyToString(vm["state"])), // TODO convert (stopped vs SHUTOFF)
			Tags:                 tags_String,
			AffinityStrategy:     affinityStrategy,
			Disks:                disks,
		}
		state.Vms = append(state.Vms, hypercoreVMState)
	}
	tflog.Debug(ctx, fmt.Sprintf("TTRT: filter_name=%s name=%s\n", filter_name, state.Vms[0].Name.String()))

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
