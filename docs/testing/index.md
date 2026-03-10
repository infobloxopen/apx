# Testing

APX uses a multi-layer testing strategy: unit tests (`go test`), CLI integration tests (testscript), and full end-to-end tests against a real Gitea instance running in k3d.

<div class="grid cards" markdown>

-   :material-test-tube: **E2E Tests**

    ---

    Run the full release workflow against a real git server with k3d + Gitea.

    [:octicons-arrow-right-24: Run E2E tests](e2e-tests.md)

-   :material-format-list-checks: **Format Maturity**

    ---

    Support level and test coverage for each schema format (Proto, OpenAPI, Avro, JSON Schema, Parquet).

    [:octicons-arrow-right-24: View matrix](format-maturity.md)

</div>
