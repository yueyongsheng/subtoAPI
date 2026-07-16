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
3. Replace all image tags with tested immutable digests after the first pull.
4. Create `data`, `postgres_data`, and `redis_data` with the ownership required by the images.
5. Replace the Caddy domain and install it as the host Caddy configuration.
6. Validate with `docker compose config` before starting containers.

The PostgreSQL initialization script only runs when `postgres_data` is empty. Changing its environment variables later does not rotate an existing database role or password.

Resource values are deliberately conservative. PostgreSQL server settings and container limits will be added after the actual server memory and CPU are inspected.
