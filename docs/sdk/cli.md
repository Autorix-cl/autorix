# CLI and direct HTTP/gRPC integration

`autorixctl` is a small control-plane CLI located in `cmd/autorixctl`. For languages without a supported SDK, use the service REST/gRPC contracts directly and own authentication, timeouts, retries, and error handling in your client.

## Build from source

```bash
go build -o autorixctl ./cmd/autorixctl
```

## Implemented CLI commands

```bash
# Evaluate a Nexus relationship check.
autorixctl check --ns documents --obj document-42 --rel viewer --subj user-7

# Mint an Argus enrollment token.
autorixctl token mint --engine nexus --env production --description "nexus worker"

# Query or verify the Argus audit trail.
autorixctl audit list
autorixctl audit verify
```

The command names and flags above are the current interface. `tokens mint` (plural), `--namespace`, `--object`, `--relation`, `--subject`, and `audit export` are not implemented commands.

## Direct REST

A direct Nexus check sends JSON to `POST /check` on the Nexus URL:

```bash
curl --fail-with-body \
  --request POST http://localhost:8080/check \
  --header 'Content-Type: application/json' \
  --data '{
    "namespace": "documents",
    "object": "document-42",
    "relation": "viewer",
    "subject_id": "user-7",
    "subject_namespace": "user"
  }'
```

Do not treat a network or HTTP error as authorization. Fail the protected request closed and record the failure.

## Direct gRPC

Protobuf definitions are available beneath `api/autorix/<service>/v1/`. Generate clients with your own supported `protoc` or Buf toolchain after selecting the service contract you need. Autorix does not ship or maintain generated Rust, .NET, Java, PHP, Ruby, or other language SDKs today.
