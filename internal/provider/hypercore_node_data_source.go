// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-data-source-read
// https://developer.hashicorp.com/terraform/plugin/framework/migrating

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-hypercore/internal/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &hypercoreNodeDataSource{}
	_ datasource.DataSourceWithConfigure = &hypercoreNodeDataSource{}
)

// NewHypercoreNodeDataSource is a helper function to simplify the provider implementation.
func NewHypercoreNodeDataSource() datasource.DataSource {
	return &hypercoreNodeDataSource{}
}

// hypercoreNodeDataSource is the data source implementation.
type hypercoreNodeDataSource struct {
	client *utils.RestClient
}

// coffeesDataSourceModel maps the data source schema data.
type hypercoreNodesDataSourceModel struct {
	FilterPeerID types.Int64          `tfsdk:"peer_id"`
	Nodes        []hypercoreNodeModel `tfsdk:"nodes"`
}

// hypercoreVMModel maps VM schema data.
type hypercoreNodeModel struct {
	UUID        types.String `tfsdk:"uuid"`
	BackplaneIP types.String `tfsdk:"backplane_ip"`
	LanIP       types.String `tfsdk:"lan_ip"`
	PeerID      types.Int64  `tfsdk:"peer_id"`
}

// Metadata returns the data source type name.
func (d *hypercoreNodeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node"
}

// Schema defines the schema for the data source.
func (d *hypercoreNodeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"peer_id": schema.Int64Attribute{
				Optional: true,
			},
			"nodes": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							Computed: true,
						},
						"backplane_ip": schema.StringAttribute{
							Computed: true,
						},
						"lan_ip": schema.StringAttribute{
							Computed: true,
						},
						"peer_id": schema.Int64Attribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *hypercoreNodeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

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
func (d *hypercoreNodeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var conf hypercoreNodesDataSourceModel
	req.Config.Get(ctx, &conf)
	// use float64, because this is the type of loaded json data
	filter_peer_id := float64(conf.FilterPeerID.ValueInt64())

	query := map[string]any{}
	// peerID=0 is reserved value, never returned by HC3
	if filter_peer_id != 0.0 {
		query = map[string]any{"peerID": filter_peer_id}
	}

	hc3_nodes := d.client.ListRecords(
		"/rest/v1/Node",
		query,
		-1.0,
		false,
	)
	tflog.Info(ctx, fmt.Sprintf("TTRT: filter_peer_id=%v node_count=%d\n", filter_peer_id, len(hc3_nodes)))

	var state hypercoreNodesDataSourceModel
	for _, node := range hc3_nodes {
		hypercoreNodeState := hypercoreNodeModel{
			UUID:        types.StringValue(utils.AnyToString(node["uuid"])),
			BackplaneIP: types.StringValue(utils.AnyToString(node["backplaneIP"])),
			LanIP:       types.StringValue(utils.AnyToString(node["lanIP"])),
			PeerID:      types.Int64Value(utils.AnyToInteger64(node["peerID"])),
		}
		state.Nodes = append(state.Nodes, hypercoreNodeState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
