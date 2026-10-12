mock_provider "datadog" {}

run "paused_by_default" {
  command = plan

  assert {
    condition     = length(datadog_synthetics_test.vanity) == 2 && length(datadog_synthetics_test.domain_expiry) == 3 && length(datadog_synthetics_test.dnssec) == 2 && length(datadog_synthetics_test.notification_probe) == 0
    error_message = "Keep two vanity groups covering seven packages, both resolvers and three renewal thresholds; no notification probe by default."
  }
  assert {
    condition     = alltrue([for test in datadog_synthetics_test.vanity : test.status == "paused"]) && datadog_synthetics_test.certificate.status == "paused" && alltrue([for test in datadog_synthetics_test.domain_expiry : test.status == "paused"]) && alltrue([for test in datadog_synthetics_test.dnssec : test.status == "paused"])
    error_message = "Creating the reviewed resources must not start scheduled monitoring by default."
  }
  assert {
    condition     = output.scheduled_requests_per_30_days == 0 && output.scheduled_requests_per_31_days == 0 && output.estimated_scheduled_cost_usd.per_30_day_month == 0
    error_message = "Paused resources must report zero scheduled runs."
  }
}

run "active_public_probes" {
  command = plan
  variables {
    enabled = true
  }

  assert {
    condition     = alltrue([for group, test in datadog_synthetics_test.vanity : test.status == "live" && test.options_list[0].tick_every == (group == "priority" ? 1800 : 7200) && test.options_list[0].min_failure_duration == 0 && test.options_list[0].min_location_failed == 1 && test.options_list[0].allow_insecure == false])
    error_message = "Keep priority checks at thirty minutes and the other group at two hours with immediate single-location alerts."
  }
  assert {
    condition     = length(datadog_synthetics_test.vanity["priority"].api_step) == 4 && length(datadog_synthetics_test.vanity["other"].api_step) == 10 && alltrue([for test in datadog_synthetics_test.vanity : length(test.api_step) <= 10])
    error_message = "Two groups must retain fourteen URL checks within the ten-step per-test default."
  }
  assert {
    condition = toset([for step in datadog_synthetics_test.vanity["priority"].api_step : step.request_definition[0].url]) == toset([
      "https://dario.cat/mergo?go-get=1", "https://dario.cat/mergo/?go-get=1",
      "https://dario.cat/cocoon?go-get=1", "https://dario.cat/cocoon/?go-get=1",
    ])
    error_message = "Mergo and Cocoon must receive both URL checks on the priority schedule."
  }
  assert {
    condition     = toset(flatten([for test in datadog_synthetics_test.vanity : [for step in test.api_step : step.request_definition[0].url]])) == toset(flatten([for name in keys(local.routes) : ["https://dario.cat/${name}?go-get=1", "https://dario.cat/${name}/?go-get=1"]]))
    error_message = "Check every reviewed package exactly once in both URL forms, with go-get=1 retained."
  }
  assert {
    condition     = alltrue(flatten([for test in datadog_synthetics_test.vanity : [for step in test.api_step : step.allow_failure == true && step.is_critical == true && step.request_definition[0].timeout == 10 && step.request_definition[0].follow_redirects == false && step.request_definition[0].allow_insecure == false]]))
    error_message = "Every failure must fail its group without preventing later package checks; keep bounded requests and valid TLS."
  }
  assert {
    condition = alltrue(flatten([for test in datadog_synthetics_test.vanity : [for step in test.api_step : (
      endswith(step.request_definition[0].url, "/?go-get=1")
      ? step.assertion[0].target == "200" && step.assertion[1].property == "content-type" && step.assertion[1].target == "^text/html([;].*)?$" && length(step.assertion) == 3
      : step.assertion[0].target == "308" && step.assertion[1].property == "location" && step.assertion[1].target == "^(https://dario[.]cat)?/${split("?", split("/", step.request_definition[0].url)[3])[0]}/[?]go-get=1$" && length(step.assertion) == 2
    )]]))
    error_message = "Bare steps must verify exact redirects; canonical steps must verify HTML and exact metadata."
  }
  assert {
    condition = alltrue(flatten([for test in datadog_synthetics_test.vanity : [for step in test.api_step : alltrue([
      for assertion in step.assertion : assertion.type != "javascript" || (
        strcontains(nonsensitive(assertion.code), jsonencode(local.routes[split("/", step.request_definition[0].url)[3]])) &&
        !strcontains(nonsensitive(assertion.code), "__EXPECTED_METADATA__") &&
        !strcontains(nonsensitive(assertion.code), "{{") && length(nonsensitive(assertion.code)) <= 49000
      )
    ]) if endswith(step.request_definition[0].url, "/?go-get=1")]]))
    error_message = "Every canonical script must inject its own reviewed baseline within the interpolation and size budgets."
  }
  assert {
    condition     = alltrue([for test in datadog_synthetics_test.vanity : strcontains(test.message, "@cc@dario.cat") && strcontains(test.message, "{{synthetics.failed_step.name}}")])
    error_message = "Grouped notifications must identify the failed package step and reach the chosen recipient."
  }
  assert {
    condition     = datadog_synthetics_test.certificate.options_list[0].accept_self_signed == false && datadog_synthetics_test.certificate.options_list[0].ignore_certificate_validation == false && datadog_synthetics_test.certificate.assertion[0].target == "14"
    error_message = "Retain valid certificate checks and the reviewed fourteen-day expiry warning."
  }
  assert {
    condition     = datadog_synthetics_test.certificate.options_list[0].tick_every == 86400 && alltrue([for test in datadog_synthetics_test.dnssec : test.options_list[0].tick_every == 86400])
    error_message = "Certificate and resolver checks must use the selected daily schedule."
  }
  assert {
    condition     = alltrue([for days, test in datadog_synthetics_test.domain_expiry : test.options_list[0].tick_every == 86400 && test.request_definition[0].url == "https://rdap.nic.cat/rdap/domain/dario.cat" && strcontains(test.assertion[1].code, "daysRemaining > ${days}") && !strcontains(test.assertion[1].code, "__MINIMUM_DAYS__")])
    error_message = "Read current registry expiration and emit distinct 90/60/30-day alerts instead of hardcoding an expiry date."
  }
  assert {
    condition     = output.scheduled_requests_per_30_days == 9540 && output.scheduled_requests_per_31_days == 9858
    error_message = "Usage must include each multistep request and scheduled resolver/certificate/expiry tests."
  }
  assert {
    condition     = output.estimated_scheduled_cost_usd.per_30_day_month == 4.77 && output.estimated_scheduled_cost_usd.per_31_day_month == 4.929
    error_message = "The default annual-rate monthly schedule must stay around five dollars in both thirty- and thirty-one-day months."
  }
}

run "additional_location_usage" {
  command = plan
  variables {
    enabled   = true
    locations = ["aws:eu-central-1", "aws:eu-west-1"]
  }
  assert {
    condition     = output.scheduled_requests_per_30_days == 19080 && output.estimated_scheduled_cost_usd.per_30_day_month == 9.54
    error_message = "Every added public location multiplies scheduled request usage."
  }
}

run "account_rate_controls_estimate" {
  command = plan
  variables {
    enabled                     = true
    api_test_rate_usd_per_10000 = 7.20
  }
  assert {
    condition     = output.scheduled_requests_per_30_days == 9540 && output.estimated_scheduled_cost_usd.per_30_day_month == 6.8688
    error_message = "Use the account's configured unit price without silently changing test frequency."
  }
}

run "isolated_failure_notification" {
  command = plan
  variables {
    notification_probe_enabled = true
  }
  assert {
    condition     = length(datadog_synthetics_test.notification_probe) == 1 && datadog_synthetics_test.notification_probe[0].status == "live" && strcontains(datadog_synthetics_test.notification_probe[0].assertion[1].code, "https://example.invalid/notification-probe") && alltrue([for test in datadog_synthetics_test.vanity : test.status == "paused"])
    error_message = "The temporary test must exercise a real wrong-metadata failure while leaving primary tests unchanged."
  }
  assert {
    condition     = !strcontains(nonsensitive(datadog_synthetics_test.notification_probe[0].assertion[1].code), "{{")
    error_message = "The failure probe must use an assertion safe for Datadog variable substitution."
  }
  assert {
    condition     = length(nonsensitive(datadog_synthetics_test.notification_probe[0].assertion[1].code)) <= 49000
    error_message = "The failure probe must retain the same assertion size budget."
  }
}

run "isolated_recovery_notification" {
  command = plan
  variables {
    notification_probe_enabled     = true
    notification_probe_should_fail = false
  }
  assert {
    condition     = strcontains(datadog_synthetics_test.notification_probe[0].assertion[1].code, local.routes["mergo"].go_import[0]) && !strcontains(datadog_synthetics_test.notification_probe[0].assertion[1].code, "https://example.invalid/notification-probe")
    error_message = "The same temporary monitor must recover with the exact production metadata."
  }
  assert {
    condition     = !strcontains(nonsensitive(datadog_synthetics_test.notification_probe[0].assertion[1].code), "{{")
    error_message = "The recovery probe must use an assertion safe for Datadog variable substitution."
  }
  assert {
    condition     = length(nonsensitive(datadog_synthetics_test.notification_probe[0].assertion[1].code)) <= 49000
    error_message = "The recovery probe must retain the same assertion size budget."
  }
}

run "recipient_required" {
  command = plan
  variables {
    notification_handles = []
  }
  expect_failures = [var.notification_handles]
}

run "private_runner_rejected" {
  command = plan
  variables {
    locations = ["pl:private-runner"]
  }
  expect_failures = [var.locations]
}
