// 识别链路只在 H5 编译；整个文件内容由条件编译包住，小程序产物中不出现。
// #ifdef H5
import { apiBaseURL, getToken, http, unwrapItem, unwrapPage } from './http'
import type { ApiResponse, Page, RecognizeCommitItem, RecognizeCommitResult, RecognizeTask } from './types'

/** upload 走原生 fetch（FormData 不能经 uni.request 转发），鉴权头需要自己带。 */
export function uploadRecognizeImage(file: File): Promise<RecognizeTask> {
  const form = new FormData()
  form.append('image', file)
  const token = getToken()
  const headers: Record<string, string> = { 'X-Client-Platform': 'h5' }
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  return fetch(`${apiBaseURL()}/recognize/tasks`, {
    method: 'POST',
    body: form,
    headers,
  }).then(async (res) => {
    const body = (await res.json().catch(() => null)) as ApiResponse<RecognizeTask> | null
    if (!body) {
      throw new Error(res.status === 401 ? '登录已过期，请重新登录' : `上传失败（HTTP ${res.status}）`)
    }
    if (body.code !== 200) {
      throw new Error(body.msg || body.err || '上传失败')
    }
    return body.data
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
