# Tasks: Timeout JSON Error

- [x] 1. Modify `platform/httpx/httpx.go` to intercept `http.TimeoutHandler` responses.
  - [x] 1.1 Create a `timeoutWriter` struct that wraps `http.ResponseWriter`.
  - [x] 1.2 Implement `WriteHeader(int)` on `timeoutWriter`. If the status is `http.StatusServiceUnavailable`, set `Content-Type: application/json`.
  - [x] 1.3 Update the `Timeout` middleware to wrap `next` with `http.TimeoutHandler(next, d, "{\"error\":\"request timed out\"}")`.
  - [x] 1.4 In the `Timeout` middleware, serve the request using the `timeoutWriter` wrapping the original `w`.
- [x] 2. Add or update tests in `platform/httpx/httpx_test.go` to verify the JSON timeout response.
