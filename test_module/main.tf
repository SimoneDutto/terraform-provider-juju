terraform {
  required_providers {
    http = {
      source  = "hashicorp/http"
      version = "~> 3.0"
    }
  }
}

# Variables for the query parameters
variable "charm_name" {
  description = "Name of the charm (e.g., postgresql)"
  type        = string
  default     = "postgresql"
}

variable "channel_name" {
  description = "Channel name (e.g., 14/stable, 16/edge)"
  type        = string
  default     = "14/stable"
}

variable "base_channel" {
  description = "Base Ubuntu channel (e.g., 22.04, 24.04)"
  type        = string
  default     = "ubuntu@22.04"
}

variable "base_architecture" {
  description = "Base architecture (e.g., amd64, arm64)"
  type        = string
  default     = "amd64"
}

# Fetch charm info from Charmhub API
data "http" "charmhub_info" {
  url = "https://api.charmhub.io/v2/charms/info/${var.charm_name}?fields=channel-map.revision.revision"

  request_headers = {
    Accept = "application/json"
  }

  lifecycle {
    postcondition {
      condition     = self.status_code == 200
      error_message = "Failed to fetch charm info from Charmhub API"
    }
  }
}

# Parse the JSON response
locals {
  charmhub_response = jsondecode(data.http.charmhub_info.response_body)
  base_version      = split("@", var.base_channel)[1]
  # Filter channel-map to find matching entry
  matching_channels = [
    for entry in local.charmhub_response["channel-map"] :
    entry if(
      entry.channel.name == var.channel_name &&
      entry.channel.base.channel == local.base_version &&
      entry.channel.base.architecture == var.base_architecture
    )
  ]

  # Extract revision number (take first match)
  revision = length(local.matching_channels) > 0 ? local.matching_channels[0].revision.revision : null
}

check "revision_found" {
  assert {
    condition     = local.revision != null
    error_message = "No matching revision found for charm '${var.charm_name}' with channel '${var.channel_name}', base '${var.base_channel}', and architecture '${var.base_architecture}'. Please verify the combination exists in Charmhub."
  }
}

# Output the revision number
output "charm_revision" {
  description = "The revision number for the specified charm channel and base"
  value       = local.revision
}
