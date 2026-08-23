# Change: Timeout JSON Error

## Context
The `platform/httpx` package provides a `Timeout` middleware using Go's standard `http.TimeoutHandler`. When a request exceeds the deadline, the standard library handler writes a plain text `"request timed out"` message to the response body with a 503 Service Unavailable status. Since Autorix engines exclusively expose JSON APIs, this text response breaks JSON parsers on client applications, resulting in unhandled parsing exceptions rather than proper error handling.

## Objective
Update the `Timeout` middleware to return a properly formatted JSON error envelope when a request times out, ensuring consistent API responses across all Autorix engines.

## Scope
- **In Scope**: Modifying the `Timeout` middleware in `platform/httpx/httpx.go` to intercept or replace the default text response with a JSON response.
- **Out of Scope**: Changing the timeout durations for individual routes or modifying the global error format schema.

## Capabilities
1. **platform-middleware**: The Timeout middleware must return a standard JSON error envelope with a 503 status code when a request exceeds its deadline.
