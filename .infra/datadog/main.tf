locals {
  # The Go regression checker and Datadog share the reviewed metadata inventory.
  routes = {
    for route in jsondecode(file("${path.module}/../../.tools/vanity/internal/vanity/baseline.json")) :
    trim(route.path, "/") => route
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

resource "datadog_synthetics_test" "vanity" {
  for_each  = local.routes
  name      = "dario.cat ${each.key}: Go import discovery"
  type      = "api"
  subtype   = "multi"
  status    = local.status
  locations = var.locations
  tags      = concat(local.tags, ["package:${each.key}"])
  message   = <<-EOT
    Go import discovery for dario.cat/${each.key} is failing.
    Check DNS, TLS, the Pages deployment, the query-preserving redirect and exact head metadata.
    Run: make -f .zas/Makefile check-vanity-live
    Runbook: https://github.com/darccio/darccio/blob/main/docs/DOMAIN-OPERATIONS.md
    ${local.notifications}
  EOT

  # Check the observed single redirect explicitly instead of following an
  # unbounded/cross-origin chain or accidentally losing the go-get query.
  api_step {
    name    = "Bare import URL retains go-get=1"
    subtype = "http"
    request_definition {
      method           = "GET"
      url              = "https://dario.cat/${each.key}?go-get=1"
      timeout          = 10
      follow_redirects = false
      allow_insecure   = false
    }
    request_headers = { Accept = "text/html" }
    assertion {
      type     = "statusCode"
      operator = "is"
      target   = "308"
    }
    assertion {
      type     = "header"
      property = "location"
      operator = "matches"
      target   = "^(https://dario[.]cat)?/${each.key}/[?]go-get=1$"
    }
    # Always check the canonical response, including when the bare URL fails.
    allow_failure = true
    is_critical   = true
  }

  api_step {
    name    = "Canonical import URL has exact real HTML metadata"
    subtype = "http"
    request_definition {
      method           = "GET"
      url              = "https://dario.cat${each.value.path}?go-get=1"
      timeout          = 10
      follow_redirects = false
      allow_insecure   = false
    }
    request_headers = { Accept = "text/html" }
    assertion {
      type     = "statusCode"
      operator = "is"
      target   = "200"
    }
    assertion {
      type     = "header"
      property = "content-type"
      operator = "matches"
      target   = "^text/html([;].*)?$"
    }
    assertion {
      type = "javascript"
      code = replace(file("${path.module}/assertions/vanity.js"), "__EXPECTED_METADATA__", jsonencode(each.value))
    }
  }

  options_list {
    tick_every           = 900
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
    tick_every                    = 3600
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
    tick_every           = 900
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
