# Spec: platform-middleware

## Feature: JSON-Formatted Timeout Responses

### Scenario: Middleware returns JSON on timeout
- **Given** an HTTP request routed through the `Timeout` middleware
- **When** the downstream handler takes longer than the configured timeout duration
- **Then** the client receives a `503 Service Unavailable` status code
- **And** the `Content-Type` header is set to `application/json`
- **And** the response body is a valid JSON error envelope containing a timeout message
