// Package optable renders the operation table of each API that an AppContract
// binds to routes, and the route table of the routes that the binding targets.
//
// # Inputs
//
// An AppContract (apiVersion appkit.infoblox.dev/v1alpha1 or
// appkit.platform.infoblox.com/v1alpha1) lists APIs in spec.provides.apis. The
// route binding is the optional routes object of an entry. The ref of the
// entry, apx://<api-id>@<version>, pins one release of one API; routes names
// the routes that serve it. An entry without routes is skipped. The OpenAPI
// document of each API gives the operations; the package opclass reads it.
//
// # External path
//
// The external path template of an operation is the path that the client
// sends: the externalPrefix of the routes object when it is set, otherwise the
// base path of the document, joined with the path of the operation. JoinPath
// normalizes the result: single slashes, no trailing slash except for the
// root, and "{name}" segments kept as written. A segment is a variable only
// when the whole segment is "{name}".
//
// # Precedence
//
// More than one API can be bound to one route. For each route the package
// selects one API for each method and path:
//
//  1. A literal segment wins over a variable segment. Two templates with a
//     different shape are both kept, because the matcher prefers the literal
//     segment, segment by segment from the left.
//  2. For one method and one path shape, a public API wins over an internal
//     API when the operationId is the same.
//  3. Any other overlap is a conflict: two APIs with the same audience, or one
//     path with different operationId values.
//
// # Outputs
//
// Table is the operation table of one API (schema
// apx.infoblox.com/operation-table/v1). RouteTable is the resolved table of
// all routes (schema traefik.infoblox.com/route-table/v1). Both encode to
// deterministic bytes: the same inputs give the same files.
package optable
