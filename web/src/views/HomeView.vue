<template>
  <div class="min-h-screen bg-background">
    <div class="container mx-auto px-4 py-8">
      <div class="max-w-4xl mx-auto">
        <header class="text-center mb-12">
          <h1 class="text-4xl font-bold mb-4">协作文档编辑器</h1>
          <p class="text-lg text-muted-foreground">基于 YJS + Monaco 的实时协作编辑应用</p>
        </header>

        <div class="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          <!-- 新建文档 -->
          <div
            class="bg-card border border-border rounded-lg p-6 hover:shadow-md transition-shadow"
          >
            <div class="flex items-center gap-3 mb-4">
              <div class="p-2 bg-primary/10 rounded-lg">
                <FileText class="h-6 w-6 text-primary" />
              </div>
              <h2 class="text-xl font-semibold">新建文档</h2>
            </div>
            <p class="text-muted-foreground mb-4">创建一个新的协作文档，邀请他人一起编辑。</p>
            <Button @click="createNewDocument" class="w-full"> 开始创建 </Button>
          </div>

          <!-- 示例文档 -->
          <div
            class="bg-card border border-border rounded-lg p-6 hover:shadow-md transition-shadow"
          >
            <div class="flex items-center gap-3 mb-4">
              <div class="p-2 bg-blue-500/10 rounded-lg">
                <BookOpen class="h-6 w-6 text-blue-500" />
              </div>
              <h2 class="text-xl font-semibold">示例文档</h2>
            </div>
            <p class="text-muted-foreground mb-4">查看预设的示例文档，了解编辑器功能。</p>
            <Button @click="openSampleDocument" variant="outline" class="w-full"> 打开示例 </Button>
          </div>

          <!-- 加入协作 -->
          <div
            class="bg-card border border-border rounded-lg p-6 hover:shadow-md transition-shadow"
          >
            <div class="flex items-center gap-3 mb-4">
              <div class="p-2 bg-green-500/10 rounded-lg">
                <Users class="h-6 w-6 text-green-500" />
              </div>
              <h2 class="text-xl font-semibold">加入协作</h2>
            </div>
            <p class="text-muted-foreground mb-4">通过文档ID加入他人的协作编辑。</p>
            <div class="space-y-3">
              <input
                v-model="documentId"
                type="text"
                placeholder="输入文档ID"
                class="w-full px-3 py-2 border border-input rounded-md bg-background text-foreground"
              />
              <Button
                @click="joinDocument"
                :disabled="!documentId.trim()"
                variant="outline"
                class="w-full"
              >
                加入协作
              </Button>
            </div>
          </div>
        </div>

        <!-- 最近文档 -->
        <section class="mt-12">
          <h2 class="text-2xl font-semibold mb-6">最近编辑的文档</h2>
          <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            <div
              v-for="doc in recentDocuments"
              :key="doc.id"
              class="bg-card border border-border rounded-lg p-4 hover:shadow-md transition-shadow cursor-pointer"
              @click="openDocument(doc.id)"
            >
              <div class="flex items-start justify-between mb-2">
                <h3 class="font-medium truncate">{{ doc.title }}</h3>
                <span class="text-xs text-muted-foreground">{{
                  formatDate(doc.lastModified)
                }}</span>
              </div>
              <p class="text-sm text-muted-foreground line-clamp-2">{{ doc.preview }}</p>
            </div>
          </div>

          <div v-if="recentDocuments.length === 0" class="text-center py-12">
            <FileText class="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <p class="text-muted-foreground">暂无最近编辑的文档</p>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { FileText, BookOpen, Users } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { useToast } from '@/composables/useToast'

const router = useRouter()
const { success, info } = useToast()

const documentId = ref('')

// 模拟最近文档数据
const recentDocuments = ref([
  {
    id: 'sample-doc-1',
    title: '项目需求文档',
    preview: '这是一个示例文档，展示了协作编辑的基本功能...',
    lastModified: new Date(Date.now() - 2 * 60 * 60 * 1000), // 2小时前
  },
  {
    id: 'sample-doc-2',
    title: '会议纪要',
    preview: '今日会议讨论了项目进度和下一步计划...',
    lastModified: new Date(Date.now() - 24 * 60 * 60 * 1000), // 1天前
  },
  {
    id: 'sample-doc-3',
    title: 'API文档',
    preview: '详细描述了系统各个接口的使用方法...',
    lastModified: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000), // 3天前
  },
])

// 创建新文档
function createNewDocument() {
  const newId = 'doc-' + Math.random().toString(36).substr(2, 9)
  info('创建新文档', `文档ID: ${newId}`)
  router.push(`/edit/${newId}`)
}

// 打开示例文档
function openSampleDocument() {
  router.push('/edit/sample-document')
}

// 加入文档协作
function joinDocument() {
  if (!documentId.value.trim()) return

  success('正在加入协作', `文档ID: ${documentId.value}`)
  router.push(`/edit/${documentId.value}`)
}

// 打开文档
function openDocument(id: string) {
  router.push(`/edit/${id}`)
}

// 格式化日期
function formatDate(date: Date) {
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const hours = Math.floor(diff / (1000 * 60 * 60))
  const days = Math.floor(hours / 24)

  if (days > 0) {
    return `${days}天前`
  } else if (hours > 0) {
    return `${hours}小时前`
  } else {
    return '刚刚'
  }
}
</script>
