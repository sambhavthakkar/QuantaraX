# API Examples (curl)

Base URL: http://127.0.0.1:8080
Optional auth: export QUANTARAX_AUTH_TOKEN and pass header: -H "X-Auth-Token: $QUANTARAX_AUTH_TOKEN"

Create transfer:
curl -s -X POST http://127.0.0.1:8080/api/v1/transfer/create \
  -H 'Content-Type: application/json' \
  -d '{"file_path":"/path/to/file.bin","recipient_id":"peer-123"}'

Accept transfer:
curl -s -X POST http://127.0.0.1:8080/api/v1/transfer/accept \
  -H 'Content-Type: application/json' \
  -d '{"transfer_token":"<token>","output_path":"/tmp/out.bin"}'

Get transfer status:
curl -s http://127.0.0.1:8080/api/v1/transfer/<session_id>/status

List transfers:
curl -s 'http://127.0.0.1:8080/api/v1/transfers?state=&limit=20&offset=0'

Get keys:
curl -s http://127.0.0.1:8080/api/v1/keys

SSE events (shows live events):
curl -s http://127.0.0.1:8080/api/v1/events | while read -r line; do echo "$line"; done
