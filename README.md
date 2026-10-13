# Elephant Public API

Public API declarations for the Elephant platform. Each service is declared in
a `.proto` file and shipped with generated Go code for
[Connect](https://connectrpc.com/). There is no Twirp: these services are
Connect only, and nothing here is served on a `/twirp/` path.

Building the generated code needs Go 1.27.1 or later.

## The APIs

| API | Package | Services | Declaration |
| --- | --- | --- | --- |
| **Assets** | `elephant.assets.v1` | `KeysService` serves the asset CDN signing keys and the rendition variants to URL-minting services; `ManagementService` exposes the take-down operations. | [proto](elephant/assets/v1/service.proto) |
| **Distribution** | `elephant.distribution.v1` | POC/work in progress. `ConfigurationService` and `ContentService` hold the configuration generations and the document feed, `ArchiveService` the bulk export, `DeliveryService` the destinations, `SearchService` the query surface and `SubscriptionsService` the matching rules. | [service](elephant/distribution/v1/service.proto), [delivery](elephant/distribution/v1/delivery.proto), [search](elephant/distribution/v1/search.proto), [subscriptions](elephant/distribution/v1/subscriptions.proto) |
| **Hub** | `elephant.hub.v1` | `HubService` administers organisations, artifacts and publishing keys in the Elephant Hub registry; `ClientService` is the machine surface repository instances and the BFF call. | [proto](elephant/hub/v1/service.proto) |
| **Live** | `elephant.live.v1` | Liveblog administration, implemented by [elephant-live](https://github.com/ttab/elephant-live): `BlogsService`, `PostsService` and `ConfigurationService`. | [proto](elephant/live/v1/service.proto) |

The `distribution` and `live` declarations carry NewsDoc documents, so they
import `newsdoc/newsdoc.proto` from
[elephant-api](https://github.com/ttab/elephant-api), and their Go code uses
the NewsDoc messages from that module.

## Using the APIs

### Go

```bash
go get github.com/ttab/elephant-public-api@latest
```

Each API is a Go package named after its protobuf package,
`elephant/<api>/v1` imported as `<api>v1`, and holds the messages. The Connect
clients and handlers are connect-go's own generated code, in a
`<api>v1connect` subpackage:

```go
import (
	"connectrpc.com/connect"
	livev1 "github.com/ttab/elephant-public-api/elephant/live/v1"
	"github.com/ttab/elephant-public-api/elephant/live/v1/livev1connect"
)

// client is an *http.Client that carries the bearer token, usually one built
// by oauth2.NewClient.
blogs := livev1connect.NewBlogsServiceClient(client, "https://live.api.tt.se")

res, err := blogs.GetBlog(ctx, connect.NewRequest(&livev1.GetBlogRequest{
	Id: id,
}))
```

`New<Service>Client` takes the base URL of the server, not a per-service path.
The methods take a `*connect.Request[T]` and return a `*connect.Response[T]`,
which is where a call's headers and trailers are read and written. The
services, one client and handler pair each:

| Package | Services |
| --- | --- |
| `elephant/assets/v1/assetsv1connect` | `KeysService`, `ManagementService` |
| `elephant/distribution/v1/distributionv1connect` | `ArchiveService`, `ConfigurationService`, `ContentService`, `DeliveryService`, `SearchService`, `SubscriptionsService` |
| `elephant/hub/v1/hubv1connect` | `ClientService`, `HubService` |
| `elephant/live/v1/livev1connect` | `BlogsService`, `PostsService`, `ConfigurationService` |

`New<Service>Handler(svc, opts...)` is the server side. It takes an
implementation of the `<Service>Handler` interface and returns the mount path
together with the handler, which is the pair `elephantine`'s API server
registers with `RegisterConnect`. An implementation embeds
`Unimplemented<Service>Handler` to answer `unimplemented` for the methods it
does not have yet.

Request headers that should follow a call from context rather than be set on
each request are propagated by one client interceptor,
`rpc.PropagateHeaders()` together with `rpc.WithOutgoingHeaders(ctx, header)`
from [`elephantine/rpc`](https://github.com/ttab/elephantine).

### Other languages

Connect speaks JSON and protobuf over HTTP `POST` on the unprefixed
`/<package>.<Service>/<Method>` paths:

```
POST /elephant.live.v1.BlogsService/GetBlog
Content-Type: application/json
Authorization: Bearer …

{"id": "…"}
```

The content types are `application/json` and `application/proto`. Connect
clients send a `Connect-Protocol-Version: 1` header and may send
`Connect-Timeout-Ms`, which becomes the handler's deadline; the servers
require neither, so a plain `curl` or `fetch` works. Connect has runtimes for
TypeScript, Swift, Kotlin and others, all generating from the `.proto` file.

A JSON response spells its fields in protojson's lowerCamelCase
(`documentUuid`), not in the names the `.proto` declares (`document_uuid`).
That is standard Connect and the services that mount these declarations do not
deviate from it, so a consumer that reads a response by hand — with `fetch` or
`curl` rather than through a generated client — reads the camelCase spelling.
Requests are unaffected: protojson unmarshalling accepts both spellings, so a
caller may send either. Unpopulated fields are omitted from a response.

The services that mount these declarations also answer gRPC and gRPC-Web on
the same paths, selected by content type, but only from inside the cluster:
the fleet's ingress speaks HTTP/1.1 to its targets and no externally reachable
gRPC target group is provided, so an external caller uses Connect over HTTP
`POST`. gRPC-Web is not a browser protocol here either — its errors arrive as
trailers that a browser cannot read cross-origin, and a browser client uses
Connect, which is what `@connectrpc/connect-web` speaks.

There is no OpenAPI specification: the `.proto` file is the declaration a
non-Go consumer generates its client from.

### Errors

An error body is Connect's:

```json
{
  "code": "not_found",
  "message": "no such blog",
  "details": [{"type": "elephantine.rpc.ErrorMeta", "value": "<base64 Any>"}]
}
```

Connect has no free-form metadata map in the body, so the key/value metadata a
service attaches to an error — `required_any_of_scopes`, `argument` and the
rest — travels as an `elephantine.rpc.ErrorMeta` error detail. Go callers read
it with `rpc.Meta(err)` and check codes with
`rpc.IsCode(err, connect.CodeNotFound)`, both from
[`elephantine/rpc`](https://github.com/ttab/elephantine); other clients read
the detail with their Connect runtime's `findDetails`. The 16 codes, the HTTP
statuses they map to, and the rest of the fleet's Connect conventions are
described in
[elephantine's `docs/connect.md`](https://github.com/ttab/elephantine/blob/main/docs/connect.md).

This module declares the messages and nothing else: it does not depend on
`elephantine`, so the generated code imports only `connectrpc.com/connect`,
the standard library and the message packages. The error helpers, header
propagation and interceptors come from `elephantine/rpc`, and a service
imports both.

## Working in this repo

The generated code is produced through the `rpc` targets from
[`ttab/mage`](https://github.com/ttab/mage). The compiler is
[buf](https://buf.build/) and every plugin is pinned there and run as
`go run <module>@<version>` — there is no Docker image, nothing is installed
and nothing is taken off `PATH`. A generator version moves when `ttab/mage` is
bumped, and the regenerated files show up in the bump's diff. Run the targets
from the repository root.

Generation needs the network. Every plugin runs as its own module and `go run`
queries the module proxy on each invocation, so `GOPROXY=off` fails even with
a warm module cache. The targets pin the toolchain the generators run under
and drop a `-mod` flag from `GOFLAGS`, so the output does not depend on the Go
version or the module mode that happens to be configured on the machine.

| Target | Purpose |
| --- | --- |
| `mage rpc:generate` | Regenerate the Go and Connect artifacts for every service. |
| `mage rpc:lint` | Check the declarations against buf's `STANDARD` lint rules. |
| `mage rpc:format` / `mage rpc:formatCheck` | Rewrite the declarations in buf's formatting, or report what it would rewrite. |
| `mage rpc:breaking` | Check the declarations for breaking changes against `main`. |
| `mage rpc:stub <app> <service> <method>` | Scaffold a new service in `elephant/<app>/v1/service.proto`. |
| `mage rpc:vendorProto <module> <file>` | Copy a `.proto` file this repository imports out of another module and into `rpc/vendor`. |

Every declaration is in the versioned layout, `elephant/<api>/v1/`, with the
protobuf package `elephant.<api>.v1` that the directory mirrors; buf checks
one against the other. An API's declaration can be split over several files,
and every `.proto` in the directory is generated for. Per file, generation
writes `<file>.pb.go` (the messages) and, for a file that declares a service,
`<api>v1connect/<file>.connect.go` (connect-go's client, handler interface and
handler). That is all: there are no adapters onto a plain service interface
and no Twirp, and a service may declare streaming methods. The declarations
pass the `STANDARD` lint rules with no exemptions, so a service name ends in
`Service` and an enum's zero value is its `_UNSPECIFIED` member.

buf compiles what is in its workspace and a workspace cannot reach outside the
repository, so a `.proto` imported from another module is vendored:

```bash
mage rpc:vendorProto github.com/ttab/elephant-api newsdoc/newsdoc.proto
```

The copy keeps the path it has in the module it came from, so the `import` in
the service's own `.proto` is unchanged, and it is compiled but never generated
for — its Go code comes from that module. The vendor directory is a buf module
root of its own, which is what the generated `buf.yaml` declares. The target is
idempotent, so rerunning it after bumping `elephant-api` reports drift rather
than producing it.

To change an API, edit its `.proto`, regenerate, and commit the proto together
with the regenerated files.

## Releasing

A release is a git tag and nothing else. Nothing is stamped with a version, so
there is no "bump to vX.Y.Z" commit. What changed in each release is in
[CHANGELOG.md](CHANGELOG.md). From a clean tree on `main` where
`mage rpc:generate` produces no diff:

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

The generators this module is built with are released: `ttab/mage` v0.15.0
runs buf, `protoc-gen-go` and `protoc-gen-connect-go` at pinned versions, so
the generated code in a tag is reproducible from released code. Keep it that way: a `ttab/mage` requirement that is a
pseudo-version of a branch is a reason not to tag, since the branch can be
force-pushed out from under the tag.

## License

Licensed under MIT.
