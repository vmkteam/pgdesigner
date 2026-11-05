# Contributing to PgDesigner

Thank you for your interest in contributing to PgDesigner!

## License

PgDesigner is licensed under the [Apache License 2.0](LICENSE). By submitting a pull request, you agree that your contribution is licensed under the same terms (Section 5 of the License).

## Developer Certificate of Origin (DCO)

Sign off every commit to certify that you wrote the code or have the right to submit it under the project's license ([developercertificate.org](https://developercertificate.org/)):

```bash
git commit -s -m "Fix FK rename in diff"
```

This adds a `Signed-off-by: Your Name <you@example.com>` line to the commit message.

## How to Contribute

1. Fork the repository and create a branch for your changes.
2. Make your changes and ensure tests pass (`make test`).
3. Submit a pull request with a clear description of what you changed and why.

## Code Style

- Go: simple, clear, no magic. Prefer stdlib over third-party. Error handling with context wrapping.
- Commit messages: concise, English.

## Questions?

If you have any questions about contributing, feel free to open an issue or a [discussion](https://github.com/vmkteam/pgdesigner/discussions), or contact us at [support@pgdesigner.io](mailto:support@pgdesigner.io).
