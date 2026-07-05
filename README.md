# waiter-proxy-kamal

Waiter proxy plugin for managing traffic via Kamal Proxy.

Kamal Proxy is a zero-downtime HTTP proxy designed for container deployments. This plugin interfaces with the `kamal-proxy` CLI to manage service routing during deployments.

## Prerequisites

- kamal-proxy binary installed and running

## Usage

```bash
waiter-proxy-kamal register web1 10.0.0.1:8080 --service web
waiter-proxy-kamal shift web1 --weight 50 --service web
waiter-proxy-kamal drain web1 --timeout 30 --service web
waiter-proxy-kamal remove web1 --service web
waiter-proxy-kamal health web1 --service web
waiter-proxy-kamal info
```

## Global Flags

- `--endpoint` - Kamal Proxy endpoint (default: http://127.0.0.1:80)
- `--service` - Service name (default: web)

## License

MIT
