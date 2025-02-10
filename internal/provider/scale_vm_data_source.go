// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-data-source-read
// https://developer.hashicorp.com/terraform/plugin/framework/migrating

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-scale/internal/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &scaleVMDataSource{}
	_ datasource.DataSourceWithConfigure = &scaleVMDataSource{}
)

// NewScaleVMDataSource is a helper function to simplify the provider implementation.
func NewScaleVMDataSource() datasource.DataSource {
	return &scaleVMDataSource{}
}

// scaleVMDataSource is the data source implementation.
type scaleVMDataSource struct {
	client *utils.RestClient
}

// coffeesDataSourceModel maps the data source schema data.
type scaleVMsDataSourceModel struct {
	FilterName types.String   `tfsdk:"name"`
	Vms        []scaleVMModel `tfsdk:"vms"`
}

// scaleVMModel maps VM schema data.
type scaleVMModel struct {
	UUID        types.String     `tfsdk:"uuid"`
	Name        types.String     `tfsdk:"name"`
	Description types.String     `tfsdk:"description"`
	PowerState  types.String     `tfsdk:"power_state"`
	VCPU        types.Int32      `tfsdk:"vcpu"`
	Memory      types.Int64      `tfsdk:"memory"`
	Tags        []types.String   `tfsdk:"tags"`
	Disks       []ScaleDiskModel `tfsdk:"disks"`
	// TODO nics
}

type ScaleDiskModel struct {
	UUID types.String  `tfsdk:"uuid"`
	Type types.String  `tfsdk:"type"`
	Slot types.Int64   `tfsdk:"slot"`
	Size types.Float64 `tfsdk:"size"`
}

// Metadata returns the data source type name.
func (d *scaleVMDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

// Schema defines the schema for the data source.
func (d *scaleVMDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
func (d *scaleVMDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *scaleVMDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var conf scaleVMsDataSourceModel
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
	)
	tflog.Info(ctx, fmt.Sprintf("TTRT: filter_name=%s vm_count=%d\n", filter_name, len(hc3_vms)))

	var state scaleVMsDataSourceModel
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
		disks := make([]ScaleDiskModel, 0)
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
			disk := ScaleDiskModel{
				UUID: types.StringValue(uuid),
				Type: types.StringValue(disk_type), // TODO convert "VIRTIO_DISK" to "virtio_disk" - or not?
				Slot: types.Int64Value(slot),
				Size: size_GB,
			}
			disks = append(disks, disk)
		}
		// VM
		memory_B := utils.AnyToInteger64(vm["mem"])
		memory_MiB := memory_B / 1024 / 1024
		scaleVMState := scaleVMModel{
			UUID:        types.StringValue(utils.AnyToString(vm["uuid"])),
			Name:        types.StringValue(utils.AnyToString(vm["name"])),
			VCPU:        types.Int32Value(int32(utils.AnyToInteger64(vm["numVCPU"]))),
			Memory:      types.Int64Value(memory_MiB),
			Description: types.StringValue(utils.AnyToString(vm["description"])),
			PowerState:  types.StringValue(utils.AnyToString(vm["state"])), // TODO convert (stopped vs SHUTOFF)
			Tags:        tags_String,
			Disks:       disks,
		}
		state.Vms = append(state.Vms, scaleVMState)
	}
	tflog.Info(ctx, fmt.Sprintf("TTRT: filter_name=%s name=%s\n", filter_name, state.Vms[0].Name.String()))

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
