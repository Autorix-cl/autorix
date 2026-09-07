# Contributing to Autorix

Thanks for improving Autorix. Contributions are accepted under the
[Apache-2.0 license](LICENSE).

## Before opening a pull request

1. Search existing issues and discussions before starting work.
2. Open an issue first for security-sensitive, cross-engine, or behavior-changing
   work so maintainers can agree on the problem and boundary.
3. Do not report security vulnerabilities publicly; follow [SECURITY.md](SECURITY.md).
4. Keep each pull request focused. Do not mix refactors, formatting-only changes,
   and behavior changes unless they are inseparable.

## Local checks

Run the checks relevant to your change before requesting review:

```bash
# Backend module
(cd janus && go test ./...)

# Console
(cd console && npm test && npm run build)

# Documentation
(cd docs && npm run build)
```

For changes spanning engines, run the repository checks documented in the
[README](README.md#testing). Do not commit generated local secrets, credentials,
private keys, `.env` files, coverage output, or built images.

## Pull request requirements

- Use a Conventional Commit title, for example
  `feat(janus): rotate refresh tokens` or `docs: clarify local bootstrap`.
- Explain the user-visible behavior, security impact, and validation performed.
- Add or update tests for behavior changes.
- Update English and Spanish public documentation when the user-facing contract
  changes. Spanish documentation must use neutral technical Spanish.
- Preserve backward compatibility or document the migration and rollback path.

Maintainers may request a smaller scope, additional tests, or documentation
before merging. Submitting a pull request means you license your contribution
under Apache-2.0, as described in Section 5 of [LICENSE](LICENSE).
