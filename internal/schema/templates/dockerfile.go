package templates

import "fmt"

// GenerateCatalogDockerfile generates a scratch-based Dockerfile for building
// the catalog OCI image with best-practice OCI labels. Build args are used so
// the CI workflow can inject dynamic values (commit SHA, timestamp, version).
func GenerateCatalogDockerfile(org string) string {
	return fmt.Sprintf(`ARG CREATED
ARG REVISION
ARG SOURCE
ARG VERSION

FROM scratch

LABEL org.opencontainers.image.title="API Catalog" \
      org.opencontainers.image.description="APX API catalog data for discovery and search" \
      org.opencontainers.image.source="${SOURCE}" \
      org.opencontainers.image.created="${CREATED}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.vendor="%s" \
      dev.apx.type="catalog"

COPY catalog.yaml /catalog.yaml
`, org)
}
