<template>
  <div class="h-screen flex flex-col bg-background">
    <!-- 顶部工具栏 -->
    <header
      class="border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60"
    >
      <div class="flex items-center justify-between px-4 py-3">
        <div class="flex items-center gap-4">
          <Button variant="ghost" size="sm" @click="goBack" class="gap-2">
            <ArrowLeft class="h-4 w-4" />
            返回
          </Button>
          <div class="h-6 w-px bg-border" />
          <h1 class="text-lg font-semibold">
            文档编辑
            <span v-if="fileId" class="text-sm text-muted-foreground font-normal ml-2">
              ID: {{ fileId }}
            </span>
          </h1>
        </div>

        <div class="flex items-center gap-3">
          <!-- 协作状态 -->
          <div class="flex items-center gap-2 text-sm text-muted-foreground">
            <div :class="['h-2 w-2 rounded-full', isConnected ? 'bg-green-500' : 'bg-red-500']" />
            {{ isConnected ? '已连接' : '连接中...' }}
          </div>

          <!-- 暗色模式切换 -->
          <DarkModeMenu />

          <!-- 保存按钮 -->
          <Button @click="handleSave" :disabled="isSaving || !isDirty" size="sm" class="gap-2">
            <LoadingSpinner v-if="isSaving" size="sm" />
            <Save v-else class="h-4 w-4" />
            {{ isSaving ? '保存中...' : '保存' }}
          </Button>
        </div>
      </div>
    </header>

    <!-- 主编辑区域 -->
    <main class="flex-1 relative">
      <!-- 加载状态 -->
      <div
        v-if="isLoading"
        class="absolute inset-0 flex items-center justify-center bg-background/80 backdrop-blur-sm z-10"
      >
        <div class="text-center">
          <LoadingSpinner size="lg" />
          <p class="mt-4 text-sm text-muted-foreground">正在加载文档...</p>
        </div>
      </div>

      <!-- 错误状态 -->
      <div
        v-else-if="error"
        class="absolute inset-0 flex items-center justify-center bg-background/80 backdrop-blur-sm z-10"
      >
        <div class="text-center max-w-md">
          <AlertCircle class="h-12 w-12 text-red-500 mx-auto mb-4" />
          <h2 class="text-lg font-semibold mb-2">加载失败</h2>
          <p class="text-sm text-muted-foreground mb-4">{{ error }}</p>
          <Button @click="loadDocument" variant="outline"> 重试 </Button>
        </div>
      </div>

      <!-- Monaco 编辑器 -->
      <MonacoEditor
        v-if="!isLoading && !error"
        v-model="content"
        :file-id="fileId"
        language="markdown"
        class="h-full"
        @save="handleSave"
        @connection-change="isConnected = $event"
      />
    </main>

    <!-- 状态栏 -->
    <footer
      class="border-t border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60"
    >
      <div class="flex items-center justify-between px-4 py-2 text-xs text-muted-foreground">
        <div class="flex items-center gap-4">
          <span>Markdown</span>
          <span v-if="content">{{ content.length }} 字符</span>
        </div>
        <div class="flex items-center gap-4">
          <span v-if="lastSaved"> 最后保存: {{ formatTime(lastSaved) }} </span>
          <kbd class="px-2 py-1 bg-muted rounded text-xs">Ctrl+S 保存</kbd>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Save, AlertCircle } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import LoadingSpinner from '@/components/ui/LoadingSpinner.vue'
import MonacoEditor from '@/components/MonacoEditor.vue'
import DarkModeMenu from '@/components/DarkModeMenu.vue'
import { getDocument, saveDocument } from '@/lib/api'
import { useToast } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()
const { success, error: showError } = useToast()

// 状态管理
const fileId = computed(() => route.params.id as string)
const content = ref('')
const originalContent = ref('')
const isLoading = ref(true)
const isSaving = ref(false)
const error = ref('')
const lastSaved = ref<Date | null>(null)
const isConnected = ref(false)

// 计算属性
const isDirty = computed(() => content.value !== originalContent.value)

// 加载文档
async function loadDocument() {
  if (!fileId.value) {
    error.value = '无效的文档ID'
    isLoading.value = false
    return
  }

  try {
    isLoading.value = true
    error.value = ''

    const response = await getDocument(fileId.value)
    content.value = response.content
    originalContent.value = response.content

    success('文档加载成功')
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载文档失败'
    showError('加载失败', error.value)
  } finally {
    isLoading.value = false
  }
}

// 保存文档
async function handleSave() {
  if (!fileId.value || isSaving.value || !isDirty.value) {
    return
  }

  try {
    isSaving.value = true

    await saveDocument(fileId.value, content.value)
    originalContent.value = content.value
    lastSaved.value = new Date()

    success('文档保存成功')
  } catch (err) {
    const message = err instanceof Error ? err.message : '保存文档失败'
    showError('保存失败', message)
  } finally {
    isSaving.value = false
  }
}

// 返回上一页
function goBack() {
  if (isDirty.value) {
    if (confirm('有未保存的更改，确定要离开吗？')) {
      router.back()
    }
  } else {
    router.back()
  }
}

// 格式化时间
function formatTime(date: Date) {
  return date.toLocaleTimeString('zh-CN', {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

// 监听路由变化
watch(
  () => route.params.id,
  () => {
    if (route.params.id) {
      loadDocument()
    }
  },
  { immediate: true },
)

// 页面离开前提示
onMounted(() => {
  const handleBeforeUnload = (e: BeforeUnloadEvent) => {
    if (isDirty.value) {
      e.preventDefault()
      e.returnValue = ''
    }
  }

  window.addEventListener('beforeunload', handleBeforeUnload)

  onUnmounted(() => {
    window.removeEventListener('beforeunload', handleBeforeUnload)
  })
})
</script>

<style scoped>
/* 自定义滚动条 */
:deep(::-webkit-scrollbar) {
  width: 8px;
  height: 8px;
}

:deep(::-webkit-scrollbar-track) {
  background: hsl(var(--background));
}

:deep(::-webkit-scrollbar-thumb) {
  background: hsl(var(--border));
  border-radius: 4px;
}

:deep(::-webkit-scrollbar-thumb:hover) {
  background: hsl(var(--muted-foreground));
}
</style>
