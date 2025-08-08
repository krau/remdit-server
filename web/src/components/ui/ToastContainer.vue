<template>
  <Teleport to="body">
    <div class="fixed top-4 right-4 z-50 flex flex-col gap-2 max-w-sm">
      <TransitionGroup name="toast" tag="div" class="flex flex-col gap-2">
        <div
          v-for="toast in toasts"
          :key="toast.id"
          :class="toastClasses(toast.type)"
          class="rounded-lg border p-4 shadow-lg backdrop-blur-sm transition-all duration-300"
        >
          <div class="flex items-start gap-3">
            <div class="flex-shrink-0">
              <CheckCircle v-if="toast.type === 'success'" class="h-5 w-5 text-green-500" />
              <XCircle v-else-if="toast.type === 'error'" class="h-5 w-5 text-red-500" />
              <AlertCircle v-else-if="toast.type === 'warning'" class="h-5 w-5 text-yellow-500" />
              <Info v-else class="h-5 w-5 text-blue-500" />
            </div>
            <div class="flex-1 min-w-0">
              <h4 class="text-sm font-medium">{{ toast.title }}</h4>
              <p v-if="toast.description" class="mt-1 text-sm text-muted-foreground">
                {{ toast.description }}
              </p>
            </div>
            <button
              @click="removeToast(toast.id)"
              class="flex-shrink-0 opacity-70 hover:opacity-100 transition-opacity"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { CheckCircle, XCircle, AlertCircle, Info, X } from 'lucide-vue-next'
import { useToast } from '@/composables/useToast'

const { toasts, removeToast } = useToast()

function toastClasses(type: string) {
  const baseClasses = 'bg-background/95 border'

  switch (type) {
    case 'success':
      return `${baseClasses} border-green-200 dark:border-green-800`
    case 'error':
      return `${baseClasses} border-red-200 dark:border-red-800`
    case 'warning':
      return `${baseClasses} border-yellow-200 dark:border-yellow-800`
    default:
      return `${baseClasses} border-blue-200 dark:border-blue-800`
  }
}
</script>

<style scoped>
.toast-enter-active {
  transition: all 0.3s ease-out;
}

.toast-leave-active {
  transition: all 0.3s ease-in;
}

.toast-enter-from {
  transform: translateX(100%);
  opacity: 0;
}

.toast-leave-to {
  transform: translateX(100%);
  opacity: 0;
}
</style>
