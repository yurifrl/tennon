# Spec Connectors (OpenAPI & GraphQL)

`OpenApiConnector` and `GraphqlConnector` turn an API **specification** into MCP tools. Unlike
an [`TennonConnector`](crds.md#tennonconnector) — which connects to an MCP server and *discovers*
its tools — a spec connector **compiles** tools from a document and runs **in-process** in the
controller/proxy. There is no Pod, no container, and nothing to sandbox: a tool call is just
an outbound HTTP or GraphQL request that Tennon makes on the caller's behalf.

This page covers how spec connectors are built, authenticated, versioned, and how
provider-specific cases like **Google** are handled without bespoke code. For the extension
strategy behind it, see [Connector Extensibility](connector-extensibility.md).

## Lifecycle

```
spec source ──fetch──▶ parse ──compile──▶ tools (name + JSON schema) ──serve──▶ /mcp/<name>
                                                                  │
                                          call ◀──render request──┘  (+ injected credential)
```

1. **Fetch.** The controller pulls the spec from `specUrl` (OpenAPI) or introspects the
   `endpoint` (GraphQL). The spec text is stored by hash, never inline.
2. **Compile.** Each OpenAPI operation (or GraphQL field/mutation) becomes a tool with an
   `inputSchema`/`outputSchema` derived from the spec. `tools.allow`/`deny` globs scope which
   operations are exposed; `tools.exposePrefix` namespaces them.
3. **Serve.** The compiled tools are served over the MCP protocol at `/mcp/<name>`, exactly
   like any other connector — the proxy and authorization engine cannot tell a spec connector
   from an MCP one.
4. **Invoke.** On a tool call the connector renders the concrete request (path/params/body
   for OpenAPI, query/variables for GraphQL) and applies the injected credential via
   declarative placements (header/query, name, prefix). The credential value is resolved at
   call time and never captured in the tool definition.

Because tools are compiled rather than discovered, the **output is uniform** with every other
connector: a set of tools with schemas and annotations that policy and approvals act
on identically.

## Authentication

A spec connector reads its credential from a Kubernetes `Secret` by convention
(`tennon-<connector-name>`) and declares *how* it is applied:

```yaml
spec:
  auth:
    in: header                 # header | query
    name: Authorization
    format: "Bearer {{ .value }}"
```

For OpenAPI, Tennon can **derive** candidate auth methods from the spec's `securitySchemes`
and pre-fill them in the UI; if none are found it surfaces "No authentication detected — add a
method, or connect without auth." OAuth2 upstreams reference a reusable
[`OAuthApp`](crds.md#oauthapp); the minted token lands in the connector's Secret and is
refreshed on expiry.

## Versioning & refresh

- The fetched spec is **content-hashed** into `status.specHash`; this is the tools cache key.
- Re-fetching is explicit (`tctl connector refresh <name>`) or, where configured, on the
  next apply. When the hash changes, tools are rebuilt; sharing, credentials, and policy are
  preserved across the rebuild.
- There is no dependency or package version to pin — the "library" is a remote document, so a
  version is whichever spec the URL points at, captured by its hash.

## Provider-specific cases: Google is not a special kind

A provider like **Google Workspace** does **not** get its own connector kind or any bespoke
runtime. Google is an `OpenApiConnector` fed by Google's **API Discovery** documents, which
the controller converts to OpenAPI 3.x in-process and then compiles like any other spec:

```yaml
apiVersion: tennon.io/v1alpha1
kind: OpenApiConnector
metadata:
  name: google-gmail
spec:
  specUrl: https://www.googleapis.com/discovery/v1/apis/gmail/v1/rest
  auth:
    in: header
    name: Authorization
    format: "Bearer {{ .value }}"   # OAuth2 token from an OAuthApp; tennon-google-gmail Secret
```

The only Google-specific *code* is the Discovery→OpenAPI converter. Everything else is data:

- **A curated catalog** (the list of Google services, their Discovery URLs, icons, and
  scopes) lives as preset data the UI seeds from — adding a service is a data row, not code.
- **An OAuth consent grouping** (which scopes are standard-user vs admin-only) is a small
  classification on those presets, driving the UI's grouping and warnings.

Multiple Google accounts (work, personal) are simply two connectors with two Secrets — there
is no connection sub-resource. See [Connectors](crds.md#connectors).

## Runtime contrast

| | spec connector | `TennonConnector` (container) |
|--|----------------|----------------------------|
| Tools | compiled from a spec | discovered by connecting |
| Runs | in-process, no Pod | a jailed container/Pod |
| Scaling | n/a (stateless HTTP) | `replicas` / native HPA |
| Sandbox | nothing to jail | [Capability Profile](capability-profiles.md) |

This is why spec connectors carry no `runtime` or `scaling` block, and why most integrations
should be a spec connector when an API offers a spec — there is far less to run and secure.

See also: [Custom Resources](crds.md) · [Authentication](authentication.md) ·
[Connector Extensibility](connector-extensibility.md).
