import * as Y from 'yjs'
import { MonacoBinding } from 'y-monaco'
import * as monaco from 'monaco-editor'
import { WebsocketProvider } from 'y-websocket'

const roomname = `monaco-demo-${new Date().toLocaleDateString('en-CA')}`

window.addEventListener('load', () => {
    const ydoc = new Y.Doc()
    const provider = new WebsocketProvider(
        'ws://localhost:8080/ws', // use the public ws server
        // `ws${location.protocol.slice(4)}//${location.host}/ws`, // alternatively: use the local ws server (run `npm start` in root directory)
        roomname,
        ydoc
    )
    const ytext = ydoc.getText('monaco')

    const editor = monaco.editor.create(/** @type {HTMLElement} */(document.getElementById('monaco-editor')), {
        value: '',
        language: 'javascript',
        theme: 'vs-dark'
    })
    const monacoBinding = new MonacoBinding(ytext, /** @type {monaco.editor.ITextModel} */(editor.getModel()), new Set([editor]), provider.awareness)

    const connectBtn = /** @type {HTMLElement} */ (document.getElementById('y-connect-btn'))
    connectBtn.addEventListener('click', () => {
        if (provider.shouldConnect) {
            provider.disconnect()
            connectBtn.textContent = 'Connect'
        } else {
            provider.connect()
            connectBtn.textContent = 'Disconnect'
        }
    })

    // @ts-ignore
    window.example = { provider, ydoc, ytext, monacoBinding }
})