// 识别链路只在 H5 编译；整个文件内容由条件编译包住，小程序产物中不出现。
// #ifdef H5
import { apiBaseURL, ApiError, CODE_OK, http, unwrapItem } from './http'
import type { ApiResponse, RecognizeCommitParams, RecognizeCommitResult, RecognizeTask } from './types'

/** upload 走原生 fetch（FormData 不能经 uni.request 转发）。 */
export function uploadRecognizeImage(file: File): Promise<RecognizeTask> {
  const form = new FormData()
  form.append('image', file)
  return fetch(`${apiBaseURL()}/recognize/tasks`, {
    method: 'POST',
    body: form,
    headers: { 'X-Client-Platform': 'h5' },
  }).then(async (res) => {
    const body = (await res.json().catch(() => null)) as ApiResponse<RecognizeTask> | null
    if (!body) {
      throw new ApiError(res.status, `上传失败（HTTP ${res.status}）`)
    }
    if (body.code !== CODE_OK) {
      throw new ApiError(body.code, body.msg || body.err || '上传失败', body.data, body.custom)
    }
    return body.data
  })
}

export function fetchRecognizeTask(id: number): Promise<RecognizeTask> {
  return http.get<RecognizeTask>('/recognize/tasks/detail', { id }).then(unwrapItem)
}

export function recognizeImageURL(id: number): string {
  return `${apiBaseURL()}/recognize/tasks/image?id=${id}`
}

export function commitRecognizeTask(params: RecognizeCommitParams): Promise<RecognizeCommitResult> {
  return http.post<RecognizeCommitResult>('/recognize/tasks/commit', params).then(unwrapItem)
}
// #endif
