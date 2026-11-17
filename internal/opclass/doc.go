// Package opclass assigns an operation class to each operation of an OpenAPI
// document. The class is one of read, list, write or custom, and it is recorded
// in the document as the extension x-operation-class.
//
// The classifier is tolerant. It is separate from the strict x-aip-method
// extension, which marks an operation only when its request and response have
// the exact AIP shape. Client generators read x-aip-method and then expect that
// shape, so this package never writes or changes x-aip-method.
//
// # Rules
//
// An explicit x-operation-class on the operation always wins. Its value must be
// read, list, write or custom; any other value is an error. Without it, the
// rules apply in this order and the first match decides:
//
//  1. AIP method (RuleAIP). x-aip-method is present and is not Custom. Get and
//     BatchGet give read. List and Search give list. Create, Update, Delete,
//     Undelete, Purge, BatchCreate, BatchUpdate and BatchDelete give write.
//     Custom and every unknown value fall through to the next rule. The match
//     ignores case.
//
//  2. Name (RuleName). The operationId starts or ends with a verb word. List
//     gives list. Read and Get give read. Create, Update, Patch and Delete give
//     write. See "Tokenization" below.
//
//  3. Verb and path shape (RuleShape). GET or HEAD on a path whose last segment
//     is one variable ("/users/{id}") gives read. GET or HEAD on any other
//     non-root path gives list, except a path with an AIP custom verb
//     ("/users/{id}:export"), which falls through. POST, PUT, PATCH and DELETE
//     give write.
//
//  4. Otherwise custom (RuleDefault).
//
// # Tokenization
//
// Rule 2 splits the operationId into words. A word boundary is any character
// that is not a letter or a digit, a lower-to-upper case change ("getUser"),
// and the end of an acronym ("HTTPServer" is HTTP, Server). A digit stays with
// the word before it. The comparison of a word with a verb ignores case.
//
// The published protobuf form Service_Method is recognized when the text after
// the last underscore starts with an upper-case letter. Only the method part is
// used then, and leading method words that repeat the leading service words are
// dropped (a plural s, es or ies on the service word is tolerated). So
// "Accounts_AccountListUsers" is read as List, Users.
//
// The first word decides when it is a verb; otherwise the last word decides.
// One exception: a first word Get or Read with a last word List gives list
// ("GetUserList").
//
// Only whole words match. "Blacklist" and "Checklist" are one word each and do
// not match List. "blackList" is two words and does match; an explicit
// x-operation-class corrects such a name. A name such as "getDashboards" gives
// read although the operation returns a collection; the name rule cannot see
// the plural. Set x-operation-class: list on that operation.
//
// # Streaming
//
// An operation streams when a response declares the content type
// text/event-stream or application/x-ndjson (media type parameters are
// ignored), when a Swagger 2 "produces" list holds one of them, or when the
// operation carries x-streaming: true.
//
// # Stable API
//
// The exported API is small on purpose; other commands build on it:
//
//   - Class, the constants Read, List, Write, Custom, Classes and ParseClass.
//   - Rule and its constants, which name the rule that decided.
//   - Input and Classify, for one operation.
//   - Operation, Document, Parse and Load, for a whole document. Operations are
//     sorted by path, then by method. ErrNoPaths marks a document without a
//     "paths" object.
//   - Annotate, which adds x-operation-class to the operations that lack it.
//   - ExtensionClass, ExtensionAIPMethod and ExtensionStreaming, the extension
//     names.
package opclass
