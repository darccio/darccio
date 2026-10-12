variable "datadog_site" {
  description = "Datadog region domain; EU1 uses datadoghq.eu. This selects the API endpoint, not the monitored domain."
  type        = string
  default     = "datadoghq.eu"
  validation {
    condition     = contains(["datadoghq.eu", "datadoghq.com", "us3.datadoghq.com", "us5.datadoghq.com", "ap1.datadoghq.com", "ap2.datadoghq.com", "uk1.datadoghq.com"], var.datadog_site)
    error_message = "Use a supported Datadog site domain, for example datadoghq.eu for EU1."
  }
}

variable "notification_handles" {
  description = "Datadog notification handles, including the leading @. Add an independently reachable fallback if available."
  type        = list(string)
  default     = ["@cc@dario.cat"]
  validation {
    condition     = length(var.notification_handles) > 0 && alltrue([for handle in var.notification_handles : can(regex("^@[^\\s]+$", handle))])
    error_message = "Provide at least one Datadog @notification handle without whitespace."
  }
}

variable "locations" {
  description = "Datadog managed public locations. Each added location increases scheduled test runs."
  type        = list(string)
  default     = ["aws:eu-central-1"]
  validation {
    condition     = length(var.locations) > 0 && length(distinct(var.locations)) == length(var.locations) && alltrue([for location in var.locations : startswith(location, "aws:") || startswith(location, "azure:") || startswith(location, "gcp:")])
    error_message = "Choose distinct managed public location IDs; the default is aws:eu-central-1."
  }
}

variable "enabled" {
  description = "Activate all production tests after reviewing the plan and account usage. False creates them paused."
  type        = bool
  default     = false
}

variable "certificate_warning_days" {
  description = "Alert when the HTTPS certificate has no more than this many days remaining."
  type        = number
  default     = 14
  validation {
    condition     = var.certificate_warning_days >= 1 && var.certificate_warning_days <= 90 && floor(var.certificate_warning_days) == var.certificate_warning_days
    error_message = "Certificate warning days must be an integer from 1 through 90."
  }
}

variable "notification_probe_enabled" {
  description = "Create a separate temporary notification test; does not change production pages or assertions."
  type        = bool
  default     = false
}

variable "notification_probe_should_fail" {
  description = "Give only the temporary test intentionally incorrect metadata; set false to exercise recovery."
  type        = bool
  default     = true
}
