package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

func NewEphemeralResource() ephemeral.EphemeralResource {
	return &ThingEphemeralResource{}
}

// ThingEphemeralResource defines the ephemeral resource implementation.
// Some ephemeral.EphemeralResource interface methods are omitted for brevity.
type ThingEphemeralResource struct{}

type ThingEphemeralResourceModel struct {
	Name  types.String `tfsdk:"name"`
	Token types.String `tfsdk:"token"`
}

func (d *ThingEphemeralResource) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_thing"
}

func (e *ThingEphemeralResource) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the thing to retrieve a token for.",
				Required:    true,
			},
			"token": schema.StringAttribute{
				Description: "Token for the thing.",
				Computed:    true,
			},
		},
	}
}

func (e *ThingEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data ThingEphemeralResourceModel

	// Read Terraform config data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Typically ephemeral resources will make external calls, however this example
	// hardcodes setting the token attribute to a specific value for brevity.

	data.Token = types.StringValue(acctest.RandomWithPrefix("token-"))

	// Save data into ephemeral result data
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
