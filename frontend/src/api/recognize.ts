// 识别链路只在 H5 编译；整个文件内容由条件编译包住，小程序产物中不出现。
// #ifdef H5
import { apiBaseURL, http, unwrapItem, unwrapPage } from './http'
import type { Page, RecognizeCommitItem, RecognizeCommitResult, RecognizeTask } from './types'

export function uploadRecognizeImage(file: File): Promise<RecognizeTask> {
  const form = new FormData()
  form.append('image', file)
  return fetch(`${apiBaseURL()}/recognize/tasks`, {
    method: 'POST',
    body: form,
    headers: { 'X-Client-Platform': 'h5' },
  })
    .then((res) => res.json())
    .then((body) => {
      if (body.code !== 200) {
        throw new Error(body.msg || body.err || '上传失败')
      }
      return body.data as RecognizeTask
    })
}

export function fetchRecognizeTask(id: number): Promise<RecognizeTask> {
  return http.get<RecognizeTask>('/recognize/tasks/detail', { id }).then(unwrapItem)
}

export function fetchRecognizeTasks(page = 1, pageSize = 10): Promise<Page<RecognizeTask>> {
  return http
    .get<RecognizeTask[]>('/recognize/tasks', { page, pageSize })
    .then(unwrapPage)
}

export function commitRecognizeTask(
  id: number,
  items: RecognizeCommitItem[],
  page: string,
  pageTotalCent: number | null,
  bookId: number,
): Promise<RecognizeCommitResult> {
  return http
    .post<RecognizeCommitResult>('/recognize/tasks/commit', { items, page, pageTotalCent, bookId }, { id })
    .then(unwrapItem)
}
// #endif
