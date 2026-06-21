# How to Run MCPHub

MCPHub ships as two binaries plus its CRDs. There is no single config file — you declare
MCPs as `Mcp` resources (via `kubectl`, GitOps, the API, or the web UI).

## Prerequisites

- Go 1.24+
- A Kubernetes cluster (v1.27+) and `kubectl`
- Node 20+ and a package manager (for building the Vite web UI)
- An OAuth2 provider (Google, Okta, Keycloak, ...) for user login

## 1. Install the CRDs

```sh
make install   # applies Mcp / McpBundle / McpPolicy CRDs
```

## 2. Run the controller + API + proxy

In-cluster:

```sh
make deploy IMG=<registry>/mcphub-controller:tag
```

Locally (against your current kube-context):

```sh
mcphub-controller \
  --api-addr=:8080 \
  --proxy-addr=:8081 \
  --oauth2-issuer=https://idp.example.com \
  --oauth2-client-id=mcphub \
  --oauth2-audience=mcphub
```

This single process runs the reconcilers, the management API (`:8080`), and the MCP proxy
(`:8081`).

## 3. Run the web server

```sh
# build the Vite UI once
cd web && npm install && npm run build

# serve it and proxy API calls to the controller
mcphub-web \
  --listen=:3000 \
  --api-url=http://localhost:8080 \
  --oauth2-issuer=https://idp.example.com \
  --oauth2-client-id=mcphub
```

Open `http://localhost:3000`, log in via OAuth2, and you have the catalog UI.

## 4. Declare an MCP

Either `kubectl apply` a resource:

```yaml
apiVersion: mcphub.io/v1alpha1
kind: Mcp
metadata:
  name: github
  namespace: mcphub
spec:
  runtime:
    type: container
    image: ghcr.io/github/github-mcp-server:latest
  transport: stdio
  env:
    - name: GITHUB_PERSONAL_ACCESS_TOKEN
      valueFrom:
        secretKeyRef:
          name: github-credentials
          key: token
  tools:
    exposePrefix: "github_"
    allow: ["search_*", "get_*", "list_*"]
```

...with its secret:

```sh
kubectl create secret generic github-credentials \
  --from-literal=token="$GITHUB_PERSONAL_ACCESS_TOKEN" -n mcphub
```

Or add it from the web UI (which writes the same `Mcp` CR for you, with auto-detected
transport and discovered tools).

## 5. Connect an agent

To a single MCP:

```json
{
  "mcpServers": {
    "mcphub-github": {
      "type": "streamable-http",
      "url": "https://mcphub.example.com/mcp/github",
      "headers": { "Authorization": "Bearer <oauth2-or-service-token>" }
    }
  }
}
```

To a bundle (see [bundles.md](bundles.md)):

```json
{
  "mcpServers": {
    "mcphub-devtools": {
      "type": "streamable-http",
      "url": "https://mcphub.example.com/bundle/devtools",
      "headers": { "Authorization": "Bearer <token>" }
    }
  }
}
```

## 6. Mint a service token (for machines)

```sh
mcphub token create --name ci-runner --scope "bundle:devtools"
# => mh_svc_...   (store securely; shown once)
```

## Uninstall

```sh
make undeploy     # remove the controller
make uninstall    # remove the CRDs
```
