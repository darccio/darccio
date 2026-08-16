<!--
tags: [ "observability", "synology", "datadog" ]
date_created: "2026-08-16T15:59:44+02:00"
-->

# Monitoring a Synology NAS with Datadog

There are two things worth watching on a NAS: the hardware (disks, RAID, temperature) and everything running on top of it. Datadog covers the first over SNMP and the second with its Agent, so I run both out of a single container on the NAS itself.

Here's the whole setup, in the order I'd do it again.

## Step 1: Enable SNMP on DSM

1. Open DSM > Control Panel > Terminal & SNMP > SNMP.
2. Check "Enable SNMP service".
3. Pick a version:
   - SNMPv2c is simpler. Set a custom community string, never `public`.
   - SNMPv3 is the one I'd use. Create a user with SHA for authentication and AES for privacy. Synology's SNMP daemon only speaks SHA-1 and AES-128, so don't reach for SHA256 or AES256.
4. Write down your NAS's LAN IP. You need that one, not `127.0.0.1`, because the Agent runs in a bridged container.

## Step 2: Open the firewall for SNMP

Under Control Panel > Security > Firewall, allow UDP port 161 from wherever the Agent lives. Here that's the NAS itself, but the traffic leaves through a Docker bridge network, so the allowed sources have to include the bridge subnet (usually `172.17.0.0/16`). Otherwise DSM's SNMP daemon never sees the packets.

## Step 3: Install Container Manager

Open Package Center and install Container Manager, which is what Docker got renamed to in DSM 7.2. Launch it once so it initializes before you create anything.

## Step 4: Get a Datadog API key

In Datadog, go to Organization Settings > API Keys and copy an existing key or create a new one. It goes straight into the compose file in the next step.

## Step 5: Create the SNMP config, then the compose project

Order matters here. Create the SNMP config file before the container exists. If the path isn't there when Docker starts the container, Docker creates an empty directory in its place and your config gets silently ignored.

1. In File Station, create the project folder, for example `/docker/datadog-agent` inside the `docker` shared folder that Container Manager set up for you (on whichever volume it landed, usually `volume1`). Inside it, create a `conf.d` subfolder.
2. Inside `conf.d`, create `snmp.yaml` with your NAS IP and the SNMPv3 credentials from step 1:

```yaml
init_config:
  loader: core
  use_device_id_as_hostname: true

instances:
  - ip_address: '192.168.1.XXX'      # your NAS LAN IP from step 1
    snmp_version: 3
    profile: synology-disk-station    # pins Datadog's official Synology profile
    user: 'synology-monitor'
    authProtocol: 'SHA'
    authKey: 'your-authentication-key'
    privProtocol: 'AES'
    privKey: 'your-privacy-key'
    tags:
      - device_type:synology_nas
      - environment:prod
```

3. Save it, then go to Project > Create in Container Manager.
4. Name the project, e.g. `datadog-agent`.
5. For Path, pick the folder from step 1 (`/volume1/docker/datadog-agent`, or wherever yours is). DSM keeps the compose file and project data there.
6. Under Source, choose "Create docker-compose.yml".
7. Paste this, with the `snmp.yaml` bind mount already in `volumes:`:

```yaml
services:
  datadog-agent:
    image: datadog/agent:7
    container_name: datadog-agent
    restart: unless-stopped
    network_mode: bridge          # host mode collides with DSM nginx on 5001
    cgroup: host                  # compose equivalent of `docker run --cgroupns host`
    environment:
      DD_API_KEY: "your-datadog-api-key"
      DD_SITE: "datadoghq.com"
      DD_ENV: "prod"
      DD_HOSTNAME: "synology-nas"
      DD_APM_ENABLED: "false"
      DD_PROCESS_AGENT_ENABLED: "true"
      DD_LOGS_ENABLED: "true"
      DD_LOGS_CONFIG_CONTAINER_COLLECT_ALL: "true"
      DD_CHECK_RUNNERS: "2"       # lower to 1 on Atom/Celeron models
      # Leave DD_SYSTEM_PROBE_ENABLED alone. DSM's 4.4 kernel lacks the eBPF support it needs.
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - /proc/:/host/proc/:ro
      - /sys/fs/cgroup/:/host/sys/fs/cgroup:ro   # goes with cgroup: host above, for correct container and process metrics
      - /volume1/docker/datadog-agent/conf.d/snmp.yaml:/etc/datadog-agent/conf.d/snmp.d/conf.yaml:ro
    labels:
      com.datadoghq.ad.check_names: '["snmp"]'
```

Fix the path in that last volume line if you picked a different folder.

8. Click Next, look over the settings and deploy. Container Manager pulls `datadog/agent:7` and starts it with the SNMP config already mounted, so there's nothing left to edit afterwards.

## Step 6: Check that it came up

Open a terminal from Container Manager, or SSH into DSM and run `docker exec -it datadog-agent bash`, then:

```bash
agent status
```

You want sections for `snmp`, `process` and `logs agent`, and no restart or crash loop. For the SNMP side specifically:

```bash
agent status | grep -A 10 snmp
```

The tag to look for is `vendor:synology`. If it's there, the mounted `snmp.yaml` was read and the `synology-disk-station` profile loaded, which means disks, RAID, SMART and temperatures all come for free. You never have to write an OID list by hand.

Then open Infrastructure > Devices in Datadog. The NAS shows up there with its model, serial number and DSM version as metadata.

## Step 7: Confirm metrics are flowing

In Metrics Explorer, search for:

- `snmp.synology.diskTemperature`
- `snmp.synology.diskStatus`
- `snmp.synology.raidStatus`
- `snmp.synology.system.temperature`
- `system.cpu.*` and `system.mem.*`, which come from the host check rather than SNMP

Give it about five minutes. If nothing turns up, skip to the troubleshooting list at the end.

## Step 8: Alerts

Monitors > New Monitor > Metric, then set up these:

| Metric | Condition | Severity |
|---|---|---|
| `snmp.synology.system.temperature` | > 60°C | Critical |
| `snmp.synology.system.temperature` | > 50°C | Warning |
| `snmp.synology.diskTemperature` | > 50°C (per disk) | Warning |
| `snmp.synology.diskStatus` | ≠ 1 (normal) | Critical |
| `snmp.synology.raidStatus` | in {degrade, crashed} | Critical |
| Volume usage (`raidFreeSize`/`raidTotalSize`) | > 90% used | Critical |
| Volume usage | > 80% used | Warning |

The temperature numbers are a starting point. Check what your drives are actually rated for and move them.

## Step 9: A dashboard

Dashboards > New Dashboard, then:

- Color-coded query-value widgets for disk status, RAID status and current temperature. These are the ones you actually glance at.
- Timeseries for CPU, memory, network throughput and storage I/O.
- A table grouped by `synology_disk_id` or `synology_raid_name` for the per-disk and per-volume breakdown.

I keep the NDM device page for the NAS pinned next to it, since that's where the topology and metadata live.

## Maintenance

DSM updates have a habit of resetting SNMP settings, so after each one check that the service is still enabled, the credentials still match and the Docker bridge subnet is still in the allowed sources. After a Container Manager update, run `agent status` once to confirm the container came back cleanly, since network defaults sometimes shift.

## When something breaks

The container restart-loops or throws certificate errors: that's host networking. Switch to `network_mode: bridge`.

No SNMP data at all: check that `ip_address` is the LAN IP and not localhost, that the bridge subnet is allowed through the firewall on UDP 161, and that the version and credentials match what's on the SNMP page exactly. SHA and AES, not SHA256 and AES256.

`snmp.yaml` mounted as a directory instead of a file: the file didn't exist yet when the container was first created. Delete the directory Docker made, create the real file in File Station, redeploy the project.

The Agent eating CPU: drop `DD_CHECK_RUNNERS` to 1.

No fan or power status: that one's expected. Those scalar OIDs ship disabled in Datadog's profile. Add them to the SNMP config by hand if you want them.
