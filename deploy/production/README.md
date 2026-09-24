# Production deployment package

This directory is a credential-free production template. It is not ready to run until the server, domain, image digest, and secrets are supplied and verified.

## Safety properties

- Sub2API binds to `127.0.0.1:8080` on the host.
- PostgreSQL and Redis have no host port mapping.
- PostgreSQL bootstrap credentials are separate from the non-superuser application owner.
- PostgreSQL and Redis share an internal-only network; only Sub2API joins the outbound network.
- Required secrets fail closed when omitted.
- URL validation defaults reject insecure HTTP and private hosts.
- Container logs rotate instead of growing without limit.

## Server preparation

1. Copy this directory to `/opt/sub2api/compose`.
2. Copy `.env.production.example` to `.env`, generate every secret, and set mode `600`.
3. Keep PostgreSQL and Redis on the tested immutable digests; replace the application tag with its verified GHCR digest after the first release build.
4. Create `data`, `postgres_data`, and `redis_data` with the ownership required by the images.
5. Keep `postgres-init` and its bootstrap script owned by root but executable/readable by the container (`755`); an unreadable bind mount prevents PostgreSQL initialization.
6. Replace the Caddy domain and install it as the host Caddy configuration. Create `/var/log/caddy` and `sub2api-access.log` as `caddy:caddy` before loading the config.
7. Format and validate the installed Caddyfile before enabling or reloading Caddy.
8. Keep port 80 restricted to Cloudflare. For the public `api-yue99.xyz` direct API, allow TCP 443 to the server's public IPv4 only after loading the primary site's Cloudflare peer-IP restriction in Caddy; keep SSH independently accessible.
9. Validate with `docker compose config` before starting containers.

## Public direct API

`api-yue99.xyz` uses a DNSPod A record pointing directly to `154.36.168.12`, without a CDN proxy. Clients use `https://api-yue99.xyz/v1` and their existing Sub2API API keys. There is no additional customer IP allowlist or direct-access entitlement. Existing per-key restrictions still apply.

The Caddy template keeps the primary `api-yue88.xyz` site restricted to Cloudflare's TCP peer networks and loopback addresses. Update these networks together with the firewall when Cloudflare publishes changes. The direct host forwards gateway paths and `/health` only; web UI, management and payment paths return 404. Forwarding headers are overwritten with the direct peer IP before reaching Sub2API. If custom forwarded-IP headers are enabled later, normalize them at this ingress too.

The direct host obtains and renews a publicly trusted certificate using TLS-ALPN-01 on TCP 443. Port 80 is not needed for its certificate validation. The direct API does not enable response compression and preserves immediate SSE forwarding and `Cache-Control: no-store, no-transform`.

Back up the live Caddyfile and firewall rules before rollout, validate and reload Caddy, then open direct HTTPS. Verify the primary site's normal access, rejection of direct primary-host requests (including forged forwarding headers), direct API authentication, complete streaming responses and usage records. A shared server and network connection do not provide traffic or bandwidth isolation between these hosts.

Example host preparation:

```bash
sudo chown -R root:root /opt/sub2api/compose/postgres-init
sudo chmod 755 /opt/sub2api/compose/postgres-init
sudo chmod 755 /opt/sub2api/compose/postgres-init/10-create-app-role.sh

sudo install -d -o caddy -g caddy -m 755 /var/log/caddy
sudo install -o caddy -g caddy -m 640 /dev/null /var/log/caddy/sub2api-access.log
sudo install -o root -g root -m 644 Caddyfile.example /etc/caddy/Caddyfile
sudo caddy fmt --overwrite /etc/caddy/Caddyfile
sudo caddy validate --config /etc/caddy/Caddyfile
```

The PostgreSQL initialization script only runs when `postgres_data` is empty. Changing its environment variables later does not rotate an existing database role or password.

Resource values are deliberately conservative. PostgreSQL server settings and container limits will be added after the actual server memory and CPU are inspected.
