### Java Development Loop

1. Add `<dependency>` to `pom.xml` with derived Maven coordinates
2. `mvn generate-sources` — generate Java code from schema
3. Import `com.{org}.apis.{domain}.{name}.{line}.*` in Java code
4. Local development via `mvn install` to `~/.m2` repository
