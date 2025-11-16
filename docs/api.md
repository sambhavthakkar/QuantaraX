# API Documentation

## Bootstrap HTTP API
- `POST /api/v1/token` Register a token
  - Body: `{ token, ephemeral_public_key, manifest_hash, relay_hints[], sender_address, ttl_seconds }`
  - Returns: `{ token, expires_at, registration_id }`
- `GET /api/v1/token/{token}` Lookup a token
  - Returns: `TokenEntry`
- `POST /api/v1/register` Register a username
  - Body: `{ username, public_key, relay_hints[], direct_address }`
  - Returns: `{ username, fingerprint, registered_at }`
- `GET /api/v1/lookup/{username}` Lookup a username
  - Returns: `UserEntry`

Future: Daemon/Relay control APIs (gRPC/REST) for transfer orchestration.
