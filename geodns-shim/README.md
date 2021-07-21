geodns-shim: DNS server which localizes client queries
=====

## Overview

geodns-shim is a minimal DNS server which receives queries, adds location
information to them and forwards them to a custom backend. The reply from
the backend is then rewritten and sent back to the client, making the process
transparent to the client.

Here is a simple use case: A website has multiple hosts around the globe but
only one URL (e.g. www.example.org). Clients should be directed to the closest
instance. The DNS request for www.example.org is sent to a geodns-shim instance
which will evaluate the client IP address using the GeoIP database. According
to the configuration the request will be rewritten (e.g. from www to www-de)
and the modified query will be send to a backend DNS which serves different
IP address for www-de, www-ca, www-hk, etc.

It's basically a hack for when your DNS provider does not provide Geo-DNS
capabilities or you want to experiment with geo-located services.

## Configuration

All settings are defined via the command line parameters. Run `geodns-shim`
without parameters to get an overview of required and optional parameters.

Required are:
- `backend` -- An IP address to forward queries too (authoritative DNS)
- `geodb` -- The path to a copy of GeoIP database (e.g. `GeoLite2-City.mmdb`)
- `rewrite` -- Rewrite rules for incoming queries

The `rewrite` option accepts multiple key=value pairs:
- `mode` -- Either `add`, `prefix` or `suffix`
- `pos` -- The position/label to be rewritten
- `sep` -- The separator used in modes `prefix`/`suffix` (optional)

In `add` mode, a new label will be inserted at position `pos` containing the
GeoIP ISO 2-letter code. `mode=add pos=1` will transform www.example.org to
www.XX.example.org (XX being an ISO-3166 Alpha 2 code).

In `prefix` or `suffix` mode, the existing label at position `pos` will be
prefixed or suffixed with the GeoIP information, optionally using a separator.
`mode=prefix pos=0` will transform www.example.org to XXwww.example.org.
`mode=suffix pos=0 sep=-` will transform www.example.org to www-XX.example.org.

Optional parameters:
- `listen` -- The IP address to listen on
- `port` -- The UDP/TCP port to listen on
- `debug` -- Enable debug output
- `nxfallback` -- Prevent rewrites to non-existant names

When the `nxfallback` option is activated, geodns-shim will check the response
from the backend for NXDOMAIN response code. If a NXDOMAIN answer is received
the query is retried with the orignal query name. If, for example, the query
was rewritten from www.example.org to www-de.example.org but this name does
not exist, the response for www.example.org (unmodified) is given to the client.
You should probably activate this option, unless you know exactly what you are
doing.

## Status

This code should be considered a proof-of-concept. This means, it is supposed
to work and generally does but comes with no guarantees of any kind.

## Limitations

- A copy of the GeoIP database is required
- DNSSEC can not be supported
