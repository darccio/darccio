<!--
publish: false
template: false
-->

# Datadog domain monitoring with Terraform

This root module manages eight Datadog Synthetic API tests for `dario.cat`.
Datadog's managed Frankfurt location runs them independently of GitHub, Pages
builds and the PDS. The owner selected EU1 and `cc@dario.cat`; EU1 uses
`https://app.datadoghq.eu` and `https://api.datadoghq.eu/`, as listed in
[Datadog's sites reference](https://docs.datadoghq.com/getting_started/site/).

| Tests | Schedule | Failure condition |
| --- | --- | --- |
| Mergo/Cocoon vanity group (four steps) | Every 30 minutes | Any bare URL's exact 308 redirect/query changes, or canonical response fails TLS/HTTP/content-type/HTML metadata checks |
| Other five vanity paths (one group, ten steps) | Every 2 hours | Same exact redirect, TLS, HTTP and metadata checks |
| Two validating DNS resolvers | Daily | Google or Cloudflare cannot return an authenticated `dario.cat` A answer with validation enabled |
| One TLS certificate test | Daily | Invalid certificate or 14 days or fewer until expiration |
| Three registry-expiry tests | Daily | 90, 60 or 30 days or fewer until registry expiration, or usable registry data is missing |

The vanity tests read the same reviewed
[baseline](../../.tools/vanity/internal/vanity/baseline.json) as the Go checker.
Their bundled HTML tokenizer checks actual tags rather than matching example
text, ignores comments/scripts, rejects metadata outside the head, and retains
duplicate-tag multiplicity. The body limit is 1 MiB. Datadog does not follow
redirects: the first step verifies the observed single hop and exact Location,
then the second requests the expected slash URL directly. A failed bare step
remains critical while allowing the canonical check to run. Every canonical
step also remains critical and lets later packages run after a failure. Alerts
are per group and identify the failed step and URL; a group recovers when all
of its steps pass. No browser test or client JavaScript is needed.

Two multistep groups fit the owner's five-test account quota and
[Datadog's default ten-step limit](https://docs.datadoghq.com/synthetics/multistep/).
Grouping reduces test count; each of the fourteen requests remains billable.

The owner selected this schedule to target **about $5 per month**. Independent
checks for Mergo/Cocoon use a thirty-minute cadence; package-specific failures
in the other imports can take up to two hours to be detected. Resolver and expiry
checks run daily. The full GitHub backstop remains every thirty minutes.

The normal monitoring budget is **9,540 API step-runs per 30 days** for one
location, before retries, manual runs and the temporary notification probe.
That is 5,760 priority-import requests, 3,600 other-import requests and 180
auxiliary requests. A 31-day month schedules **9,858** runs.
Each multistep request and each additional location contributes billed runs.

| Published rate per 10,000 API runs | Estimate per 30-day month | Estimate per 31-day month |
| --- | --- | --- |
| $5 with an annual agreement | **$4.77/month** | **$4.93/month** |
| $6 with a month-to-month agreement | $5.72/month | $5.91/month |
| $7.20 on-demand | $6.87/month | $7.10/month |

Rates were checked on 12 October 2026 against the
[Datadog price list](https://www.datadoghq.com/pricing/list/); see also the
[per-step billing definition](https://www.datadoghq.com/pricing/?product=synthetic-monitoring).
Your account's agreement determines actual charges. Set local
`api_test_rate_usd_per_10000` to its actual rate to get the appropriate estimate;
this input changes the estimate only. Usage outputs are calculated from the
actual planned schedules and step counts and report zero for paused production
tests. The estimate excludes retries, manual runs and the optional notification
probe and does not impose an account spending cap.

## Create and inspect the tests

Install Terraform 1.7 or newer (CI validates with 1.16.5). Provider 4.25.0 and its
checksums are locked. Use an organization API key and a scoped application key
with `synthetics_read` and `synthetics_write` permissions. Obtain them privately
from the EU account; see [Datadog's key guide](https://docs.datadoghq.com/account_management/api-app-keys/).
The provider reads `DD_API_KEY` and `DD_APP_KEY` from the environment; the
configuration contains no credential variables.

From the repository root, in Bash:

```sh
cd .infra/datadog
umask 077
cp terraform.tfvars.example terraform.tfvars
read -r -s -p 'Datadog API key: ' DD_API_KEY
printf '\n'
read -r -s -p 'Datadog application key: ' DD_APP_KEY
printf '\n'
export DD_API_KEY DD_APP_KEY
terraform init -input=false -lockfile=readonly
terraform validate
terraform plan -out=monitoring.tfplan
terraform apply monitoring.tfplan
```

Review that a fresh installation creates eight **paused** tests in the intended EU
account and uses your selected recipient. Open them in Datadog and run each on
demand to verify actual request and assertion execution, including the bundled
parser in Datadog's sandbox. Local VM tests and Terraform schema validation
cannot establish runtime acceptance by the hosted service. If an assertion is
rejected, retain its error and fix the implementation before activation.

This module initially uses Terraform's local backend. Keep the state and its
backups in private durable storage and run from the same initialized directory.
For shared management, configure your existing encrypted remote backend and
migrate the state before another operator runs it. State, saved plans, local
variables and credentials stay out of Git; Terraform's dependency lock stays
committed. See [Terraform state storage](https://developer.hashicorp.com/terraform/language/state/backends).

## Resume a partially failed apply

OpenTofu can manage this module with the same `init`, `validate`, `plan` and
`apply` commands. After an apply creates some tests and then fails, keep the
existing local state, backups and variables in this directory. The successful
resources are recorded in state; a failed apply is not rolled back.

The first owner apply created the certificate, two DNSSEC and three registry
tests, while Datadog rejected all seven vanity tests. The bundled HTML entity
table contained literal double braces which Datadog interpreted as a global
variable reference. The assertion build now escapes that string data without
changing its JavaScript value. CI checks all seven injected assertions and both
notification-probe variants for accidental variable openers.

A second rejection identified a separate size problem: the 94,833-byte injected
assertion and a trivial padded script of the same size were both rejected.
Paused size probes accepted 32,768 bytes and rejected 65,536 bytes; those
observations do not establish Datadog's exact limit. A finer probe accepted
51,200 bytes and rejected 60,000 bytes. The compact build retains
parse5's tokenizer and entity decoding while removing unused diagnostics and
location tracking. Its build checks every injected route and notification
variant against a conservative 49,000-byte budget. The owner successfully
created a paused HTTP test with the actual 48,858-byte Mergo assertion and
verified cleanup. This establishes API acceptance of the replacement code;
managed-location execution still needs verification before activation.

The next owner apply created five vanity tests (darna, awaken, errors, x and
xdr), while Mergo and Cocoon hit `Quota reached for multi tests: 5`. The grouped
layout reuses two confirmed existing resources through `moved` blocks:

| Existing resource | Group | Existing public ID retained |
| --- | --- | --- |
| `vanity["darna"]` | `vanity["priority"]`: Mergo/Cocoon | `kv2-z2c-77h` |
| `vanity["xdr"]` | `vanity["other"]`: remaining five | `m4f-5g8-vxh` |

The existing Awaken, Errors and X test resources become redundant and are
removed by the same reviewed plan. Both groups update their reused resources
in place, so migration needs no extra multistep quota. Use the same initialized
directory, existing state and variables to create a **fresh** plan:

```sh
tofu plan -out=monitoring.tfplan
tofu apply monitoring.tfplan
```

Review the plan before applying it. For this eleven-resource partial state and
the same variables, expect **0 additions, 2 in-place changes and 3 destructions**.
The two moved resources retain their public IDs and the six certificate,
DNSSEC and renewal resources remain unchanged. Stop if the plan replaces a
group or changes those auxiliary tests unexpectedly. Keep the existing state;
removing resources from state leaves remote tests consuming account quota.
Do not reuse the failed saved plan or delete tests manually before applying.
On-demand hosted assertions, activation and received notices remain the next
verification steps.

The offline migration regression builds prior state from the pinned pre-group
revision and eleven fake resource IDs. With provider 4.25.0 it verifies two
in-place updates, three redundant-test removals, six auxiliary no-ops and both
retained IDs. Its isolated provider uses fake keys and a loopback-only endpoint
with validation and refresh disabled; it never reads real state or plans and
never applies resources. Run it with an initialized provider cache:

```sh
python3 tests/quota-migration.py
```

## Activate production monitoring

In your local `terraform.tfvars`, change `enabled` to `true`, then review and
apply a fresh plan:

```sh
terraform plan -out=monitoring.tfplan
terraform apply monitoring.tfplan
terraform output test_ids
terraform output scheduled_requests_per_30_days
terraform output scheduled_requests_per_31_days
terraform output estimated_scheduled_cost_usd
```

The existing tests become live, with one failed location sufficient to alert,
no additional failure-duration window, and one short retry for vanity/DNSSEC/
RDAP tests. Hourly reminders accompany continuing availability/DNSSEC failures;
expiry failures remind daily. A successful subsequent scheduled run recovers.
Alerts and recovery messages include `@cc@dario.cat` without state-specific
conditions. Add a reachable fallback handle outside `dario.cat` to the local
`notification_handles` list when available, so a domain failure cannot hide its
own notification.

## Receive a real failure and recovery notice

Create only the additional temporary test with a deliberately wrong expected
Mergo repository. It requests the unchanged public page; production tests and
tags keep their normal expectations.

```sh
terraform plan -var='notification_probe_enabled=true' -out=notification.tfplan
terraform apply notification.tfplan
```

Wait for its scheduled wrong-metadata failure and confirm the email arrived.
Then restore the expectation on the **same** temporary test:

```sh
terraform plan -var='notification_probe_enabled=true' \
  -var='notification_probe_should_fail=false' -out=notification.tfplan
terraform apply notification.tfplan
```

Confirm a scheduled successful run and its received recovery email. Remove the
temporary test by planning/applying normally again (the local file keeps
`notification_probe_enabled = false`). Verify that the plan deletes only this
additional test and keeps the eight production resources.

For each expiry test, use Datadog's **Simulate Notifications** action and receive
the `[TEST]` notice. These prove routing; local threshold fixtures prove the
90/60/30-day calculations. Simulated notifications and paused manual tests do
not prove that a real monitor failure/recovery transition executed. See
[Datadog's notification simulation](https://docs.datadoghq.com/synthetics/notifications/)
and [alert state rules](https://docs.datadoghq.com/synthetics/guide/how-synthetics-monitors-trigger-alerts/).

Record the activation date, location, last scheduled successes, temporary
failure/recovery timestamps and expiry-notification receipt locations in
[DOMAIN-OPERATIONS.md](../../docs/DOMAIN-OPERATIONS.md). Keep private evidence
and account material outside Git. T01 closes after those operational checks;
the prepared module alone does not close it.

## Maintain and validate

The standalone parser is committed with its pinned source, build dependencies
and license notices under `assertions/`.

The four unchanged parse5 TypeScript files in `assertions/vendor/parse5/` are
pinned to the upstream release commit and checked against `SOURCE.json` before
building. Compiling their original TypeScript lets esbuild inline state enums.
The build specializes only unused diagnostic/location callbacks and derived
tag IDs, and shortens an explicit list of internal property names. It leaves
quoted dictionary keys, Datadog APIs and baseline metadata keys intact. Full
parse5 and entities license notices remain in the generated assertion; no
source fetch or module loader is needed by Datadog.
Each parser fixture also runs against the unmodified npm tokenizer in the same
restricted runtime and compares acceptance or the exact rejection message.

To rebuild it after a reviewed change:

```sh
npm --prefix assertions ci --ignore-scripts --no-audit --no-fund
npm --prefix assertions run build
node assertions/build.mjs --check
node --test assertions.test.mjs assertions/*.test.mjs
terraform fmt -check -recursive
terraform init -backend=false -input=false -lockfile=readonly
terraform validate
terraform test
python3 tests/quota-migration.py
```

The Terraform tests use a mocked provider; CI needs no Datadog credentials and
does not apply resources. Real managed-location execution and email receipt
remain account checks. Review monitor plans whenever the shared baseline or
the site's redirect policy changes. Keep these resources independent of the
site deployment workflow.

To pause the production checks, set local `enabled = false` and plan/apply;
disable an active notification probe separately. Terraform destroy removes
only resources recorded in this module's state. Pausing/removing monitoring
does not change DNS, Pages, public content or the GitHub backstop.
