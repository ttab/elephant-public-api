# Changelog

The log starts at v0.0.11, the earliest release listed here; releases before
that are not reconstructed. The entries are derived from the release tags, and
the linked pull requests hold the detail.

## [v0.2.0] - Unreleased

**Behaviour change (distribution eventlog):** `DocumentEvent.event_type` has a
third value, `"deleted"`, for the repository's deletion of a document. It used
to be reported as `"unpublished"`, which is a different instruction to a
consumer: an unpublish stops the content being served and leaves the document,
while a deletion terminates the document's generation and is also what an
erasure expresses as, so a mirror removes its copies. Both carry a negative
version, so a consumer that derived the kind from the sign never saw the
difference and has to read `event_type` now. It is not part of the hashed event
payload, so a verifier is unaffected.

Changes:

- `DocumentEvent.nonce` is the generation the event belongs to — the
  repository's per-generation document nonce, the same value a pushed object's
  envelope carries as `document_nonce`. Group by `(doc_uuid, nonce)`: the
  highest version within a generation is that generation's state, and a
  document that comes back after a deletion arrives under a new nonce with its
  versions starting over. The nil UUID means imported history.
- `SubscriptionMatch.event_type` documents what it actually reports. Only
  stored versions are matched, so the value is always `"published"`; a consumer
  that has to react to unpublishes and deletions follows the eventlog through
  `Content.GetNewDocuments`.

## [v0.1.0] - 2026-09-06

**Breaking:** the module is Connect only. The Twirp clients, the Twirp servers
and the `/twirp/` paths are gone rather than kept beside the new surface, so
every consumer moves in one step:
`New<Service>ProtobufClient` and `New<Service>JSONClient` become
`<package>connect.New<Service>ServiceClient(httpClient, baseURL)`, which takes
the base URL of the server rather than a per-service path, and
`New<Service>Server` becomes
`<package>connect.New<Service>ServiceHandler(svc, opts...)`, which returns the
mount path together with the handler for `elephantine`'s `RegisterConnect`.
Both sides are expressed in the same plain service interface, now generated
into `service.rpc.go` with the name and the method set `protoc-gen-twirp`
declared, so an implementation of a service compiles unchanged: a server swaps
its mount line, a client its constructor. Error checks move from
`elephantine.IsTwirpErrorCode` to `rpc.IsCode(err, connect.Code…)` and error
construction to the `rpc` helpers, both from `github.com/ttab/elephantine/rpc`.
The paths become the unprefixed `/<package>.<Service>/<Method>`, so an ingress
rule that routes on `/twirp/` needs one for them. An error body is Connect's
`{"code","message","details"}` where Twirp's was `{"code","msg","meta"}`:
there is no free-form meta map, so the key/value metadata travels as an
`elephantine.rpc.ErrorMeta` detail, read with `rpc.Meta(err)` in Go and with
the Connect runtime's `findDetails` elsewhere. A service mounting these
declarations also answers gRPC and gRPC-Web on the same paths, selected by
content type, but that reach ends at the cluster: the fleet's ingress speaks
HTTP/1.1 to its targets and no externally reachable gRPC target group is
provided, so gRPC and gRPC-Web are an in-cluster calling convention and are
not offered to external callers or to browsers.

**Breaking (JSON field names):** a JSON response now spells its fields in
protojson's lowerCamelCase (`documentUuid`) where the Twirp servers spelled
them the way the `.proto` declares them (`document_uuid`). Connect's JSON
codec is protojson with its default options and a service must not install a
`UseProtoNames` codec to paper over the difference — every Connect runtime and
proxy assumes the standard encoding. Requests are unaffected, because
protojson unmarshalling accepts both spellings, so a caller can move its path
before it moves its field names. Generated clients — Go, `@protobuf-ts`,
`connect-es` — parse into the generated types and see nothing of this. The
consumer this reaches is the one that reads a response body by hand with
`fetch` or `curl`: change only the path and every multi-word field reads
`undefined`.

**Migration order for the consumers.** Seven repositories pin this module and
three of them are libraries, which fixes the order they have to move in.
`elephant-assets-client` first: it is imported by `elephant-distribution`'s
own packages, so minimal version selection drags its `elephant-public-api` pin
into distribution's build and a distribution that moves first fails to compile
in a dependency. Its `twirp.BadRoute` fallback in `fetchVariants` — the one
that lets key fetching keep working against an asset service that predates
`GetVariants` — has to become `rpc.IsCode(err, connect.CodeUnimplemented)`:
`errors.As` to a `twirp.Error` never matches a `*connect.Error`, so left as it
is the fallback stops firing and the degradation is silent. Then `distconf`.
Then `elephant-hub` and `elephant-live`, which `replace` the module to a local
checkout and therefore already fail `go build ./...` against this branch. Then
`elephant-assets`. Then `elephant-distribution`, the largest of them, with
some 750 `twirp.*` references across 47 files and a hand-rolled sequence of
`New<Service>Server` mounts in `cmd/elephant-distribution/main.go` that has to
grow a Connect arm. Then `dist-import`.

`elephant-assets` needs one thing more than a bump. It is deployed on TT stage
behind an ingress scoped to `path: /twirp/`
(`apps/elephant-assets/base/ingress.yaml` in `ttab/deploy`, the only
path-scoped RPC ingress in the fleet; the prefix exists to keep the `/v1/`
serving path CloudFront-only, so widening it to `/` is not the fix). Its
Connect mount must therefore ship with per-service Connect path rules for
`/elephant.assets.Keys/` and `/elephant.assets.Management/` in the same
change, in the base and in the stage overlay, or every Connect call returns
404 from ingress-nginx. Stage is not production — the assets clients are not
deployed there at all, and nothing that consumes these declarations runs in
production — which is why a clean Connect-only break is acceptable rather than
a dual-stack period.

**Build:** the `go` directive moves from 1.26.5 to 1.27.1, so every consumer's
own floor moves with it: a module that imports this one and declares an older
`go` directive stops building until it is raised, and a CI job that resolves
its toolchain from `go.mod` downloads 1.27.1. The floor is the toolchain the
pinned generators are run under, which is what makes the committed output
reproducible rather than a function of whatever Go is on the machine.

**New service (hub):** `elephant.hub` gains a `Client` service for the hub's
machine callers — elephant-repository instances and the elephant BFF — which
authenticate with OIDC client credentials from the nexus realm and are gated
per method by the scope claim. `RegisterGeneration` records the schema set an
instance has activated and is idempotent on the caller and the generation
number, `GetLatestGeneration` reads back the highest number that caller has
registered so an instance can resume after a restart, and `GetAssetURLs`
resolves up to a hundred artifacts to their manifests, signatures and signed
CDN asset URLs. Both generation methods require the `hub_generations` scope and
`GetAssetURLs` requires `hub_assets`. `GetAssetURLs` decides access per
artifact and reports a per-artifact `AssetURLError` — `not_found`, `forbidden`,
`revoked` or `invalid` — instead of failing the whole batch, and each resolved
`ArtifactAssetURLs` carries the raw manifest bytes, the signature over exactly
those bytes as stored at publish time, and the signing key's fingerprint,
public key and current `PublishingKeyStatus`, so a consumer verifies what it
loads rather than trusting the hub to have done it. Verify the signature
against the manifest bytes as returned, without re-serialising them.

**Removed (OpenAPI):** the OpenAPI 3 specifications under `docs/` are gone.
They described the Twirp paths and Twirp's error schema only, nobody generated
a client from them, and the generator that wrote them cannot run under buf. The
`.proto` files are the declaration a non-Go consumer generates from. With
nothing left to stamp a version into, a release is a plain git tag: there is no
`rpc:release` target and no "bump to vX.Y.Z" commit any more.

**Generators:** the module is generated by the released `ttab/mage` v0.13.1,
which pins `protoc-gen-elephant-rpc` to elephantine v0.29.0, so the generated
code in this release is reproducible from released generators.

**Generation:** the artifacts are generated with buf through the `rpc`
namespace of `ttab/mage`, which runs the compiler and every plugin as
`go run <module>@<version>` from a pin it holds, so regenerating needs no
Docker, installs nothing and takes nothing off `PATH`. `mage rpc:generate`
replaces `mage twirp:generate`, and `mage rpc:vendorProto <module> <file>`
copies a `.proto` this repository imports out of another module into
`rpc/vendor` — `newsdoc/newsdoc.proto`, which the distribution and live
declarations import, is vendored out of elephant-api that way. The copy keeps
its path, so the imports are unchanged, and the generated `buf.yaml` declares
the vendor directory as a buf module root of its own; the target is idempotent,
so CI can run it and let `git diff --exit-code` report drift. Regeneration
needs the network — every plugin is its own module and `go run` queries the
module proxy on each invocation, so `GOPROXY=off` fails even with a warm
module cache — and the targets pin the toolchain the generators run under, so
the committed output does not depend on the Go version installed on the
machine. `google.golang.org/protobuf` is v1.36.12, which is the version the
pinned `protoc-gen-go` stamps into the generated headers.

Changes:

- `github.com/twitchtv/twirp` is out of `go.mod` and out of the build, and
  `connectrpc.com/connect` v1.20.0 is in. Twirp is still visible in
  `go list -m all` because elephant-api v0.24.2, which the module depends on
  for the NewsDoc messages, still requires it in its own `go.mod`; that goes
  away when elephant-api drops Twirp. The module still does not depend on
  `elephantine` — the generated code imports only connect, `context`,
  `net/http` and the message packages. (#1)
- The wire contract does not move. The `FileDescriptorProto` each `.pb.go`
  embeds is identical to the one on `main` for every file except
  `hub/service.proto`, which carries the new `Client` service, so no existing
  message, field or method changed. The rest of the `.pb.go` diff is
  protoc-gen-go v1.36.12 run by buf where the `elephant-twirptools` image ran
  v1.36.2 under protoc: the descriptor becomes a string constant, `unsafe` is
  imported, and the header records `protoc (unknown)` because buf reports no
  protoc version. (#1)
- Generation now writes, per service directory, `service.pb.go` for the
  messages, `service.rpc.go` for the plain service interface, and under
  `<package>connect/` both `service.connect.go` — connect-go's own
  `New<Service>Client` and `New<Service>Handler`, which speak in
  `*connect.Request[T]` — and `service.elephant.go`, the adapters that put them
  on the plain interface. The `Service` infix is what tells the two apart; use
  the adapters unless a call needs per-call access to headers or trailers, for
  which `rpc.PropagateHeaders()` and `rpc.WithOutgoingHeaders` are the
  mechanism. (#1)
- Each `<package>connect` package has a test that fails to compile if a
  regeneration renames or drops an adapter, and fails if a mount path grows a
  prefix. (#1)
- `mage rpc:generate` now discovers a service declared as
  `<app>/v1/service.proto` as well as `<app>/service.proto`, and `mage
  rpc:stub` scaffolds the versioned layout. The declarations released so far
  keep the flat layout and regenerate identically; a new API can be added
  versioned without moving them. (#1)
- The README now describes the services per API, the Go client and handler
  constructors, the JSON-over-POST call, the lowerCamelCase JSON field names
  and the error body for a non-Go consumer, that gRPC and gRPC-Web are
  in-cluster only, the buf generation and what a release is, and points at
  elephantine's `docs/connect.md` for the codes and the conventions the fleet
  shares. (#1)

## [v0.0.15] - 2026-08-21

**Breaking (distribution):** erasure informs rather than withdraws, and
`Delivery.ClearDeliveryErasureObligation` is removed along with its request and
response messages. Nothing is deleted at a destination any more: an erasure
drops the document's undelivered queue rows and sends every destination that
was pushed a copy a deletion marker in the same feed the content arrived in, so
an unreachable destination's obligation is an ordinary undelivered queue row
that delivers when the destination recovers, and `DeleteDestination` is the one
write-off left. `DeliveryErasureObligation.cleared_reason` changes with it:
`withdrawn` becomes `notified`, `operator_cleared` is gone, and
`already_terminated` is new for a destination whose every generation already
had a deletion marker at the head of its chain — the ordinary case for an
erasure that follows a repository delete. `objects_outstanding` no longer falls
when an obligation clears; it stands as the record of how much content the
marker is about, and a value above `objects_at_request` is what re-opens a
discharged obligation. `cleared_by` is always empty, and `sweep_state` (field
8) is removed and its number reserved. `DeliveryOperation.kind` loses
`erasure_obligation_cleared`, and `dist_admin` on its own is no longer required
for a call that no longer exists.

**Behaviour change (erasure trigger):** deleting a document in the repository
is now the ordinary trigger for an erasure — the delete event drives the same
erasure `EraseDocument` records, so an editorial delete inside becomes an
erasure outside. The RPC is for the content that trigger cannot reach: a
document deleted before distribution ingested delete events, and content that
exists here only as imported history with no live repository document to
delete. Calling it for an already-erased document is a no-op. `complete` on
`ErasureStatus` now means that nothing we control still holds the content,
nothing more will be sent, and every destination that was given a copy has been
sent a marker — not that the copies are gone from the destinations.

**Behaviour change (erasure proof):** `GetErasureStatus` documents one window
outside the proof rather than two. An object whose write completed but whose
inventory row was never recorded is indistinguishable from a file the customer
put there themselves, so if it is the only copy a destination ever received
that destination is not on the list; the queue row survives, so the next pass
re-sends and records it. The SFTP sweep window is gone with `sweep_state`.

## [v0.0.14] - 2026-08-20

**Breaking (distribution):** erasure gains delivery obligations.
`ErasureStatus` carries `pending_deliveries`, a repeated
`DeliveryErasureObligation` naming every destination that was given a copy of
the erased document and how each one was accounted for — the transport, the
object counts at request and outstanding, `cleared_at`, `cleared_reason`
(`withdrawn`, `destination_deleted` or `operator_cleared`), whether a push
worker will currently attempt it, the SFTP `sweep_state` and
`last_success_at`. An erasure is no longer complete once the archived index
partitions are re-materialized: every destination that was given a copy has to
be accounted for as well, so a consumer that polls `GetErasureStatus` for
completion waits on delivery too. `EraseDocument` erases the extracted delivery
fields and the objects written to destinations along with the stored versions,
the archived objects and the index copies.

**Breaking (distribution):** `Delivery.ClearDeliveryErasureObligation`
discharges one destination's obligation by hand, for a destination that will
never be reachable again. It requires `dist_admin` on its own, takes a required
`reason` and an `acknowledge` that must be true, is refused for a destination a
push worker would currently be started for, and keeps the inventory rows as the
record of which objects were written off. `DeleteDestination` discharges the
destination's outstanding obligations as `destination_deleted` for the same
reason, and `DeliveryOperation.kind` gains `erasure_obligation_cleared`, which
`ListDeliveryOperations` now returns.

## [v0.0.13] - 2026-08-19

**Breaking (distribution):** the renderer contract is corrected, and
`RendererConfiguration.full_document` (field 6) is removed and its number and
name reserved. A renderer's triggers decide whether it is invoked for a
document, not which blocks belong to it: every invoked renderer is called once
with the whole document and may answer for any of its top-level blocks, so
there is nothing left to opt into. Declaration order now resolves a
disagreement rather than assigning ownership — where two renderers return a
replacement for the same block the one declared first wins and the other's
entry is dropped, and insertions are applied in the same order. A renderer
failure of any kind makes the renderer contribute nothing to that document,
every block rendering as if it had returned no entry, instead of falling back
for a claimed subset. The top-level block is the addressable unit throughout:
nested blocks carry no id, and a renderer that wants to change something inside
one renders the block it sits in.

## [v0.0.12] - 2026-08-19

Changes:

- Distribution can render delivered documents to HTML. `include_html` is new on
  `GetNewDocumentsRequest`, `GetDocumentRequest`, `GetDocumentVersionsRequest`,
  `ListPublishedVersionsRequest`, `ListPlannedVersionsRequest` and `Search`'s
  `QueryRequest`, and the corresponding `rendered_html` on `DocumentItem`,
  `GetDocumentResponse`, `PublishedVersion`, `LoadedDocumentVersion` and `Hit`.
  The HTML is an `<article>` fragment of block-level markup with no document
  frame, carrying the rendition links of the delivered document; only
  `data-block-id` is a versioned hook, and element choices and class names may
  change between releases. It follows the loaded document, so a deferred entry,
  a tombstone or a type the configuration renders no HTML for comes back
  without HTML rather than as an error, and the search surface needs
  `load_documents` or `subset` and the same `dist_read` scope they do.
- `DestinationSpec.include_html` (field 14) delivers the rendered HTML to a
  push destination, as a `rendered_html` member beside `document` in the
  envelope. It can be changed on an existing destination: it changes an
  envelope member rather than an object key, and the envelope version is not
  bumped for it, so a consumer relies on the ingest contract's obligation to
  ignore unknown members. A retraction never carries HTML.
- The configuration generation carries the rendering setup:
  `HTMLRenderingConfiguration` on `ConfigGeneration.renderers` and
  `RegisterConfigGenerationRequest.renderers` names the document types that
  render, the rendition variant the markup embeds (service-level, because the
  HTML is cached across callers) and the operator-registered renderers, each a
  `RendererConfiguration` of kind `js` or `remote` with a cache-busting
  `revision`, a `RendererPolicy` or a `policy_preset` sanitizer, and a
  `RendererCircuitBreaker`. It is validated at registration rather than at
  render time, so a generation that cannot render is never created.
- `Configuration` gains `SetRendererSecret`, `DeleteRendererSecret` and
  `ListRendererSecrets` for the shared secret a remote renderer's HMAC
  signatures are computed with. A renderer names its secret implicitly by its
  own name. The secret is write-only — there is no call that returns it, and
  `ListRendererSecrets` reports only the name, the keyring entry it is sealed
  under and when it was written — and an environment-provided secret takes
  precedence over a stored one. Store the secret before registering a
  generation that names the renderer: registration resolves every remote
  renderer's secret and refuses a generation whose secrets do not resolve.

## [v0.0.11] - 2026-08-14

Changes:

- A delivery destination can name a directory on a customer's SSH server
  instead of a bucket. `SFTPTransport` is a new arm on
  `DestinationSpec.transport` (field 13, deliberately not contiguous with `s3`
  at 3), and `SFTPCredentials` a new arm on
  `SetDestinationCredentialsRequest.credentials` (field 3); `CredentialStatus`
  reports the kind as `sftp`. Nothing existing changes number, type or meaning,
  and an older server decoding the sftp arm sees an unknown field and reports
  that the destination needs a transport.
- The username, the path and the host keys sit on the transport rather than on
  the credential, because credentials are write-only at every scope. At least
  one host key in `authorized_keys` form is required and a server presenting
  anything else is refused: there is no trust-on-first-use and no
  accept-and-log mode, and several keys may be given so that a rekeying is not
  an outage. The host must not resolve into the loopback, link-local, private,
  unique-local or carrier-grade-NAT ranges, checked both when the destination
  is written and on every connection.
- `SetDestinationCredentialsResponse.public_key` answers a credential write
  with the `authorized_keys` line of the key just stored — both the check that
  the upload parsed to the key the customer was given and the artefact they
  install. It is on the response rather than on `CredentialStatus` because it
  is derived from the plaintext in hand; a readable projection of a stored
  credential would be the oracle write-only credentials exist to avoid. An
  encrypted private key is refused rather than stored beside its passphrase.
