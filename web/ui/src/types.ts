export interface Connector {
  kind: string
  metadata: {
    name: string
    namespace: string
    createdAt: string
  }
  spec: {
    runtime: {
      remote?: {
        url: string
        transport: string
      }
    }
    tools?: {
      exposePrefix?: string
      allow?: string[]
    }
    headers?: Record<string, string>
    tags?: string[]
  }
  status?: {
    phase: string
    discoveredTools?: number
    url?: string
  }
}

export interface Secret {
  name: string
  keys: string[]
}

export interface SecretCreateRequest {
  name: string
  data: Record<string, string>
}
