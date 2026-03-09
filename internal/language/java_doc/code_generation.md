### Java

Java APIs are published as Maven artifacts using conventional groupId and
artifactId derivation from the API identity.

Maven coordinates are derived as:
- groupId: `com.{org}.apis` (hyphens become dots)
- artifactId: `{domain}-{name}-{line}-proto`

**Key characteristics:**
- Full Maven coordinate: `com.{org}.apis:{domain}-{name}-{line}-proto`
- Java package: `com.{org}.apis.{domain}.{name}.{line}`
- Consumer adds `<dependency>` to `pom.xml` with the derived coordinates
- Local dev via `mvn install` to local `~/.m2` repository
- Requires `org` in `apx.yaml` for Maven coordinate derivation
