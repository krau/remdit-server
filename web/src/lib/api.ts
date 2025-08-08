import type { DocumentResponse, SaveDocumentRequest } from '@/types/document'


// 获取文档内容
export async function getDocument(fileId: string): Promise<DocumentResponse> {
  const response = await fetch(`/api/file/${fileId}`)
  if (!response.ok) {
    throw new Error(`Failed to fetch document: ${response.statusText}`)
  }
  return response.json()
}

// 保存文档内容
export async function saveDocument(
  fileId: string,
  content: string
): Promise<void> {

  const request: SaveDocumentRequest = { content }
  const response = await fetch(`/api/file/${fileId}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(request)
  })

  if (!response.ok) {
    throw new Error(`Failed to save document: ${response.statusText}`)
  }
}
