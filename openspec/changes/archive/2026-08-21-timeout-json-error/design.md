# Design: Timeout JSON Error

## Approach
The `Timeout` middleware in `platform/httpx/httpx.go` currently uses `http.TimeoutHandler(next, d, "request timed out")`. The third argument is a string that `TimeoutHandler` writes as plain text when the timeout fires.
Unfortunately, `http.TimeoutHandler` in the standard library doesn't easily allow setting JSON headers or providing a custom response writer directly for the timeout case. However, it does allow passing a custom string which acts as the body. Wait, `http.TimeoutHandler` allows passing `Option` functions in newer Go versions (like `WithOption`), but in Go standard library `TimeoutHandler` accepts a `msg` string. Wait, no. `http.TimeoutHandler` accepts a string `msg`. If you want a custom JSON, there is a trick. `http.TimeoutHandler(next, d, "{\"error\":\"request timed out\"}")` would output JSON, but the `Content-Type` would remain `text/plain` because `TimeoutHandler` sets it to text/plain (or rather, leaves it blank and `http` sniffing might decide, or it explicitly writes `text/plain`!).

Let's check `http.TimeoutHandler` options in Go 1.25. Actually, since Go 1.20, there is `http.TimeoutHandler` but also `http.TimeoutHandler(..., ...)` doesn't let you set headers.
Wait! `http.TimeoutHandler` sets `Content-Type: text/plain` by default if it's returning the default message, but what if we write a custom `http.Handler` for timeouts? Or we can wrap `TimeoutHandler` with a custom `ResponseWriter`? No, if we wrap `TimeoutHandler` with a custom `ResponseWriter` that intercepts the 503 and overwrites headers, we can set `Content-Type: application/json` and output the JSON body.

Actually, the simplest way is to wrap `next` in our own `Timeout` logic utilizing `context.WithTimeout`. But `http.TimeoutHandler` does exactly that securely. We can just use `http.TimeoutHandler` and pass `jsonMessage`. But wait, `http.TimeoutHandler` accepts a `msg string`.

Wait, in Go `http.TimeoutHandler(next, dt, msg)`: if we want to change headers, we can use `http.TimeoutOption` (Wait, does `TimeoutOption` exist? No). 

Let's implement a custom `Timeout` middleware:
Instead of `http.TimeoutHandler`, we can create a handler that uses `context.WithTimeout`. Or better, we can intercept the response from `http.TimeoutHandler` by wrapping the `ResponseWriter`. If `http.TimeoutHandler` calls `WriteHeader(503)` and the body is the timeout message, we intercept it, change the header to `application/json`, and write the JSON envelope.

Alternatively, `TimeoutHandler` has `http.HandlerFunc(func(w, r) ...)` logic internally.
Wait, let's see how `http.TimeoutHandler` is implemented or what's the cleanest way:
```go
func TimeoutJSON(d time.Duration) Middleware {
    // We can write a custom timeout handler that replicates http.TimeoutHandler
    // but sets Content-Type to application/json.
}
```
Actually, we can just intercept `WriteHeader` in a custom `ResponseWriter`. If status == 503 and we are in the `TimeoutHandler`... wait, `TimeoutHandler` sets headers before writing the body. 
To keep it robust and follow the current approach, we can define a custom `timeoutHandler` or wrap the ResponseWriter. Let's see the simplest robust approach: write a custom `ResponseWriter` wrapper.

Let's just implement a custom timeout handler using `http.TimeoutHandler` wrapped by an interceptor, or reimplement the timeout logic securely. Reimplementing is risky. Let's wrap `http.TimeoutHandler`.
If we wrap `http.TimeoutHandler`:
```go
func Timeout(d time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		// Use standard TimeoutHandler, but with a JSON body string
		jsonBody := `{"error":"request timed out"}`
		th := http.TimeoutHandler(next, d, jsonBody)
		
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap w to intercept WriteHeader(503) and force Content-Type
			th.ServeHTTP(&timeoutWriter{w}, r)
		})
	}
}
```
Wait, `TimeoutHandler` also supports `http.ErrHandlerTimeout` (in Go 1.20+ maybe?). 
Let's check `http.TimeoutHandler` in Go documentation.

## Decisions
- **Why wrap `http.TimeoutHandler`?**
  Writing a robust concurrency-safe timeout handler from scratch is notoriously difficult (managing panics, hijacked connections, etc.). The standard library's `http.TimeoutHandler` handles these edge cases perfectly.
  By wrapping it, we can provide a JSON string as the `msg` and use an intercepting `ResponseWriter` to overwrite the `Content-Type` header to `application/json` when a 503 is returned.
