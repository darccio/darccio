output "test_ids" {
  description = "Datadog public IDs for vanity groups and auxiliary tests. These are resource IDs, not credentials."
  value = {
    vanity             = { for name, test in datadog_synthetics_test.vanity : name => test.id }
    certificate        = datadog_synthetics_test.certificate.id
    domain_expiry      = { for days, test in datadog_synthetics_test.domain_expiry : days => test.id }
    dnssec             = { for resolver, test in datadog_synthetics_test.dnssec : resolver => test.id }
    notification_probe = one(datadog_synthetics_test.notification_probe[*].id)
  }
}

locals {
  # Count actual planned API steps and intervals, not a separate schedule copy.
  # All production tests use var.locations; the temporary probe is excluded.
  scheduled_requests_per_day = var.enabled ? length(var.locations) * (
    sum([for test in datadog_synthetics_test.vanity : length(test.api_step) * 86400 / test.options_list[0].tick_every]) +
    86400 / datadog_synthetics_test.certificate.options_list[0].tick_every +
    sum([for test in datadog_synthetics_test.domain_expiry : 86400 / test.options_list[0].tick_every]) +
    sum([for test in datadog_synthetics_test.dnssec : 86400 / test.options_list[0].tick_every])
  ) : 0
}

output "scheduled_requests_per_30_days" {
  description = "Scheduled API steps per 30 days; excludes retries, manual runs and the optional notification probe."
  value       = local.scheduled_requests_per_day * 30
}

output "scheduled_requests_per_31_days" {
  description = "Scheduled API steps in a 31-day month; excludes retries, manual runs and the optional notification probe."
  value       = local.scheduled_requests_per_day * 31
}

output "estimated_scheduled_cost_usd" {
  description = "Monthly schedule estimate at the configured USD rate per 10,000 API steps. Not a spending cap or invoice; excludes retries, manual runs and the optional probe."
  value = {
    per_30_day_month = local.scheduled_requests_per_day * 30 * var.api_test_rate_usd_per_10000 / 10000
    per_31_day_month = local.scheduled_requests_per_day * 31 * var.api_test_rate_usd_per_10000 / 10000
  }
}
