package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ResourceAction(t *testing.T) {
	modelName := acctest.RandomWithPrefix("tf-test-action")
	appName := "test-app"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: frameworkProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceActionBasic(modelName, appName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("juju_application.this", "model", modelName),
					resource.TestCheckResourceAttr("juju_application.this", "name", appName),
				),
			},
		},
	})
}

func testAccResourceActionBasic(modelName, appName string) string {
	return fmt.Sprintf(`
		resource "juju_model" "this" {
		  name = %q
		}
		
		resource "juju_application" "this" {
		  model = juju_model.this.name
		  name = %q
		  charm {
			name    = "traefik-k8s"
    	    channel = "latest/stable"
		  }
		  trust = true
		}

		resource "juju_action" "action" {
		    model_name = juju_model.this.name
  			action_name = "show-proxied-endpoints"
  			receiver    = "${juju_application.this.name}/0"
		}
		`, modelName, appName)
}
