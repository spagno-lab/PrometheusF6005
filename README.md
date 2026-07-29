# Prometheus F6005 (v3)

A theoretically simple way to export metrics from your ZTE F6005v3 ONT to Prometheus.

## Required environment variables

| Name             | Description                  | Default Value      |
|------------------|------------------------------|--------------------|
| `ENDPOINT`       | HTTP address to the ONT      | http://192.168.1.1 |
| `ONT_USERNAME`   | Username for the ONT         | admin              |
| `ONT_PASSWORD`   | Password for the ONT         | admin              |
| `ONT_RELOGIN_DELAY` | Seconds to wait before replacing an expired session | 60 |

## Usage

Example docker-compose section:

```yaml
  f6005_exporter:
    image: ghcr.io/lucathehacker/prometheusf6005
    environment:
      - ENDPOINT=http://192.168.1.1
      - ONT_USERNAME=admin
      - ONT_PASSWORD=admin
    expose:
      - 80
```

## Notes

The ZTE ONT only accepts one web session at a time. If the session expires or is
invalidated by another login, the exporter drops it and waits
`ONT_RELOGIN_DELAY` seconds before creating a new one. This cooldown lets the ONT
release its previous session. The Prometheus HTTP endpoint stays available and
the process does not need to be restarted.

Concurrent Prometheus scrapes are serialized to avoid opening competing sessions
against the ONT.

###### The code sucks, I know. I wrote it with a bad headache and no sleep.

## Grafana Integration

You can visualize the data with our [Grafana Dashboard](https://grafana.com/grafana/dashboards/23256).  
You can find alerts in the [grafana_alerts.yaml](./grafana_alerts.yaml) file.
