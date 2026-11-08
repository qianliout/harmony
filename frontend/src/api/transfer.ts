import { apiBaseURL, http, queryString, unwrapItem } from './http'
import type { ImportCommitResult, ImportPreview, ImportRow } from './types'

export interface ExportParams {
  format: 'csv' | 'json'
  bookId?: number
  eventId?: number
  personId?: number
}

export function exportRecords(params: ExportParams): void {
  const url = `${apiBaseURL()}/export${queryString(params)}`
  // #ifdef H5
  const link = document.createElement('a')
  link.href = url
  link.download = params.format === 'json' ? 'harmony-export.json' : 'harmony-export.csv'
  link.rel = 'noopener'
  document.body.appendChild(link)
  link.click()
  setTimeout(() => link.remove(), 0)
  // #endif
  // #ifndef H5
  uni.downloadFile({
    url,
    success: (res) => {
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
    headers: { 'X-Client-Platform': 'h5' },
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
