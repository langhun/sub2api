import { onMounted, onUnmounted, type Ref } from 'vue'

export interface KeyboardShortcutOptions {
  searchInputRef?: Ref<HTMLInputElement | null>
  onRefresh?: () => void
  onSelectAll?: () => void
  onClearSelection?: () => void
  onDelete?: () => void
  onEscape?: () => void
  disabled?: Ref<boolean>
}

export function useKeyboardShortcuts(options: KeyboardShortcutOptions = {}) {
  const {
    searchInputRef,
    onRefresh,
    onSelectAll,
    onClearSelection,
    onDelete,
    onEscape,
    disabled
  } = options

  const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad|iPod/.test(navigator.platform)

  const isModifierKey = (event: KeyboardEvent) => {
    return isMac ? event.metaKey : event.ctrlKey
  }

  const isInputElement = (element: Element | null): boolean => {
    if (!element) return false
    const tagName = element.tagName.toLowerCase()
    return (
      tagName === 'input' ||
      tagName === 'textarea' ||
      tagName === 'select' ||
      element.hasAttribute('contenteditable')
    )
  }

  const handleKeyDown = (event: KeyboardEvent) => {
    if (disabled?.value) return

    const target = event.target as Element
    const isInInput = isInputElement(target)

    // Escape: 关闭模态框/清除选择
    if (event.key === 'Escape') {
      if (onEscape) {
        onEscape()
        return
      }
      if (onClearSelection && !isInInput) {
        onClearSelection()
        event.preventDefault()
      }
      return
    }

    // Ctrl/Cmd + K: 聚焦搜索框
    if (event.key === 'k' && isModifierKey(event)) {
      if (searchInputRef?.value) {
        event.preventDefault()
        searchInputRef.value.focus()
        searchInputRef.value.select()
      }
      return
    }

    // Ctrl/Cmd + A: 全选（在表格中）
    if (event.key === 'a' && isModifierKey(event) && !isInInput) {
      if (onSelectAll) {
        event.preventDefault()
        onSelectAll()
      }
      return
    }

    // Ctrl/Cmd + R: 刷新数据（阻止默认刷新）
    if (event.key === 'r' && isModifierKey(event)) {
      if (onRefresh) {
        event.preventDefault()
        onRefresh()
      }
      return
    }

    // Delete: 删除选中项（需要确认）
    if (event.key === 'Delete' && !isInInput) {
      if (onDelete) {
        event.preventDefault()
        onDelete()
      }
      return
    }
  }

  onMounted(() => {
    document.addEventListener('keydown', handleKeyDown)
  })

  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeyDown)
  })

  return {
    isMac
  }
}

