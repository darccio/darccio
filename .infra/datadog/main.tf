locals {
  # The Go regression checker and Datadog share the reviewed metadata inventory.
  routes = {
    for route in jsondecode(file("${path.module}/../../.tools/vanity/internal/vanity/baseline.json")) :
    trim(route.path, "/") => route
  }
  # Each package retains its bare and canonical request. Grouping fits the
  # account's five-multistep-test quota and the default ten-step limit.
  vanity_groups = {
    priority = { packages = ["mergo", "cocoon"], interval = 1800 }
    other = {
      packages = [for name in keys(local.routes) : name if !contains(["mergo", "cocoon"], name)]
      interval = 7200
    }
  }
  vanity_steps = {
    for group, settings in local.vanity_groups : group => flatten([
      for name in settings.packages : [
        { package = name, canonical = false },
        { package = name, canonical = true },
      ]
    ])
  }
  status        = var.enabled ? "live" : "paused"
  notifications = join(" ", var.notification_handles)
  tags          = ["site:dario.cat", "env:production", "managed-by:terraform", "purpose:domain-protection"]
  dns_resolvers = {
    google     = "https://dns.google/resolve?name=dario.cat&type=A&do=1&cd=0"
    cloudflare = "https://cloudflare-dns.com/dns-query?name=dario.cat&type=A&do=1&cd=0"
  }
  probe_route = merge(local.routes["mergo"], {
    go_import = var.notification_probe_should_fail ? ["dario.cat/mergo git https://example.invalid/notification-probe"] : local.routes["mergo"].go_import
  })
}

# Reuse two tests from the owner's quota-limited partial apply. Updates retain
# their remote IDs; Terraform removes the other three redundant vanity tests.
moved {
  from = datadog_synthetics_test.vanity["darna"]
  to   = datadog_synthetics_test.vanity["priority"]
}

moved {
  from = datadog_synthetics_test.vanity["xdr"]
  to   = datadog_synthetics_test.vanity["other"]
}

resource "datadog_synthetics_test" "vanity" {
  for_each  = local.vanity_groups
  name      = "dario.cat ${each.key}: Go import discovery (${join(", ", each.value.packages)})"
  type      = "api"
  subtype   = "multi"
  status    = local.status
  locations = var.locations
  tags      = concat(local.tags, ["vanity-group:${each.key}"], [for name in each.value.packages : "package:${name}"])
  message   = <<-EOT
    Go import discovery is failing for the dario.cat ${each.key} group.
    Failed step: {{synthetics.failed_step.name}}
    Failed URL: {{synthetics.failed_step.url}}
    Check DNS, TLS, the Pages deployment, the query-preserving redirect and exact head metadata.
    Run: make -f .zas/Makefile check-vanity-live
    Runbook: https://github.com/darccio/darccio/blob/main/docs/DOMAIN-OPERATIONS.md
    ${local.notifications}
  EOT

  dynamic "api_step" {
    for_each = local.vanity_steps[each.key]
    iterator = check
    content {
      name    = "${check.value.package}: ${check.value.canonical ? "canonical metadata" : "bare redirect preserves go-get=1"}"
      subtype = "http"
      request_definition {
        method           = "GET"
        url              = "https://dario.cat/${check.value.package}${check.value.canonical ? "/" : ""}?go-get=1"
        timeout          = 10
        follow_redirects = false
        allow_insecure   = false
      }
      request_headers = { Accept = "text/html" }
      assertion {
        type     = "statusCode"
        operator = "is"
        target   = check.value.canonical ? "200" : "308"
      }
      assertion {
        type     = "header"
        property = check.value.canonical ? "content-type" : "location"
        operator = "matches"
        target   = check.value.canonical ? "^text/html([;].*)?$" : "^(https://dario[.]cat)?/${check.value.package}/[?]go-get=1$"
      }
      dynamic "assertion" {
        for_each = check.value.canonical ? [local.routes[check.value.package]] : []
        content {
          type = "javascript"
          code = replace(file("${path.module}/assertions/vanity.js"), "__EXPECTED_METADATA__", jsonencode(assertion.value))
        }
      }
      # Check every package after a failure, while still failing the group.
      allow_failure = true
      is_critical   = true
    }
  }

  options_list {
    tick_every           = each.value.interval
    allow_insecure       = false
    min_failure_duration = 0
    min_location_failed  = 1
    retry {
      count    = 1
      interval = 1000
    }
    monitor_options {
      renotify_interval = 60
    }
  }

  lifecycle {
    precondition {
      condition     = length(local.vanity_steps[each.key]) > 0 && length(local.vanity_steps[each.key]) <= 10
      error_message = "Each vanity group must fit Datadog's default one-to-ten-step limit."
    }
  }
}

resource "datadog_synthetics_test" "certificate" {
  name      = "dario.cat: TLS certificate expires within ${var.certificate_warning_days} days"
  type      = "api"
  subtype   = "ssl"
  status    = local.status
  locations = var.locations
  tags      = concat(local.tags, ["check:certificate"])
  message   = "Check Cloudflare's certificate renewal and domain configuration. ${local.notifications}"
  request_definition {
    host       = "dario.cat"
    port       = "443"
    servername = "dario.cat"
    timeout    = 10
  }
  assertion {
    type     = "certificate"
    operator = "isInMoreThan"
    target   = tostring(var.certificate_warning_days)
  }
  options_list {
    tick_every                    = 86400
    accept_self_signed            = false
    ignore_certificate_validation = false
    min_failure_duration          = 0
    min_location_failed           = 1
    monitor_options {
      renotify_interval = 1440
    }
  }
}

resource "datadog_synthetics_test" "domain_expiry" {
  for_each  = toset(["90", "60", "30"])
  name      = "dario.cat: registry expiry within ${each.key} days or RDAP unavailable"
  type      = "api"
  subtype   = "http"
  status    = local.status
  locations = var.locations
  tags      = concat(local.tags, ["check:domain-expiry", "threshold-days:${each.key}"])
  message   = "Registry expiry is within ${each.key} days, or its authoritative RDAP data cannot be verified. Check renewal, billing and contacts in SWPanel; verify the paid registry term. ${local.notifications}"
  request_definition {
    method  = "GET"
    url     = "https://rdap.nic.cat/rdap/domain/dario.cat"
    timeout = 10
  }
  request_headers = { Accept = "application/rdap+json, application/json" }
  assertion {
    type     = "statusCode"
    operator = "is"
    target   = "200"
  }
  assertion {
    type = "javascript"
    code = replace(file("${path.module}/expiry.js"), "__MINIMUM_DAYS__", each.key)
  }
  options_list {
    tick_every           = 86400
    follow_redirects     = false
    allow_insecure       = false
    min_failure_duration = 0
    min_location_failed  = 1
    retry {
      count    = 1
      interval = 1000
    }
    monitor_options {
      renotify_interval = 1440
    }
  }
}

resource "datadog_synthetics_test" "dnssec" {
  for_each  = local.dns_resolvers
  name      = "dario.cat: DNSSEC validates through ${each.key}"
  type      = "api"
  subtype   = "http"
  status    = local.status
  locations = var.locations
  tags      = concat(local.tags, ["check:dnssec", "resolver:${each.key}"])
  message   = "dario.cat DNSSEC validation or its ${each.key} validating resolver is failing. Compare the parent DS and Cloudflare DNSKEY; retain signing until a removed parent DS has expired from caches. ${local.notifications}"
  request_definition {
    method  = "GET"
    url     = each.value
    timeout = 10
  }
  request_headers = { Accept = "application/dns-json" }
  assertion {
    type     = "statusCode"
    operator = "is"
    target   = "200"
  }
  assertion {
    type = "javascript"
    code = file("${path.module}/dnssec.js")
  }
  options_list {
    tick_every           = 86400
    follow_redirects     = false
    allow_insecure       = false
    min_failure_duration = 0
    min_location_failed  = 1
    retry {
      count    = 1
      interval = 1000
    }
    monitor_options {
      renotify_interval = 60
    }
  }
}

# An opt-in, short-lived test produces a real wrong-metadata alert and recovery
# without changing any production asset, DNS record or primary monitor.
resource "datadog_synthetics_test" "notification_probe" {
  count     = var.notification_probe_enabled ? 1 : 0
  name      = "dario.cat: temporary alert-delivery probe"
  type      = "api"
  subtype   = "http"
  status    = "live"
  locations = [var.locations[0]]
  tags      = concat(local.tags, ["check:notification-probe"])
  message   = "This is the temporary owner-run notification test. Receive its failure, then its recovery after restoring the expected metadata in Terraform. ${local.notifications}"
  request_definition {
    method  = "GET"
    url     = "https://dario.cat/mergo/?go-get=1"
    timeout = 10
  }
  request_headers = { Accept = "text/html" }
  assertion {
    type     = "statusCode"
    operator = "is"
    target   = "200"
  }
  assertion {
    type = "javascript"
    code = replace(file("${path.module}/assertions/vanity.js"), "__EXPECTED_METADATA__", jsonencode(local.probe_route))
  }
  options_list {
    tick_every           = 60
    follow_redirects     = false
    allow_insecure       = false
    min_failure_duration = 0
    min_location_failed  = 1
  }
}
