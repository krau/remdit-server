import { toast } from 'vue-sonner'

export function useToast() {
  function success(title: string, description?: string) {
    return toast.success(title, { description })
  }

  function error(title: string, description?: string) {
    return toast.error(title, { description })
  }

  function info(title: string, description?: string) {
    return toast.info(title, { description })
  }

  function warning(title: string, description?: string) {
    return toast.warning(title, { description })
  }

  return {
    success,
    error,
    info,
    warning
  }
}
