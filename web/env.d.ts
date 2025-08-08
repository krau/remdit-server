/// <reference types="vite/client" />

declare module 'y-monaco' {
  import type * as Y from 'yjs'
  import type * as monaco from 'monaco-editor'
  import type { Awareness } from 'y-protocols/awareness'

  export class MonacoBinding {
    constructor(
      ytext: Y.Text,
      model: monaco.editor.ITextModel,
      editors?: Set<monaco.editor.ICodeEditor>,
      awareness?: Awareness
    )
    destroy(): void
  }
}

declare module 'y-websocket' {
  import type * as Y from 'yjs'
  import type { Awareness } from 'y-protocols/awareness'

  export class WebsocketProvider {
    constructor(
      serverUrl: string,
      roomname: string,
      doc: Y.Doc,
      options?: {
        connect?: boolean
        awareness?: Awareness
        params?: any
        WebSocketPolyfill?: any
        resyncInterval?: number
        maxBackoffTime?: number
      }
    )

    awareness: Awareness
    wsconnected: boolean
    wsconnecting: boolean
    shouldConnect: boolean

    connect(): void
    disconnect(): void
    destroy(): void

    on(event: string, handler: Function): void
    off(event: string, handler: Function): void
  }
}
