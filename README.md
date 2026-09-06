# Elephant Public API

Public API declarations for the Elephant platform. Each service is declared in
a `.proto` file and shipped with generated Go code for
[Connect](https://connectrpc.com/), which also serves gRPC and gRPC-Web. There
is no Twirp: these services are Connect only, and nothing here is served on a
`/twirp/` path.

## The APIs

| API | Package | Services | Declaration |
| --- | --- | --- | --- |
| **Assets** | `elephant.assets` | `Keys` serves the asset CDN signing keys and the rendition variants to URL-minting services; `Management` exposes the take-down operations. | [proto](assets/service.proto) |
| **Distribution** | `elephant.distribution` | POC/work in progress. `Configuration` and `Content` hold the configuration generations and the document feed, `Archive` the bulk export, `Delivery` the destinations, `Search` the query surface and `Subscriptions` the matching rules. | [service](distribution/service.proto), [delivery](distribution/delivery.proto), [search](distribution/search.proto), [subscriptions](distribution/subscriptions.proto) |
| **Hub** | `elephant.hub` | `Hub` administers organisations, artifacts and publishing keys in the Elephant Hub registry; `Client` is the machine surface repository instances and the BFF call. | [proto](hub/service.proto) |
| **Live** | `elephant.live` | Liveblog administration, implemented by [elephant-live](https://github.com/ttab/elephant-live): `Blogs`, `Posts` and `Configuration`. | [proto](live/service.proto) |

The `distribution` and `live` declarations carry NewsDoc documents, so they
import `newsdoc/newsdoc.proto` from
[elephant-api](https://github.com/ttab/elephant-api), and their Go code uses
the NewsDoc messages from that module.

## Using the APIs

### Go

```bash
go get github.com/ttab/elephant-public-api@latest
```

Each service package holds the messages and the plain service interface, which
is the contract the protocol is expressed in:

```go
GetBlog(ctx context.Context, req *live.GetBlogRequest) (*live.GetBlogResponse, error)
```

The Connect clients live in a `<package>connect` subpackage and return that
interface, so a client and an implementation are interchangeable:

```go
import (
	"github.com/ttab/elephant-public-api/live"
	"github.com/ttab/elephant-public-api/live/liveconnect"
)

// client is an *http.Client that carries the bearer token, usually one built
// by oauth2.NewClient.
var blogs live.Blogs = liveconnect.NewBlogsServiceClient(
	client, "https://live.api.tt.se")
```

`New<Service>ServiceClient` takes the base URL of the server, not a per-service
path. The constructors, one pair per service:

| Package | Services |
| --- | --- |
| `assets/assetsconnect` | `Keys`, `Management` |
| `distribution/distributionconnect` | `Archive`, `Configuration`, `Content`, `Delivery`, `Search`, `Subscriptions` |
| `hub/hubconnect` | `Client`, `Hub` |
| `live/liveconnect` | `Blogs`, `Posts`, `Configuration` |

`New<Service>ServiceHandler(svc, opts...)` is the server side. It takes an
implementation of the plain interface and returns the mount path together with
the handler, which is the pair `elephantine`'s API server registers with
`RegisterConnect`.

The same packages also carry connect-go's own generated `New<Service>Client`
and `New<Service>Handler`, which speak in `*connect.Request[T]` and
`*connect.Response[T]`. The `Service` infix is what distinguishes the plain
adapters from them. Use the adapters unless a call needs per-call access to
headers or trailers.

Per-call request headers are a context value and one client interceptor,
`rpc.PropagateHeaders()` and `rpc.WithOutgoingHeaders(ctx, header)` from
[`elephantine/rpc`](https://github.com/ttab/elephantine).

### Other languages

Connect speaks JSON and protobuf over HTTP `POST` on the unprefixed
`/<package>.<Service>/<Method>` paths:

```
POST /elephant.live.Blogs/GetBlog
Content-Type: application/json
Authorization: Bearer …

{"id": "…"}
```

The content types are `application/json` and `application/proto`, and the
services that mount these declarations serve gRPC and gRPC-Web on the same
paths, selected by content type. Connect clients send a
`Connect-Protocol-Version: 1` header and may send `Connect-Timeout-Ms`, which
becomes the handler's deadline; the servers require neither, so a plain `curl`
or `fetch` works. Connect has runtimes for TypeScript, Swift, Kotlin and
others, all generating from the `.proto` file.

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
`context`, `net/http` and the message packages. The error helpers, header
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

| Target | Purpose |
| --- | --- |
| `mage rpc:generate` | Regenerate the Go and Connect artifacts for every service. |
| `mage rpc:stub <app> <service> <method>` | Scaffold a new service proto. |
| `mage rpc:vendorProto <module> <file>` | Copy a `.proto` file this repository imports out of another module and into `rpc/vendor`. |

Per service directory, generation writes `service.pb.go` (the messages),
`service.rpc.go` (the plain service interface) and, under
`<package>connect/`, `service.connect.go` (connect-go's client and handler)
plus `service.elephant.go` (the adapters that put them on the plain interface).
A `.proto` file that declares no service is compiled to messages and nothing
else, and nothing but Go is generated.

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
there is no "bump to vX.Y.Z" commit. From a clean tree on `main` where
`mage rpc:generate` produces no diff:

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

## License

Licensed under MIT.
