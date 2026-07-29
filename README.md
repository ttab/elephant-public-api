# Elephant Public API

Public Elephant API declarations.

* Assets API
* Distribution API: POC/Work in progress
* Hub API
* Live API: liveblog administration, implemented by
  [elephant-live](https://github.com/ttab/elephant-live)

Re-generate Go code and OpenAPI spec using: `mage twirp:generate`

That stamps the specs with the version of the last ancestor git tag. Use `mage
twirp:release <version>` to stamp them with the version you're about to tag.
