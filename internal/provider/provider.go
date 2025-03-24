// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-hypercore/internal/utils"
)

// Ensure HypercoreProvider satisfies various provider interfaces.
var _ provider.Provider = &HypercoreProvider{}
var _ provider.ProviderWithFunctions = &HypercoreProvider{}

// HypercoreProvider defines the provider implementation.
type HypercoreProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// HypercoreProviderModel describes the provider data model.
type HypercoreProviderModel struct {
	Host       types.String  `tfsdk:"host"`
	Username   types.String  `tfsdk:"username"`
	Password   types.String  `tfsdk:"password"`
	AuthMethod types.String  `tfsdk:"auth_method"`
	Timeout    types.Float64 `tfsdk:"timeout"`
}

func (p *HypercoreProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "hypercore"
	resp.Version = p.version
}

func (p *HypercoreProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "Hypercore Computing host URI; can also be set with `HC_HOST` environment variable.",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Hypercore Computing username; can also be set with `HC_USERNAME` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Hypercore Computing password; can also be set with `HC_PASSWORD` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"auth_method": schema.StringAttribute{
				MarkdownDescription: "Hypercore Computing authentication method; can also be set with `HC_AUTH_METHOD` environment variable. It can be set to `oidc` or `local` (default).",
				Optional:            true,
			},
			"timeout": schema.Float64Attribute{
				MarkdownDescription: "Hypercore Computing request timeout; can also be set with `HC_TIMEOUT` environment variable. Default is set to `60.0` seconds.",
				Optional:            true,
			},
		},
	}
}

func (p *HypercoreProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	scHost := os.Getenv("HC_HOST")
	scUsername := os.Getenv("HC_USERNAME")
	scPassword := os.Getenv("HC_PASSWORD")
	scAuthMethod := os.Getenv("HC_AUTH_METHOD")

	var scTimeoutF float64
	scTimeout := os.Getenv("HC_TIMEOUT")

	var data HypercoreProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if data.Host.ValueString() != "" {
		scHost = data.Host.ValueString()
	}

	if data.Username.ValueString() != "" {
		scUsername = data.Username.ValueString()
	}

	if data.Password.ValueString() != "" {
		scPassword = data.Password.ValueString()
	}

	if data.AuthMethod.ValueString() != "" {
		scAuthMethod = data.AuthMethod.ValueString()
	}

	if !data.Timeout.IsNull() && !data.Timeout.IsUnknown() {
		scTimeout = fmt.Sprint(data.Timeout.ValueFloat64())
	}

	if scHost == "" {
		resp.Diagnostics.AddError(
			"Missing Host URI Configuration",
			"While configuring the provider, the host URI was not found in "+
				"the HC_HOST environment variable or provider "+
				"configuration block host attribute.",
		)
	}

	if scUsername == "" {
		resp.Diagnostics.AddError(
			"Missing Username Configuration",
			"While configuring the provider, the Username was not found in "+
				"the HC_USERNAME environment variable or provider "+
				"configuration block username attribute.",
		)
	}

	if scPassword == "" {
		resp.Diagnostics.AddError(
			"Missing Password Configuration",
			"While configuring the provider, the Password was not found in "+
				"the HC_PASSWORD environment variable or provider "+
				"configuration block password attribute.",
		)
	}

	if scAuthMethod == "" {
		scAuthMethod = "local"
	}

	if scTimeout == "" {
		scTimeoutF = 60.0
		data.Timeout = types.Float64PointerValue(&scTimeoutF)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Hypercore client configuration for data sources and resources
	restClient, _ := utils.NewRestClient(
		scHost,
		scUsername,
		scPassword,
		scAuthMethod,
		scTimeoutF,
	)
	restClient.Login()
	tflog.Debug(ctx, fmt.Sprintf("Logged in with session ID: %s\n", restClient.AuthHeader["Cookie"]))

	// client := restClient
	resp.DataSourceData = restClient
	resp.ResourceData = restClient
}

func (p *HypercoreProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewHypercoreVMResource,
		NewHypercoreNicResource,
		NewHypercoreDiskResource,
		NewHypercoreVirtualDiskResource,
		NewHypercoreISOResource,
		NewHypercoreVMPowerStateResource,
		NewHypercoreVMBootOrderResource,
		NewHypercoreVMSnapshotResource,
	}
}

func (p *HypercoreProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewHypercoreVMDataSource,
		NewHypercoreNodeDataSource,
	}
}

func (p *HypercoreProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &HypercoreProvider{
			version: version,
		}
	}
}
