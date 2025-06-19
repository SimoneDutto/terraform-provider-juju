// Copyright 2023 Canonical Ltd.
// Licensed under the Apache License, Version 2.0, see LICENCE file for details.

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/juju/juju/api/client/action"
	"github.com/juju/juju/rpc/params"
	"github.com/juju/terraform-provider-juju/internal/juju"
	"github.com/juju/terraform-provider-juju/internal/wait"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &actionResource{}
var _ resource.ResourceWithConfigure = &actionResource{}
var _ resource.ResourceWithImportState = &actionResource{}

func NewActionResource() resource.Resource {
	return &actionResource{}
}

type actionResource struct {
	client *juju.Client

	// subCtx is the context created with the new tflog subsystem for applications.
	subCtx context.Context
}

type actionResourceModel struct {
	ActionName types.String `tfsdk:"action_name"`
	ModelName  types.String `tfsdk:"model_name"`
	Args       types.Map    `tfsdk:"args"`
	Output     types.Map    `tfsdk:"output"`
	Receiver   types.String `tfsdk:"receiver"` // The receiver of the action, if applicable.
	// ID required by the testing framework
	ID types.String `tfsdk:"id"`
}

func (s *actionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (s *actionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*juju.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *juju.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	s.client = client
	// Create the local logging subsystem here, using the TF context when creating it.
	s.subCtx = tflog.NewSubsystem(ctx, LogResourceSSHKey)
}

func (s *actionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action"
}

func (s *actionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resource representing a juju action.",
		Attributes: map[string]schema.Attribute{
			"model_name": schema.StringAttribute{
				Description: "The name of the model to operate in.",
				Required:    true,
			},
			"action_name": schema.StringAttribute{
				Description: "The name of the model to operate in.",
				Required:    true,
			},
			"args": schema.MapAttribute{
				Description: "args for action",
				Optional:    true,
				ElementType: types.StringType,
			},
			"receiver": schema.StringAttribute{
				Description: "The receiver of the action, if applicable.",
				Required:    true,
			},
			"output": schema.MapAttribute{
				ElementType: types.StringType,
				Description: "The output of the action.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (s *actionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Prevent panic if the provider has not been configured.
	if s.client == nil {
		addClientNotConfiguredError(&resp.Diagnostics, "ssh_key", "create")
		return
	}

	var plan actionResourceModel

	// Read Terraform configuration from the request into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	actionParams := make(map[string]string, len(plan.Args.Elements()))
	resp.Diagnostics.Append(plan.Args.ElementsAs(ctx, &actionParams, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	actionParamsInterface := make(map[string]interface{}, len(actionParams))
	for k, v := range actionParams {
		actionParamsInterface[k] = v
	}

	enqActionId, err := s.client.Actions.EnqueueAction(
		juju.EnqueueActionArgs{
			ModelName: plan.ModelName.ValueString(),
			ActionQ: action.Action{
				Receiver:   plan.Receiver.ValueString(),
				Name:       plan.ActionName.ValueString(),
				Parameters: actionParamsInterface,
			},
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Enqueuing Action",
			fmt.Sprintf("Could not enqueue action %q: %s", plan.ActionName.ValueString(), err),
		)
		return
	}
	actionResult, err := wait.WaitFor(wait.WaitForCfg[juju.ActionResultArgs, action.ActionResult]{
		Context: ctx,
		Input: juju.ActionResultArgs{
			ModelName: plan.ModelName.ValueString(),
			ActionId:  enqActionId,
		},
		GetData:        s.client.Actions.ActionResult,
		DataAssertions: []wait.Assert[action.ActionResult]{assertActionRunning},
		NonFatalErrors: []error{juju.RetryReadError},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Waiting for Action Completion",
			fmt.Sprintf("Could not wait for action %q to complete: %s", plan.ActionName.ValueString(), err),
		)
		return
	}
	outputMap := make(map[string]types.String, len(actionResult.Output))
	for k, v := range actionResult.Output {
		outputMap[k] = types.StringValue(fmt.Sprint(v))
	}
	plan.Output, resp.Diagnostics = types.MapValueFrom(ctx, types.StringType, outputMap)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", plan.ModelName.ValueString(), plan.ActionName, enqActionId))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (s *actionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Prevent panic if the provider has not been configured.
	if s.client == nil {
		addClientNotConfiguredError(&resp.Diagnostics, "ssh_key", "read")
		return
	}

	var plan actionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
}

func (s *actionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (s *actionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func assertActionRunning(resultFromAPI action.ActionResult) error {
	if resultFromAPI.Error != nil {
		return resultFromAPI.Error
	}
	switch resultFromAPI.Status {
	case params.ActionRunning, params.ActionPending:
		return juju.NewRetryReadError("action is still running or pending, waiting for completion")
	case params.ActionCompleted:
		return nil
	default:
		return errors.New("action is not running or completed, status: " + resultFromAPI.Status)
	}
}
