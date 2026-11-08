import { apiBaseURL, getToken, http, queryString, unwrapItem } from './http'
import type { ImportCommitResult, ImportPreview, ImportRow } from './types'

export interface ExportParams {
  format: 'csv' | 'json'
  bookId?: number
  eventId?: number
  personId?: number
}

export function exportRecords(params: ExportParams): void {
  const url = `${apiBaseURL()}/export${queryString(params)}`
  const token = getToken()
  const filename = params.format === 'json' ? 'harmony-export.json' : 'harmony-export.csv'
  // #ifdef H5
  fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {} })
    .then(async (res) => {
      if (!res.ok) throw new Error(`导出失败（HTTP ${res.status}）`)
      return res.blob()
    })
    .then((blob) => {
      const objectUrl = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = objectUrl
      link.download = filename
      document.body.appendChild(link)
      link.click()
      setTimeout(() => {
        link.remove()
        URL.revokeObjectURL(objectUrl)
      }, 0)
    })
    .catch((err) => {
      uni.showToast({ title: err instanceof Error ? err.message : '导出失败', icon: 'none' })
    })
  // #endif
  // #ifndef H5
  uni.downloadFile({
    url,
    header: token ? { Authorization: `Bearer ${token}` } : {},
    success: (res) => {
      if (res.statusCode !== 200) {
        uni.showToast({ title: '导出失败', icon: 'none' })
        return
      }
      uni.openDocument({ filePath: res.tempFilePath, showMenu: true })
    },
  })
  // #endif
}

export function previewImportFile(file: File): Promise<ImportPreview> {
  // #ifdef H5
  const form = new FormData()
  form.append('file', file)
  return fetch(`${apiBaseURL()}/import/preview`, {
    method: 'POST',
    body: form,
    headers: {
      'X-Client-Platform': 'h5',
      ...(getToken() ? { Authorization: `Bearer ${getToken()}` } : {}),
    },
  })
    .then((res) => res.json())
    .then((body) => {
      if (body.code !== 200) {
        throw new Error(body.msg || body.err || '导入预览失败')
      }
      return body.data as ImportPreview
    })
  // #endif
  // #ifndef H5
  return Promise.reject(new Error('请在小程序内选择聊天文件'))
  // #endif
}

export function commitImport(rows: ImportRow[]): Promise<ImportCommitResult> {
  return http.post<ImportCommitResult>('/import/commit', { rows }).then(unwrapItem)
}
