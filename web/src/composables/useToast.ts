import { ref } from 'vue'

export interface Toast {
    id: string
    title: string
    description?: string
    type: 'success' | 'error' | 'info' | 'warning'
    duration?: number
}

const toasts = ref<Toast[]>([])

export function useToast() {
    function addToast(toast: Omit<Toast, 'id'>) {
        const id = Math.random().toString(36).substr(2, 9)
        const newToast: Toast = {
            id,
            duration: 3000,
            ...toast
        }

        toasts.value.push(newToast)

        if (newToast.duration && newToast.duration > 0) {
            setTimeout(() => {
                removeToast(id)
            }, newToast.duration)
        }

        return id
    }

    function removeToast(id: string) {
        const index = toasts.value.findIndex(t => t.id === id)
        if (index > -1) {
            toasts.value.splice(index, 1)
        }
    }

    function success(title: string, description?: string) {
        return addToast({ title, description, type: 'success' })
    }

    function error(title: string, description?: string) {
        return addToast({ title, description, type: 'error' })
    }

    function info(title: string, description?: string) {
        return addToast({ title, description, type: 'info' })
    }

    function warning(title: string, description?: string) {
        return addToast({ title, description, type: 'warning' })
    }

    return {
        toasts,
        addToast,
        removeToast,
        success,
        error,
        info,
        warning
    }
}
