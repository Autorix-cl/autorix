# Autorix service SLO defaults

These are conservative **starting objectives**, not evidence of achieved production reliability. They apply to the public identity path: **Ego, Janus, and Aegis**.

| SLI | Objective | Window | Failure definition |
| --- | --- | --- | --- |
| HTTP availability | 99.9% | rolling 30 days | HTTP 5xx responses divided by all HTTP responses |

The SLI uses `autorix_http_requests_total`, emitted by the shared platform HTTP middleware with `engine`, `method`, `path`, and `status` labels. HTTP 4xx responses are deliberately excluded: authentication denials, authorization denials, and malformed client requests are expected outcomes and must not consume service-availability budget.

## Burn-rate policy

For a 99.9% objective, the error budget is 0.1%.

- **Page:** 14.4x burn in both 5m and 1h windows; this consumes roughly 2% of the monthly budget in one hour.
- **Ticket:** 6x burn in both 30m and 6h windows; this consumes roughly 5% of the monthly budget in six hours.

Low-traffic periods are excluded with a positive request-rate condition. Latency, token issuance, authorization correctness, and dependency availability have no SLO alert yet because the repository does not define a reviewed user-facing objective or a complete SLI for them. Do not treat their operational metrics as an SLO by implication.
