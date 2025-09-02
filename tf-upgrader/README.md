# tf-upgrader

A command-line tool to upgrade Terraform configurations for the Juju provider from the old `model` field to the new `model_uuid` field.

## What it does

This tool automatically transforms Juju provider resources that use `model = juju_model.*.name` references to use `model_uuid = juju_model.*.uuid` instead. It supports the following resource types:

- `juju_application`
- `juju_offer` 
- `juju_ssh_key`
- `juju_access_model`
- `juju_access_secret`
- `juju_integration`

It also upgrades output blocks that reference `juju_model.*.name` to use `juju_model.*.uuid`.

## Installation

```bash
go install github.com/juju/terraform-provider-juju/tf-upgrader@latest
```

## Usage

Upgrade a single file:
```bash
tf-upgrader path/to/file.tf
```

Upgrade all `.tf` files in a directory:
```bash
tf-upgrader path/to/terraform/directory
```

## Examples

**Before:**
```terraform
resource "juju_application" "app" {
  name  = "postgresql"
  model = juju_model.test.name
  charm {
    name = "postgresql"
  }
}

output "model_name" {
  value = juju_model.test.name
}
```

**After:**
```terraform
resource "juju_application" "app" {
  name       = "postgresql"
  model_uuid = juju_model.test.uuid
  charm {
    name = "postgresql"
  }
}

output "model_name" {
  value = juju_model.test.uuid
}
```

## What won't be upgraded

- Resources that already use `model_uuid`
- Resources that reference variables (e.g., `model = var.model_name`)
- Resources without model references

The tool will show warnings for variables that contain "model" in their name, as these may need manual review.

## Testing

```bash
go test -v
```

