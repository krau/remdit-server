<template>
  <div class="min-h-screen bg-background">
    <div class="container mx-auto px-4 py-8">
      <div class="max-w-4xl mx-auto">
        <header class="text-center mb-12 relative">
          <!-- 暗色模式切换按钮 -->
          <div class="absolute top-0 right-0">
            <DarkModeMenu />
          </div>

          <h1 class="text-4xl font-bold mb-4">Remdit</h1>
          <p class="text-lg text-muted-foreground">Collaborative Remote Document Editing</p>
        </header>

        <div class="grid gap-6 md:grid-cols-2 lg:grid-cols-1">
          <div
            class="bg-card border border-border rounded-lg p-6 hover:shadow-md transition-shadow"
          >
            <div class="flex items-center gap-3 mb-4">
              <div class="p-2 bg-green-500/10 rounded-lg">
                <FileText class="h-6 w-6 text-green-500" />
              </div>
              <h2 class="text-xl font-semibold">协作编辑</h2>
            </div>
            <p class="text-muted-foreground mb-4">
              开始协作编辑本地/远程服务器上的文件, 仅需一个ID
            </p>
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
                开始协作
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { FileText } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import DarkModeMenu from '@/components/DarkModeMenu.vue'

const router = useRouter()

const documentId = ref('')

function joinDocument() {
  if (!documentId.value.trim()) return

  router.push(`/edit/${documentId.value}`)
}
</script>
