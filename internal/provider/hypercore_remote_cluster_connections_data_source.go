// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-data-source-read
// https://developer.hashicorp.com/terraform/plugin/framework/migrating

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-hypercore/internal/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &hypercoreRemoteClusterConnectionsDataSource{}
	_ datasource.DataSourceWithConfigure = &hypercoreRemoteClusterConnectionsDataSource{}
)

// NewHypercoreRemoteClusterConnectionDataSource is a helper function to simplify the provider implementation.
func NewHypercoreRemoteClusterConnectionsDataSource() datasource.DataSource {
	return &hypercoreRemoteClusterConnectionsDataSource{}
}

// hypercoreRemoteClusterConnectionsDataSource is the data source implementation.
type hypercoreRemoteClusterConnectionsDataSource struct {
	client *utils.RestClient
}

// coffeesDataSourceModel maps the data source schema data.
type hypercoreRemoteClusterConnectionsDataSourceModel struct {
	FilterRemoteClusterName  types.String                            `tfsdk:"remote_cluster_name"`
	RemoteClusterConnections []hypercoreRemoteClusterConnectionModel `tfsdk:"remote_clusters"`
}

// hypercoreVMModel maps VM schema data.
type hypercoreRemoteClusterConnectionModel struct {
	UUID             types.String `tfsdk:"uuid"`
	ClusterName      types.String `tfsdk:"cluster_name"`
	ConnectionStatus types.String `tfsdk:"connection_status"`
	ReplicationOk    types.Bool   `tfsdk:"replication_ok"`
	RemoteNodeIPs    types.List   `tfsdk:"remote_node_ips"`
	RemoteNodeUUIDs  types.List   `tfsdk:"remote_node_uuids"`
}

// Metadata returns the data source type name.
func (d *hypercoreRemoteClusterConnectionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_remote_cluster_connections"
}

// Schema defines the schema for the data source.
func (d *hypercoreRemoteClusterConnectionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"remote_cluster_name": schema.StringAttribute{
				Optional: true,
			},
			"remote_clusters": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							Computed: true,
						},
						"cluster_name": schema.StringAttribute{
							Computed: true,
						},
						"connection_status": schema.StringAttribute{
							Computed: true,
						},
						"replication_ok": schema.BoolAttribute{
							Computed: true,
						},
						"remote_node_ips": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
						"remote_node_uuids": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *hypercoreRemoteClusterConnectionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *hypercoreRemoteClusterConnectionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var conf hypercoreRemoteClusterConnectionsDataSourceModel
	req.Config.Get(ctx, &conf)

	filterName := conf.FilterRemoteClusterName.ValueString()

	query := map[string]any{}
	if filterName != "" {
		query = map[string]any{
			"remoteClusterInfo": map[string]any{
				"clusterName": filterName,
			},
		}
	}

	hc3RemoteClusters := d.client.ListRecords(
		"/rest/v1/RemoteClusterConnection",
		query,
		-1.0,
		true,
	)

	tflog.Info(ctx, fmt.Sprintf("TTRT: filter_name=%v remote_cluster_count=%d\n", filterName, len(hc3RemoteClusters)))

	var state hypercoreRemoteClusterConnectionsDataSourceModel
	for _, remoteCluster := range hc3RemoteClusters {
		remoteClusterInfo := utils.AnyToMap(remoteCluster["remoteClusterInfo"])

		remoteNodeIPs := utils.AnyToListOfStrings(remoteCluster["remoteNodeIPs"])
		remoteNodeUUIDs := utils.AnyToListOfStrings(remoteCluster["remoteNodeUUIDs"])

		// Go list of string to Terraform list of string
		remoteNodeIPsValues := make([]attr.Value, len(remoteNodeIPs))
		for i, remoteIP := range remoteNodeIPs {
			remoteNodeIPsValues[i] = types.StringValue(remoteIP)
		}
		remoteNodeUUIDsValues := make([]attr.Value, len(remoteNodeUUIDs))
		for i, remoteUUID := range remoteNodeUUIDs {
			remoteNodeUUIDsValues[i] = types.StringValue(remoteUUID)
		}

		tfRemoteNodeIPs, _diag := types.ListValue(types.StringType, remoteNodeIPsValues)
		if _diag.HasError() {
			resp.Diagnostics.Append(_diag...)
			return
		}
		tfRemoteNodeUUIDs, _diag := types.ListValue(types.StringType, remoteNodeUUIDsValues)
		if _diag.HasError() {
			resp.Diagnostics.Append(_diag...)
			return
		}

		// Save into state
		hypercoreRemoteClusterConnectionState := hypercoreRemoteClusterConnectionModel{
			UUID:             types.StringValue(utils.AnyToString(remoteCluster["uuid"])),
			ClusterName:      types.StringValue(utils.AnyToString(remoteClusterInfo["clusterName"])),
			ConnectionStatus: types.StringValue(utils.AnyToString(remoteCluster["connectionStatus"])),
			ReplicationOk:    types.BoolValue(utils.AnyToBool(remoteCluster["replicationOK"])),
			RemoteNodeIPs:    tfRemoteNodeIPs,
			RemoteNodeUUIDs:  tfRemoteNodeUUIDs,
		}
		state.RemoteClusterConnections = append(state.RemoteClusterConnections, hypercoreRemoteClusterConnectionState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
