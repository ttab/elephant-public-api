# Elephant Public API

Public API declarations for the Elephant platform. Each service is declared in
a `.proto` file and shipped with generated Go code for
[Connect](https://connectrpc.com/). There is no Twirp: these services are
Connect only, and nothing here is served on a `/twirp/` path.

Building the generated code needs Go 1.27.1 or later.

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

Generation needs the network. Every plugin runs as its own module and `go run`
queries the module proxy on each invocation, so `GOPROXY=off` fails even with
a warm module cache. The targets pin the toolchain the generators run under
and drop a `-mod` flag from `GOFLAGS`, so the output does not depend on the Go
version or the module mode that happens to be configured on the machine.

| Target | Purpose |
| --- | --- |
| `mage rpc:generate` | Regenerate the Go and Connect artifacts for every service. |
| `mage rpc:stub <app> <service> <method>` | Scaffold a new service proto, in the versioned layout `<app>/v1/service.proto`. |
| `mage rpc:vendorProto <module> <file>` | Copy a `.proto` file this repository imports out of another module and into `rpc/vendor`. |

Per service directory, generation writes `service.pb.go` (the messages),
`service.rpc.go` (the plain service interface) and, under
`<package>connect/`, `service.connect.go` (connect-go's client and handler)
plus `service.elephant.go` (the adapters that put them on the plain interface).
A `.proto` file that declares no service is compiled to messages and nothing
else, and nothing but Go is generated.

The existing declarations sit one directory per API (`assets/service.proto`);
`mage rpc:generate` discovers both that layout and the versioned
`<app>/v1/service.proto` one, so a new API can be scaffolded versioned without
moving the ones that are already released.

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

The generators this module is built with are released: `ttab/mage` v0.13.1
runs buf and the plugins at pinned versions and pins `protoc-gen-elephant-rpc`
to elephantine v0.29.0, so the generated code in a tag is reproducible from
released code. Keep it that way: a `ttab/mage` requirement that is a
pseudo-version of a branch is a reason not to tag, since the branch can be
force-pushed out from under the tag.

## License

Licensed under MIT.
