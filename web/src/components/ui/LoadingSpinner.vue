<template>
  <div class="inline-flex items-center gap-2">
    <div
      class="animate-spin rounded-full border-2 border-current border-t-transparent"
      :class="sizeClasses"
    />
    <span v-if="text" class="text-sm text-muted-foreground">{{ text }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  size?: 'sm' | 'md' | 'lg'
  text?: string
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
})

const sizeClasses = computed(() => {
  switch (props.size) {
    case 'sm':
      return 'h-4 w-4'
    case 'lg':
      return 'h-8 w-8'
    default:
      return 'h-6 w-6'
  }
})
</script>
