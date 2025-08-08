<template>
  <div
    ref="editorContainer"
    class="w-full h-full min-h-[400px] border border-border rounded-md overflow-hidden"
  />
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import * as monaco from 'monaco-editor'
import { MonacoBinding } from 'y-monaco'
import { WebsocketProvider } from 'y-websocket'
import * as Y from 'yjs'

interface Props {
  modelValue: string
  fileId: string
  language?: string
  readonly?: boolean
  height?: string
}

interface Emits {
  (e: 'update:modelValue', value: string): void
  (e: 'save'): void
  (e: 'connection-change', connected: boolean): void
}

const props = withDefaults(defineProps<Props>(), {
  language: 'markdown',
  readonly: false,
  height: '100%',
})

const emit = defineEmits<Emits>()

const editorContainer = ref<HTMLElement>()
let editor: monaco.editor.IStandaloneCodeEditor | null = null
let binding: MonacoBinding | null = null
let provider: WebsocketProvider | null = null
let ydoc: Y.Doc | null = null

// 设置Monaco编辑器主题
function setupTheme() {
  // 定义暗色主题
  monaco.editor.defineTheme('dark-theme', {
    base: 'vs-dark',
    inherit: true,
    rules: [],
    colors: {
      'editor.background': '#0a0a0a',
      'editor.foreground': '#fafafa',
      'editorLineNumber.foreground': '#525252',
      'editor.selectionBackground': '#374151',
      'editor.lineHighlightBackground': '#1f1f1f',
    },
  })

  // 设置主题
  const isDark = document.documentElement.classList.contains('dark')
  monaco.editor.setTheme(isDark ? 'dark-theme' : 'vs')
}

// 监听主题变化
function setupThemeObserver() {
  const observer = new MutationObserver(() => {
    setupTheme()
  })

  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class'],
  })

  return observer
}

onMounted(() => {
  if (!editorContainer.value) return

  // 设置主题
  setupTheme()
  const themeObserver = setupThemeObserver()

  // 创建YJS文档
  ydoc = new Y.Doc()
  // 创建WebSocket提供者
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/api/ws`
  provider = new WebsocketProvider(wsUrl, props.fileId, ydoc)

  // 获取共享文本类型
  const ytext = ydoc.getText('monaco')

  // 创建Monaco编辑器
  editor = monaco.editor.create(editorContainer.value, {
    value: props.modelValue,
    language: props.language,
    theme: document.documentElement.classList.contains('dark') ? 'dark-theme' : 'vs',
    readOnly: props.readonly,
    automaticLayout: true,
    fontSize: 14,
    lineNumbers: 'on',
    minimap: { enabled: false },
    wordWrap: 'on',
    scrollBeyondLastLine: false,
    renderWhitespace: 'boundary',
    folding: true,
    lineDecorationsWidth: 10,
    lineNumbersMinChars: 4,
  })
  // 创建YJS绑定
  binding = new MonacoBinding(ytext, editor.getModel()!, new Set([editor]), provider.awareness)

  // 如果有初始内容，设置到YJS文档中
  if (props.modelValue) {
    ytext.insert(0, props.modelValue)
  }
  // 监听连接状态
  provider.on('status', ({ status }: { status: string }) => {
    emit('connection-change', status === 'connected')
  })

  // 初始连接状态
  setTimeout(() => {
    emit('connection-change', provider?.wsconnected || false)
  }, 100)

  // 监听内容变化
  editor.onDidChangeModelContent(() => {
    const value = editor?.getValue() || ''
    emit('update:modelValue', value)
  })

  // 添加保存快捷键
  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
    emit('save')
  })

  // 清理函数
  onUnmounted(() => {
    themeObserver.disconnect()
    binding?.destroy()
    provider?.destroy()
    editor?.dispose()
    ydoc?.destroy()
  })
})

// 监听modelValue变化
watch(
  () => props.modelValue,
  (newValue, oldValue) => {
    if (editor && newValue !== oldValue) {
      // 如果YJS文档存在，更新YJS文档而不是直接更新编辑器
      if (ydoc) {
        const ytext = ydoc.getText('monaco')
        const currentContent = ytext.toString()

        if (currentContent !== newValue) {
          // 清空并重新插入内容
          if (currentContent.length > 0) {
            ytext.delete(0, currentContent.length)
          }
          if (newValue) {
            ytext.insert(0, newValue)
          }
        }
      } else {
        // 如果YJS文档还不存在，直接更新编辑器
        if (editor.getValue() !== newValue) {
          editor.setValue(newValue)
        }
      }
    }
  },
)

// 监听readonly变化
watch(
  () => props.readonly,
  (readonly) => {
    if (editor) {
      editor.updateOptions({ readOnly: readonly })
    }
  },
)

// 暴露编辑器实例
defineExpose({
  getEditor: () => editor,
  getValue: () => editor?.getValue() || '',
  setValue: (value: string) => editor?.setValue(value),
  focus: () => editor?.focus(),
})
</script>

<style scoped>
/* Monaco编辑器样式 */
:deep(.monaco-editor) {
  --vscode-editor-background: transparent;
}

:deep(.monaco-editor .margin) {
  background-color: transparent;
}
</style>
