terraform {
  required_version = ">= 1.7, < 2.0"

  required_providers {
    datadog = {
      source  = "DataDog/datadog"
      version = "= 4.25.0"
    }
  }
}

# Authentication comes from DD_API_KEY and DD_APP_KEY, never Terraform variables.
provider "datadog" {
  api_url = "https://api.${var.datadog_site}/"
}
