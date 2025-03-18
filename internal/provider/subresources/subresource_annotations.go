package subresources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/juju/names/v5"
	"github.com/juju/terraform-provider-juju/internal/juju"
)

type annotations struct {
	client *juju.Client
}

func NewAnnotationsSubresource(client *juju.Client) *annotations {
	return &annotations{
		client: client,
	}
}

func (a *annotations) Schema() schema.MapAttribute {
	return schema.MapAttribute{
		Description: "Annotations for the model",
		Optional:    true,
		ElementType: types.StringType,
		PlanModifiers: []planmodifier.Map{
			mapplanmodifier.UseStateForUnknown(),
		},
	}
}

func (a *annotations) Create(ctx context.Context, planAnnotations types.Map, modelName string, modelUUID string) diag.Diagnostics {
	var annotations map[string]string
	diags := planAnnotations.ElementsAs(ctx, &annotations, false)
	if diags.HasError() {
		return diags
	}
	if len(annotations) > 0 {
		err := a.client.Annotations.SetAnnotations(&juju.SetAnnotationsInput{
			ModelName:   modelName,
			Annotations: annotations,
			EntityTag:   names.NewModelTag(modelUUID),
		})
		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to set annotations for model %q, got error: %s", modelName, err))
			return diags
		}
	}
	return diags
}

func (a *annotations) Update()

func (a *annotations) Delete()
