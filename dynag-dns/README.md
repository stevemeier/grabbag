dynag: DNS server with health checks

## Overview

This is a very minimal DNS server which figures out which records to return
to a client based on health checks.

A simple example could be a web-server farm: The DNS should only return IP address
which are actually up and running. To do this, you define the records in the
config file (config.json) and use a health check such as cURL to determine the
availability of each host.

dynag will then periodically run checks to see if each node is healthy and remove
records from the DNS for which the check failed. Once the check succeeds again,
those records will be re-enabled.

## Configuration

The configuration uses a simple JSON-format. See config.json for an example.

The `server` section defines options such as IP address and port to listen on.
The `names` array defines a set of names for which the server will become
authoritative and run the specified `command` each `interval` of seconds.

If the check is successful (exit code == 0), the `rr` record will be enabled in
the DNS. If the check fails (exit code > 0), `rr` will be disabled.

If all `rr` for a name are disabled, the server will return SERVFAIL.

## Status

This code should be considered a proof-of-concept. This means, it is supposed
to work and generally does but comes with no guarantees of any kind.

## Limitations

- Wildcard records are not supported
- DNSSEC is not supported
