#!/usr/bin/env python3
"""Offline provider 4.25.0 plan regression for the quota-limited partial apply.

No apply, refresh, import or account API operation is performed. The isolated
provider has fake credentials, validation disabled and a loopback-only endpoint.
Prior attributes come from the real pre-group configuration's provider plan;
only five known-success vanity instances and six auxiliary instances get fake
remote IDs. Never reads a repository's real state, saved plans or credentials.
"""
import argparse
from collections import Counter
import json
import os
from pathlib import Path
import subprocess
import tempfile
import uuid


PREVIOUS = '4432d447817d5e6c5d9c189efb2e17b4802946bb'
TF_FILES = ['versions.tf', 'main.tf', 'variables.tf', 'outputs.tf', '.terraform.lock.hcl', 'expiry.js', 'dnssec.js', 'assertions/vanity.js']
SUCCESSFUL = {'darna', 'awaken', 'errors', 'x', 'xdr'}
GROUPS = {'priority': (['mergo', 'cocoon'], 1800, 'darna'), 'other': (['awaken', 'darna', 'errors', 'x', 'xdr'], 7200, 'xdr')}


def require(condition, message):
    if not condition:
        raise AssertionError(message)


def safe_environment(config):
    env = {name: value for name, value in os.environ.items()
           if not name.startswith(('DD_', 'DATADOG_', 'AWS_', 'GOOGLE_', 'TF_', 'TOFU_'))
           and name.lower() not in ('http_proxy', 'https_proxy', 'all_proxy')}
    env.update(TF_CLI_CONFIG_FILE=str(config), TF_IN_AUTOMATION='1', CHECKPOINT_DISABLE='1',
               DD_API_KEY='offline-fake-api-key', DD_APP_KEY='offline-fake-app-key')
    return env


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=Path, default=Path(__file__).resolve().parents[3])
    parser.add_argument('--terraform', default='terraform')
    args = parser.parse_args()
    root = args.root.resolve()
    provider_cache = root / '.infra/datadog/.terraform/providers'
    binaries = list(provider_cache.glob('registry.terraform.io/datadog/datadog/4.25.0/*/terraform-provider-datadog_v4.25.0*'))
    require(any(binary.is_file() for binary in binaries), 'Pinned local provider 4.25.0 is required; run terraform init in .infra/datadog first')
    with tempfile.TemporaryDirectory(prefix='dario-datadog-quota-offline-') as temporary:
        verify(root, provider_cache, args.terraform, Path(temporary))


def verify(root, provider_cache, terraform, fixture):
    config = fixture / 'terraform.rc'
    config.write_text('disable_checkpoint = true\n')
    env = safe_environment(config)

    def command(directory, *arguments, json_output=False):
        require(arguments[0] in ('init', 'plan', 'show', 'providers'), 'Only offline plan/schema operations are allowed')
        if arguments[0] == 'plan':
            require('-refresh=false' in arguments, 'Refresh must remain disabled')
        result = subprocess.run([str(terraform), *arguments], cwd=directory, env=env, capture_output=True, text=True, timeout=120)
        if result.returncode:
            # All credentials are fake; omit full plans/assertion code anyway.
            diagnostic = (result.stderr or result.stdout)[-1800:]
            raise AssertionError('Offline %s failed: %s' % (arguments[0], diagnostic))
        return json.loads(result.stdout) if json_output else result.stdout

    def checkout(name, previous):
        destination = fixture / name
        module = destination / '.infra/datadog'
        module.mkdir(parents=True)
        for filename in TF_FILES:
            relative = '.infra/datadog/' + filename
            if previous:
                content = subprocess.check_output(['git', '-C', str(root), 'show', PREVIOUS + ':' + relative])
            else:
                content = (root / relative).read_bytes()
            target = destination / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(content)
        baseline = '.tools/vanity/internal/vanity/baseline.json'
        target = destination / baseline
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(subprocess.check_output(['git', '-C', str(root), 'show', PREVIOUS + ':' + baseline]) if previous else (root / baseline).read_bytes())
        # Terraform's override-file merge replaces just provider configuration,
        # leaving actual resources, dynamic steps and moved blocks untouched.
        (module / 'offline_override.tf.json').write_text(json.dumps({'provider': {'datadog': {
            'api_url': 'http://127.0.0.1:9/', 'api_key': 'offline-fake-api-key',
            'app_key': 'offline-fake-app-key', 'validate': False,
        }}}))
        command(module, 'init', '-no-color', '-input=false', '-backend=false', '-get=false', '-lockfile=readonly', '-plugin-dir=' + str(provider_cache))
        return module

    previous = checkout('previous', True)
    command(previous, 'plan', '-no-color', '-input=false', '-refresh=false', '-var=enabled=true', '-out=previous.tfplan')
    old_plan = command(previous, 'show', '-json', 'previous.tfplan', json_output=True)
    schema = command(previous, 'providers', 'schema', '-json', json_output=True)
    resource_schema = schema['provider_schemas']['registry.terraform.io/datadog/datadog']['resource_schemas']['datadog_synthetics_test']
    state_resources = {}
    old_ids = {}
    for resource in old_plan['planned_values']['root_module']['resources']:
        if resource['type'] != 'datadog_synthetics_test':
            continue
        name, index = resource['name'], resource.get('index')
        if name == 'notification_probe' or name == 'vanity' and index not in SUCCESSFUL:
            continue
        require(name in ('vanity', 'certificate', 'domain_expiry', 'dnssec'), 'Unexpected prior managed resource')
        number = len(old_ids) + 1
        fake_id = 'off-000-%03d' % number
        values = resource['values']
        values['id'] = fake_id
        values['monitor_id'] = 10000 + number
        old_ids[resource['address']] = fake_id
        group = state_resources.setdefault(name, {'mode': 'managed', 'type': resource['type'], 'name': name,
            'provider': 'provider["registry.terraform.io/datadog/datadog"]', 'instances': []})
        instance = {'schema_version': resource_schema['version'], 'attributes': values, 'sensitive_attributes': []}
        if index is not None:
            instance['index_key'] = index
        group['instances'].append(instance)
    require(len(old_ids) == 11, 'Expected five successful vanity and six auxiliary instances')
    current = checkout('current', False)
    (current / 'terraform.tfstate').write_text(json.dumps({'version': 4, 'terraform_version': old_plan['terraform_version'],
        'serial': 1, 'lineage': str(uuid.uuid4()), 'outputs': {}, 'resources': list(state_resources.values())}))
    command(current, 'plan', '-no-color', '-input=false', '-refresh=false', '-var=enabled=true', '-out=migration.tfplan')
    plan = command(current, 'show', '-json', 'migration.tfplan', json_output=True)
    changes = {change['address']: change for change in plan['resource_changes'] if change['mode'] == 'managed'}
    actions = Counter(tuple(change['change']['actions']) for change in changes.values())
    require(actions == Counter({('update',): 2, ('delete',): 3, ('no-op',): 6}), 'Migration must have exactly 0 creates, 2 updates, 3 deletes and 6 no-ops: ' + repr(actions))
    routes = json.loads((root / '.tools/vanity/internal/vanity/baseline.json').read_text())
    by_name = {route['path'].strip('/'): route for route in routes}
    for key, (packages, interval, anchor) in GROUPS.items():
        address = 'datadog_synthetics_test.vanity["%s"]' % key
        change = changes[address]
        old_address = 'datadog_synthetics_test.vanity["%s"]' % anchor
        require(change.get('previous_address') == old_address, 'Moved block must retain the known existing anchor')
        before, after = change['change']['before'], change['change']['after']
        require(before['id'] == after['id'] == old_ids[old_address], 'Moved update must retain the existing remote ID')
        require(after['subtype'] == 'multi' and after['status'] == 'live', 'Groups must remain live multistep tests')
        require(after['options_list'][0]['tick_every'] == interval, 'Actual planned group cadence differs')
        steps = after['api_step']
        require(len(steps) == len(packages) * 2, 'Actual planned group step count differs')
        for i, package in enumerate(packages):
            bare, canonical = steps[i * 2:i * 2 + 2]
            for step in (bare, canonical):
                request = step['request_definition'][0]
                require(request['method'] == 'GET' and request['timeout'] == 10 and request['follow_redirects'] is False and request['allow_insecure'] is False, 'Grouped request safety options changed')
                require(step['allow_failure'] is True and step['is_critical'] is True, 'Every package must continue after failure while failing the group')
            require(bare['request_definition'][0]['url'] == 'https://dario.cat/%s?go-get=1' % package, 'Bare discovery URL/query differs')
            require(canonical['request_definition'][0]['url'] == 'https://dario.cat/%s/?go-get=1' % package, 'Canonical discovery URL/query differs')
            require(bare['assertion'][0]['target'] == '308' and canonical['assertion'][0]['target'] == '200', 'Grouped HTTP status assertions changed')
            script = canonical['assertion'][2]
            require(script['type'] == 'javascript' and all(value in script['code'] for value in by_name[package]['go_import'] + by_name[package]['go_source']), 'Canonical exact metadata assertion lost its baseline')
            require('__EXPECTED_METADATA__' not in script['code'] and '{{' not in script['code'], 'Unresolved/unsafe code marker in grouped assertion')
            require(len(script['code'].encode('utf-8')) <= 49000, 'Grouped metadata assertion exceeds the 49,000 UTF-8 byte budget')
    deleted = {address for address, change in changes.items() if change['change']['actions'] == ['delete']}
    require(deleted == {'datadog_synthetics_test.vanity["%s"]' % name for name in SUCCESSFUL - {'darna', 'xdr'}}, 'Delete only the three redundant successful vanity tests')
    for address, change in changes.items():
        if change['change']['actions'] == ['no-op']:
            require(change['change']['before'] == change['change']['after'] and change['change']['after']['id'] == old_ids[address], 'Auxiliary resource values/IDs must remain unchanged')
    print(json.dumps({'provider': '4.25.0', 'prior_fake_resources': len(old_ids), 'create': 0, 'update': 2,
        'delete': 3, 'auxiliary_no_op': 6, 'moved_ids_preserved': True, 'priority_steps': 4, 'other_steps': 10,
        'account_calls': 'none; loopback endpoint, validate=false, refresh=false', 'temporary_files': 'removed on exit'}, sort_keys=True))


if __name__ == '__main__':
    main()
