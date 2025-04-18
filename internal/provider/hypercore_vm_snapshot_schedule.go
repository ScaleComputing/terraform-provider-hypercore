// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-hypercore/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &HypercoreVMSnapshotScheduleResource{}
var _ resource.ResourceWithImportState = &HypercoreVMSnapshotScheduleResource{}

func NewHypercoreVMSnapshotScheduleResource() resource.Resource {
	return &HypercoreVMSnapshotScheduleResource{}
}

// HypercoreVMSnapshotScheduleResource defines the resource implementation.
type HypercoreVMSnapshotScheduleResource struct {
	client *utils.RestClient
}

// HypercoreVMSnapshotScheduleResourceModel describes the resource data model.
type HypercoreVMSnapshotScheduleResourceModel struct {
	Id    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Rules types.List   `tfsdk:"rules"`
}

type RulesModel struct {
	Name                   types.String `tfsdk:"name"`
	StartTimestamp         types.String `tfsdk:"start_timestamp"`
	Frequency              types.String `tfsdk:"frequency"`
	LocalRetentionSeconds  types.Int64  `tfsdk:"local_retention_seconds"`
	RemoteRetentionSeconds types.Int64  `tfsdk:"remote_retention_seconds"`
}

var rulesModelAttrType = map[string]attr.Type{
	"name":                     types.StringType,
	"start_timestamp":          types.StringType,
	"frequency":                types.StringType,
	"local_retention_seconds":  types.Int64Type,
	"remote_retention_seconds": types.Int64Type,
}

func GetRulesAttrValues(rules []RulesModel) ([]attr.Value, diag.Diagnostics) {
	var ruleValues []attr.Value
	for _, rule := range rules {
		ruleMap := map[string]attr.Value{
			"name":                     rule.Name,
			"start_timestamp":          rule.StartTimestamp,
			"frequency":                rule.Frequency,
			"local_retention_seconds":  rule.LocalRetentionSeconds,
			"remote_retention_seconds": rule.RemoteRetentionSeconds,
		}
		obj, diags := types.ObjectValue(rulesModelAttrType, ruleMap)
		if diags.HasError() {
			return nil, diags
		}
		ruleValues = append(ruleValues, obj)
	}
	return ruleValues, nil
}

func (r *HypercoreVMSnapshotScheduleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_snapshot_schedule"
}

func (r *HypercoreVMSnapshotScheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Hypercore VM snapshot schedule resource to manage VM snapshots",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Snapshot schedule identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Snapshot schedule name.",
				Required:            true,
			},
			"rules": schema.ListNestedAttribute{
				MarkdownDescription: "Scheduled snapshot rules.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Rule name",
							Required:            true,
						},
						"start_timestamp": schema.StringAttribute{
							MarkdownDescription: "Local timezone timestamp (2010-01-01 00:00:00) of when a snapshot is to be taken",
							Required:            true,
						},
						"frequency": schema.StringAttribute{
							MarkdownDescription: "Frequency based on RFC-2445 (FREQ=MINUTELY;INTERVAL=5)",
							Required:            true,
						},
						"local_retention_seconds": schema.Int64Attribute{
							MarkdownDescription: "Number of seconds before snapshots are removed",
							Required:            true,
						},
						"remote_retention_seconds": schema.Int64Attribute{
							MarkdownDescription: "Number of seconds before snapshots are removed. If not set, it'll be the same as `local_retention_seconds`",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(0),
						},
					},
				},
			},
		},
	}
}

func (r *HypercoreVMSnapshotScheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Info(ctx, "TTRT HypercoreVMSnapshotScheduleResource CONFIGURE")
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

func (r *HypercoreVMSnapshotScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	tflog.Info(ctx, "TTRT HypercoreVMSnapshotScheduleResource CREATE")
	var data HypercoreVMSnapshotScheduleResourceModel

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

	restClient := *r.client
	scheduleName := data.Name.ValueString()

	var scheduleRules []RulesModel
	if data.Rules.IsUnknown() {
		scheduleRules = []RulesModel{}
	} else {
		if len(data.Rules.Elements()) != 0 {
			scheduleRules = make([]RulesModel, len(data.Rules.Elements()))
			diags := data.Rules.ElementsAs(ctx, &scheduleRules, false)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		}
	}

	var payloadScheduleRules []map[string]any
	if len(scheduleRules) != 0 {
		for _, scheduleRule := range scheduleRules {
			payloadScheduleRules = append(payloadScheduleRules, map[string]any{
				"dtstart":                        scheduleRule.StartTimestamp.ValueString(),
				"rrule":                          scheduleRule.Frequency.ValueString(),
				"name":                           scheduleRule.Name.ValueString(),
				"localRetentionDurationSeconds":  scheduleRule.LocalRetentionSeconds.ValueInt64(),
				"remoteRetentionDurationSeconds": scheduleRule.RemoteRetentionSeconds.ValueInt64(),
			})
		}
	} else {
		payloadScheduleRules = []map[string]any{} // empty list
	}

	tflog.Info(ctx, fmt.Sprintf("TTRT Create: scheduleRules = %v", scheduleRules))

	// Create schedule
	payload := map[string]any{
		"name":   scheduleName,
		"rrules": payloadScheduleRules,
	}
	scheduleUUID, schedule, _diag := utils.CreateVMSnapshotSchedule(restClient, payload, ctx)
	if _diag != nil {
		resp.Diagnostics.AddWarning(_diag.Summary(), _diag.Detail())
	}

	tflog.Info(ctx, fmt.Sprintf("TTRT Created: schedule_uuid=%s, name=%s, rules=%v, schedule=%s", scheduleUUID, scheduleName, scheduleRules, schedule))

	// TODO: Check if HC3 matches TF

	// Retrieve rules data
	var ruleValues []attr.Value
	var _diags diag.Diagnostics
	if schedule["rrules"] != nil {
		hc3Rules := utils.AnyToListOfMap(schedule["rrules"])
		for i := range hc3Rules {
			if scheduleRules[i].RemoteRetentionSeconds.IsUnknown() {
				scheduleRules[i].RemoteRetentionSeconds = types.Int64Value(0)
			}
		}

		ruleValues, _diags = GetRulesAttrValues(scheduleRules)
		if _diags != nil {
			resp.Diagnostics.Append(_diags...)
			return
		}
	} else {
		ruleValues = []attr.Value{} // make it an empty list
	}

	data.Rules, _diags = types.ListValue(
		types.ObjectType{AttrTypes: rulesModelAttrType},
		ruleValues,
	)
	if _diags.HasError() {
		resp.Diagnostics.Append(_diags...)
		return
	}

	// save into the Terraform state.
	data.Id = types.StringValue(scheduleUUID)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "Created a schedule")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreVMSnapshotScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	tflog.Info(ctx, "TTRT HypercoreVMSnapshotScheduleResource READ")
	var data HypercoreVMSnapshotScheduleResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Schedule read ======================================================================
	restClient := *r.client
	scheduleUUID := data.Id.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("TTRT HypercoreSnapshotSchedule Read oldState scheduleUUID=%s\n", scheduleUUID))

	pHc3Schedule := utils.GetVMSnapshotScheduleByUUID(restClient, scheduleUUID)
	if pHc3Schedule == nil {
		resp.Diagnostics.AddError("Schedule not found", fmt.Sprintf("Schedule not found - scheduleUUID=%s", scheduleUUID))
		return
	}
	hc3Schedule := *pHc3Schedule

	var scheduleRules []RulesModel
	var ruleValues []attr.Value
	var diags diag.Diagnostics
	if hc3Schedule["rrules"] != nil {
		hc3Rules := utils.AnyToListOfMap(hc3Schedule["rrules"])
		scheduleRules = make([]RulesModel, len(hc3Rules))
		for i, hc3Rule := range hc3Rules {
			scheduleRules[i].Name = types.StringValue(utils.AnyToString(hc3Rule["name"]))
			scheduleRules[i].Frequency = types.StringValue(utils.AnyToString(hc3Rule["rrule"]))
			scheduleRules[i].StartTimestamp = types.StringValue(utils.AnyToString(hc3Rule["dtstart"]))
			scheduleRules[i].LocalRetentionSeconds = types.Int64Value(utils.AnyToInteger64(hc3Rule["localRetentionDurationSeconds"]))
			scheduleRules[i].RemoteRetentionSeconds = types.Int64Value(utils.AnyToInteger64(hc3Rule["remoteRetentionDurationSeconds"]))
		}
		ruleValues, diags = GetRulesAttrValues(scheduleRules)
		if diags != nil {
			resp.Diagnostics.Append(diags...)
			return
		}
	} else {
		ruleValues = []attr.Value{} // make it an empty list
	}

	scheduleName := utils.AnyToString(hc3Schedule["name"])
	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreSnapshot: schedule_uuid=%s, name=%s, rules=%v\n", scheduleUUID, scheduleName, scheduleRules))

	// ====== Save into the Terraform state ======
	// Save schedule UUID
	data.Id = types.StringValue(scheduleUUID)

	// Save schedule name
	data.Name = types.StringValue(scheduleName)

	// Save schedule rules
	data.Rules, diags = types.ListValue(
		types.ObjectType{AttrTypes: rulesModelAttrType},
		ruleValues,
	)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreVMSnapshotScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	tflog.Info(ctx, "TTRT HypercoreVMSnapshotScheduleResource UPDATE")
	var data_state HypercoreVMSnapshotScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data_state)...)
	var data HypercoreVMSnapshotScheduleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	restClient := *r.client
	scheduleUUID := data.Id.ValueString()
	scheduleName := data.Name.ValueString()

	var scheduleRules []RulesModel
	if len(data.Rules.Elements()) != 0 {
		scheduleRules = make([]RulesModel, len(data.Rules.Elements()))
		diags := data.Rules.ElementsAs(ctx, &scheduleRules, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	var dataStateScheduleRules []RulesModel
	if len(data_state.Rules.Elements()) != 0 {
		dataStateScheduleRules = make([]RulesModel, len(data.Rules.Elements()))
		diags := data_state.Rules.ElementsAs(ctx, &dataStateScheduleRules, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}
	tflog.Debug(
		ctx, fmt.Sprintf(
			"TTRT HypercoreVMSnapshotSchedule Update schedule_uuid=%s REQUESTED schedule_name=%s, rules=%v\n",
			scheduleUUID, scheduleName, scheduleRules),
	)
	tflog.Debug(ctx, fmt.Sprintf(
		"TTRT HypercoreVMSnapshotSchedule Update schedule_uuid=%s STATE     schedule_name=%s, rules=%v\n",
		scheduleUUID, data_state.Name.ValueString(), dataStateScheduleRules),
	)

	var payloadScheduleRules []map[string]any
	if len(scheduleRules) != 0 {
		for _, scheduleRule := range scheduleRules {
			payloadScheduleRules = append(payloadScheduleRules, map[string]any{
				"dtstart":                        scheduleRule.StartTimestamp.ValueString(),
				"rrule":                          scheduleRule.Frequency.ValueString(),
				"name":                           scheduleRule.Name.ValueString(),
				"localRetentionDurationSeconds":  scheduleRule.LocalRetentionSeconds.ValueInt64(),
				"remoteRetentionDurationSeconds": scheduleRule.RemoteRetentionSeconds.ValueInt64(),
			})
		}
	} else {
		payloadScheduleRules = []map[string]any{} // empty list
	}

	tflog.Info(ctx, fmt.Sprintf("TTRT Update: scheduleRules = %v", scheduleRules))

	// Update schedule
	payload := map[string]any{
		"name":   scheduleName,
		"rrules": payloadScheduleRules,
	}
	_diag := utils.UpdateVMSnapshotSchedule(restClient, scheduleUUID, payload, ctx)
	if _diag != nil {
		resp.Diagnostics.AddWarning(_diag.Summary(), _diag.Detail())
	}

	// TODO: Check if HC3 matches TF

	// Retrieve rules data (it could be inconsistent)
	hc3Schedule := utils.GetVMSnapshotScheduleByUUID(restClient, scheduleUUID)
	var ruleValues []attr.Value
	var diags diag.Diagnostics
	if (*hc3Schedule)["rrules"] != nil {
		hc3Rules := utils.AnyToListOfMap((*hc3Schedule)["rrules"])
		for i := range hc3Rules {
			if scheduleRules[i].RemoteRetentionSeconds.IsUnknown() {
				scheduleRules[i].RemoteRetentionSeconds = types.Int64Value(0)
			}
		}

		ruleValues, diags = GetRulesAttrValues(scheduleRules)
		if diags != nil {
			resp.Diagnostics.Append(diags...)
			return
		}
	} else {
		ruleValues = []attr.Value{} // make it an empty list
	}

	data.Rules, diags = types.ListValue(
		types.ObjectType{AttrTypes: rulesModelAttrType},
		ruleValues,
	)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreVMSnapshotSchedule: schedule_uuid=%s, name=%s, rules=%v", scheduleUUID, scheduleName, scheduleRules))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HypercoreVMSnapshotScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	tflog.Info(ctx, "TTRT HypercoreVMSnapshotScheduleResource DELETE")
	var data HypercoreVMSnapshotScheduleResourceModel

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
	scheduleUUID := data.Id.ValueString()
	taskTag := restClient.DeleteRecord(
		fmt.Sprintf("/rest/v1/VirDomainSnapshotSchedule/%s", scheduleUUID),
		-1,
		ctx,
	)
	taskTag.WaitTask(restClient, ctx)
}

func (r *HypercoreVMSnapshotScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// NOTE: Do we need import state or would it be better to have a data source instead?
	defer utils.RecoverDiagnostics(ctx, &resp.Diagnostics)

	tflog.Info(ctx, "TTRT HypercoreVMSnapshotScheduleResource IMPORT_STATE")

	scheduleUUID := req.ID
	tflog.Info(ctx, fmt.Sprintf("TTRT HypercoreVMSnapshotScheduleResource: scheduleUUID=%s", scheduleUUID))

	restClient := *r.client
	hc3Schedule := utils.GetVMSnapshotScheduleByUUID(restClient, scheduleUUID)

	if hc3Schedule == nil {
		msg := fmt.Sprintf("VM Schedule import, schedule not found -  'schedule_uuid'='%s'.", req.ID)
		resp.Diagnostics.AddError("VM Schedule import error, schedule not found", msg)
		return
	}

	scheduleName := utils.AnyToString((*hc3Schedule)["name"])
	tflog.Info(ctx, fmt.Sprintf("TTRT Import: schedule=%v", *hc3Schedule))

	var scheduleRules []map[string]any
	if (*hc3Schedule)["rrule"] != nil {
		hc3Rules := utils.AnyToListOfMap((*hc3Schedule)["rrule"])
		for _, hc3Rule := range hc3Rules {
			scheduleRules = append(scheduleRules, map[string]any{
				"name":                     hc3Rule["name"],
				"start_timestamp":          hc3Rule["dtstart"],
				"frequency":                hc3Rule["rrule"],
				"local_retention_seconds":  hc3Rule["localRetentionDurationSeconds"],
				"remote_retention_seconds": hc3Rule["remoteRetentionDurationSeconds"],
			})
		}
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), scheduleUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), scheduleName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rules"), scheduleRules)...)
}
