output "test_ids" {
  description = "Datadog public IDs for the managed tests. These are resource IDs, not credentials."
  value = {
    vanity             = { for name, test in datadog_synthetics_test.vanity : name => test.id }
    certificate        = datadog_synthetics_test.certificate.id
    domain_expiry      = { for days, test in datadog_synthetics_test.domain_expiry : days => test.id }
    dnssec             = { for resolver, test in datadog_synthetics_test.dnssec : resolver => test.id }
    notification_probe = one(datadog_synthetics_test.notification_probe[*].id)
  }
}

output "scheduled_requests_per_30_days" {
  description = "Scheduled API steps per 30 days; excludes retries, manual runs and the optional notification probe."
  value       = var.enabled ? length(var.locations) * (length(local.routes) * 2 * 96 * 30 + 24 * 30 + 3 * 30 + length(local.dns_resolvers) * 96 * 30) : 0
}
