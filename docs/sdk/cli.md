# CLI & Direct API/gRPC Integration Guide

For languages without a dedicated native SDK (e.g. **Rust**, **C# / .NET**, **Java / Kotlin**, **PHP**, **Ruby**), Autorix provides the `autorixctl` CLI and high-throughput Protobuf gRPC / REST endpoints.

---

## 💻 1. The `autorixctl` CLI

`autorixctl` is the command-line control plane client for developers and DevOps pipelines.

### 1.1 Installation

Download the precompiled binary from the repository release artifacts or compile directly:

```bash
# Build locally
go build -o autorixctl ./cmd/autorixctl
```

### 1.2 Common CLI Commands

#### Evaluating a ReBAC Permission:
```bash
autorixctl check \
  --namespace documents \
  --object roadmap_2026 \
  --relation editor \
  --subject user:usr_alice
```

#### Minting an Engine Enrollment Token:
```bash
autorixctl tokens mint \
  --engine nexus \
  --environment production \
  --ttl 24h
```

#### Verifying the SHA-256 Merkle Audit Chain:
```bash
autorixctl audit verify
```

#### Exporting Audit Logs:
```bash
autorixctl audit export --format csv --output audit.csv
```

---

## ⚡ 2. Universal gRPC Integration (Rust, C#, Java)

All Autorix services expose standard Protobuf contracts (`api/autorix/<service>/v1/`). You can generate native clients using `protoc` or `buf`.

### 2.1 Generating gRPC Clients with Buf

```bash
# For Rust (tonic)
buf generate --template buf.gen.rust.yaml

# For C# (Grpc.Tools)
buf generate --template buf.gen.csharp.yaml

# For Java / Kotlin (grpc-java)
buf generate --template buf.gen.java.yaml
```

### 2.2 Direct gRPC `Check` Call in Rust (Tonic)

```rust
use autorix_nexus::v1::nexus_service_client::NexusServiceClient;
use autorix_nexus::v1::CheckRequest;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let mut client = NexusServiceClient::connect("http://127.0.0.1:50051").await?;

    let request = tonic::Request::new(CheckRequest {
        namespace: "documents".into(),
        object: "roadmap_2026".into(),
        relation: "viewer".into(),
        subject_id: "usr_alice".into(),
        subject_namespace: "user".into(),
        subject_relation: "".into(),
        request_context: None,
        resolution_token: "".into(),
    });

    let response = client.check(request).await?;
    println!("Allowed: {}", response.into_inner().allowed);

    Ok(())
}
```

---

## 🌐 3. Direct REST Integration (PHP, Ruby, cURL)

Any HTTP client can interact directly with the REST endpoints:

### Direct Check in PHP (`cURL`)
```php
<?php
$payload = json_encode([
    'namespace' => 'documents',
    'object' => 'roadmap_2026',
    'relation' => 'viewer',
    'subject_id' => 'usr_alice',
    'subject_namespace' => 'user'
]);

$ch = curl_init('http://localhost:8080/check');
curl_setopt($ch, CURLOPT_POSTFIELDS, $payload);
curl_setopt($ch, CURLOPT_HTTPHEADER, ['Content-Type:application/json']);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);

$result = json_decode(curl_exec($ch), true);
curl_close($ch);

if ($result['allowed']) {
    echo "Permission granted\n";
}
?>
```
