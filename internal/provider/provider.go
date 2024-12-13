// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ScaleProvider satisfies various provider interfaces.
var _ provider.Provider = &ScaleProvider{}
var _ provider.ProviderWithFunctions = &ScaleProvider{}

// ScaleProvider defines the provider implementation.
type ScaleProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ScaleProviderModel describes the provider data model.
type ScaleProviderModel struct {
	Host     types.String `tfsdk:"host"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func (p *ScaleProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "scale"
	resp.Version = p.version
}

func (p *ScaleProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
        MarkdownDescription: "Scale Computing host URI; can also be set with SC_HOST environment variable.",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Scale Computing username; can also be set with SC_USERNAME environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Scale Computing password; can also be set with SC_PASSWORD environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *ScaleProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	scHost := os.Getenv("SC_HOST")
	scUsername := os.Getenv("SC_USERNAME")
	scPassword := os.Getenv("SC_PASSWORD")

	var data ScaleProviderModel

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

	if scHost == "" {
		resp.Diagnostics.AddError(
			"Missing Host URI Configuration",
			"While configuring the provider, the host URI was not found in "+
				"the SC_HOST environment variable or provider "+
				"configuration block host attribute.",
		)
	}

	if scUsername == "" {
		resp.Diagnostics.AddError(
			"Missing Username Configuration",
			"While configuring the provider, the Username was not found in "+
				"the SC_USERNAME environment variable or provider "+
				"configuration block username attribute.",
		)
	}

	if scPassword == "" {
		resp.Diagnostics.AddError(
			"Missing Password Configuration",
			"While configuring the provider, the Password was not found in "+
				"the SC_PASSWORD environment variable or provider "+
				"configuration block password attribute.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	// if data.Endpoint.IsNull() { /* ... */ }

	// Scale client configuration for data sources and resources
	client := http.DefaultClient
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ScaleProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewScaleVMResource,
	}
}

func (p *ScaleProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *ScaleProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ScaleProvider{
			version: version,
		}
	}
}
