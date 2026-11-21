// 与后端统一响应契约对齐：传输层恒为 HTTP 200，业务状态在 code，成功为 200。
// 字段名与后端响应一一对应，金额单位为分，时间戳为毫秒，日期为 YYYY-MM-DD。

/** 后端统一响应体。 */
export interface ApiResponse<T> {
  code: number
  msg: string
  err: string
  data: T
  totalItems: number
  itemsPerPage: number
  startIndex: number
  custom: Record<string, unknown> | null
}

/** 分页列表的展开结构。 */
export interface Page<T> {
  items: T[]
  total: number
}

export type Direction = 'in' | 'out'

/** 亲友：姓名 + 备注唯一，displayName 形如「张三 · 村东」。 */
export interface PersonItem {
  id: number
  name: string
  remark: string
  displayName: string
  inCent: number
  outCent: number
  recordCount: number
  lastOn: string
  createdAt: number
  updatedAt: number
}

/** 礼簿只记收礼，合计只有收。 */
export interface BookItem {
  id: number
  name: string
  category: string
  categoryLabel: string
  description: string
  recordCount: number
  inCent: number
  createdAt: number
  updatedAt: number
}

/** 记录：收挂礼簿（bookId > 0），送记事情（bookId 为 0）。 */
export interface RecordItem {
  id: number
  personId: number
  personName: string
  personRemark: string
  personDisplayName: string
  bookId: number
  bookName: string
  direction: Direction
  amountCent: number
  matter: string
  note: string
  occurredOn: string
  createdAt: number
  updatedAt: number
}

/** 记录筛选结果汇总，金额单位为分。 */
export interface RecordSummary {
  count: number
  inCent: number
  outCent: number
}

export interface RecognizeCandidate {
  personId: number
  personName: string
  personRemark: string
}

/** 识别出的一行；confidence 为 0~100。 */
export interface RecognizeItem {
  seq: number
  name: string
  amountCent: number
  note: string
  raw: string
  confidence: number
  match: {
    status: 'single' | 'none' | 'multiple'
    personId: number
    candidates: RecognizeCandidate[]
  }
}

export interface RecognizeTask {
  id: number
  imageSha256: string
  imageSize: number
  status: 'pending' | 'running' | 'succeeded' | 'failed'
  provider: string
  model: string
  items: RecognizeItem[]
  sumCent: number
  errorMessage: string
  committed: 'true' | 'false'
  finishedAt: number
  createdAt: number
  updatedAt: number
}

export interface RecognizeCommitItem {
  personId?: number
  personName?: string
  personRemark?: string
  amountCent: number
  note: string
}

export interface RecognizeCommitParams {
  id: number
  bookId: number
  direction: Direction
  matter: string
  occurredOn: string
  items: RecognizeCommitItem[]
}

export interface RecognizeCommitResult {
  recordIds: number[]
  count: number
  newPersons: number
  sumCent: number
}
