// 与后端统一响应契约对齐：传输层恒为 HTTP 200，业务状态在 code，成功为 200。
// 字段名与 docs/02-后端设计.md 的 resp 一一对应，金额单位为分。

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

export interface ConfirmConflict {
  id?: number
  summary: string
  createdAt?: number
}

export interface RecordFilters {
  bookId: number
  eventId: number
  itemKind: string
  direction: string
  status: string
  from: string
  to: string
}

export interface ConfirmPayload {
  kind: string
  title: string
  message: string
  conflicts: ConfirmConflict[]
  confirmText: string
}

/** 后端健康检查响应。 */
export interface HealthStatus {
  status: string
  database: string
  version: string
  time: number
}

export interface DictItem {
  id: number
  type: string
  code: string
  label: string
  color: string
  sort: number
  builtin: boolean
}

export interface PersonBrief {
  id: number
  name: string
  relation: string
  relationLabel: string
  phone: string
  remark: string
  createdAt: number
  updatedAt: number
  inCent: number
  outCent: number
  pendingCount: number
  lastDate: string
}

export interface PersonSummary {
  inCent: number
  outCent: number
  netCent: number
  pendingCount: number
  pendingCent: number
  recordCount: number
  lastDate: string
}

export interface PersonDetail {
  id: number
  name: string
  relation: string
  relationLabel: string
  phone: string
  remark: string
  createdAt: number
  updatedAt: number
  summary: PersonSummary
}

export interface PersonRef {
  id: number
  name: string
  relation: string
  relationLabel: string
}

export interface ItemPart {
  itemKind: string
  itemKindLabel: string
  direction: string
  cent: number
  recordCount: number
}

export interface BookItem {
  id: number
  name: string
  category: string
  categoryLabel: string
  categoryColor: string
  defaultEventId: number
  defaultEventName: string
  description: string
  status: string
  mergedIntoId: number
  mergedIntoName: string
  mergedAt: number
  recordCount: number
  inCent: number
  outCent: number
  pendingCount: number
  createdAt: number
  updatedAt: number
  byItem?: ItemPart[]
}

export interface EventItem {
  id: number
  name: string
  type: string
  typeLabel: string
  eventDate: string
  datePrecision: string
  location: string
  host: string
  hostLabel: string
  hostPersonId: number
  hostPersonName: string
  placeholder: boolean
  remark: string
  recordCount: number
  inCent: number
  outCent: number
  personCount: number
  repaidCount: number
  pendingCount: number
  createdAt: number
  updatedAt: number
}

export interface RecordItem {
  id: number
  personId: number
  personName: string
  personRelation: string
  eventId: number
  eventName: string
  eventType: string
  bookId: number
  bookName: string
  originBookId: number
  originBookName: string
  itemKind: string
  itemKindLabel: string
  direction: 'in' | 'out'
  amountCent: number
  giftName: string
  giftValueCent: number
  totalCent: number
  occurredOn: string
  datePrecision: string
  status: 'pending' | 'repaid' | 'none'
  statusLabel: string
  remark: string
  paperNote: string
  paperPage: string
  paperSeq: number
  source: string
  repayCount: number
  createdAt: number
  updatedAt: number
}

export interface RecordDetail extends RecordItem {
  repayRecords: RecordItem[]
  originRecords: RecordItem[]
}

export interface EventBrief {
  id: number
  name: string
  type: string
  eventDate: string
  host: string
  location: string
}

export interface TimelinePart {
  label: string
  cent: number
}

export interface TimelineGroup {
  event: EventBrief
  inCent: number
  outCent: number
  parts: TimelinePart[]
  records: RecordItem[]
}

export interface Timeline {
  groups: TimelineGroup[]
}

export interface EventPersonLine {
  bookName: string
  itemKindLabel: string
  direction: string
  cent: number
  recordIds: number[]
}

export interface EventPerson {
  person: PersonRef
  inCent: number
  outCent: number
  pendingCount: number
  details: EventPersonLine[]
}

export interface EventSummary {
  event: EventBrief
  summary: {
    inCent: number
    outCent: number
    personCount: number
    repaidCount: number
    pendingCount: number
    recordCount: number
  }
  persons: EventPerson[]
}

export interface PendingPerson {
  person: PersonRef
  count: number
  totalCent: number
  earliest: string
}

export interface PendingList {
  persons: PendingPerson[]
  personCount: number
  totalCount: number
  totalCent: number
}

export interface MonthStat {
  month: number
  inCent: number
  outCent: number
  recordCount: number
}

export interface KindStat {
  itemKind: string
  itemKindLabel: string
  direction: string
  cent: number
  recordCount: number
}

export interface EventTypeStat {
  type: string
  typeLabel: string
  cent: number
  recordCount: number
  eventCount: number
}

export interface PersonStat {
  person: PersonRef
  inCent: number
  outCent: number
  netCent: number
  recordCount: number
}

export interface BookStat {
  book: BookItem
  inCent: number
  outCent: number
  recordCount: number
}

export interface YearStats {
  year: number
  inCent: number
  outCent: number
  netCent: number
  recordCount: number
  personCount: number
  monthly: MonthStat[]
  byItem: KindStat[]
  byEventType: EventTypeStat[]
  topPersons: PersonStat[]
  byBook: BookStat[]
}

export interface SearchPersonHit {
  id: number
  name: string
  relation: string
  relationLabel: string
  recordCount: number
}

export interface SearchEventHit {
  id: number
  name: string
  type: string
  typeLabel: string
  eventDate: string
  host: string
}

export interface SearchBookHit {
  id: number
  name: string
  category: string
  categoryLabel: string
  recordCount: number
}

export interface SearchRecordHit extends RecordItem {
  matchField: string
}

export interface SearchGroup<T> {
  total: number
  items: T[]
}

export interface SearchResult {
  intent: string
  query: string
  persons: SearchGroup<SearchPersonHit>
  events: SearchGroup<SearchEventHit>
  books: SearchGroup<SearchBookHit>
  records: SearchGroup<SearchRecordHit>
}

export interface Settings {
  showOwing: boolean
  lastBookId: number
  lastEventId: number
  quickAmounts: number[]
}

export interface PaperPage {
  page: string
  declaredCent: number
  recordCent: number
  recordCount: number
  hasDeclared: boolean
  consistent: boolean
  diffCent: number
}

export interface PaperPageDetail extends PaperPage {
  records: RecordItem[]
}

export interface MergeSourceSnapshot {
  id: number
  name: string
  category: string
  recordCount: number
}

export interface MergePreview {
  target: BookItem
  sources: BookItem[]
  recordCount: number
  crossCategory: boolean
  categories: string[]
  itemCounts: { itemKind: string; itemKindLabel: string; recordCount: number }[]
}

export interface MergeLog {
  id: number
  targetBookId: number
  targetName: string
  sources: MergeSourceSnapshot[]
  recordCount: number
  crossCategory: boolean
  createdAt: number
}

export interface RepayItem {
  bookId: number
  itemKind: string
  amountCent: number
  giftName: string
  giftValueCent: number
}

export interface EventParam {
  id: number
  name: string
  type: string
  date: string
  datePrecision: string
}

export interface PaperDraft {
  event: EventParam | null
  occurredOn: string
  datePrecision: string
  items: RepayItem[]
  raw: string
  confidence: 'high' | 'low'
}

export interface PaperParseResult {
  onlyMark: boolean
  repays: PaperDraft[]
}

export interface RecognizeItem {
  seq: number
  name: string
  amountUpper: string
  amountLower: number | null
  amount: number
  giftName: string
  itemKind: string
  direction: string
  date: string
  note: string
  raw: string
  confidence: number
  amountMismatch: boolean
  match?: {
    status: 'single' | 'none' | 'multiple'
    personId: number
    candidates: { id: number; name: string; relationLabel: string; lastDate: string }[]
  }
  repayDrafts?: PaperDraft[]
  duplicateOf?: number
}

export interface RecognizeTask {
  id: number
  status: 'pending' | 'running' | 'succeeded' | 'failed'
  provider: string
  model: string
  imagePath: string
  imageUrl: string
  imageSha256: string
  imageSize: number
  errorMessage: string
  committed: boolean
  page: string
  pageTotalCent: number
  sumCent: number
  items: RecognizeItem[]
  createdAt: number
  finishedAt: number
}

export interface RecognizeCommitItem {
  personId?: number
  personName?: string
  eventId: number
  bookId: number
  itemKind: string
  direction: string
  amountCent: number
  giftName: string
  giftValueCent: number
  occurredOn: string
  datePrecision: string
  status: string
  remark: string
  paperNote: string
  paperPage: string
  paperSeq: number
  repays: {
    originIds?: number[]
    markRepaid?: boolean
    event: EventParam | null
    occurredOn: string
    datePrecision: string
    items: RepayItem[]
    remark?: string
  }[]
}

export interface RecognizeCommitResult {
  recordIds: number[]
  count: number
  newPersons: number
  repayCount: number
  sumCent: number
}

export interface ImportRow {
  date: string
  personName: string
  relation: string
  eventName: string
  eventType: string
  eventDate: string
  bookName: string
  itemKind: string
  direction: string
  amount: string
  giftName: string
  giftValue: string
  status: string
  remark: string
}

export interface ImportPreviewRow {
  line: number
  ok: boolean
  errors: string[]
  row: ImportRow
}

export interface ImportPreview {
  rows: ImportPreviewRow[]
  total: number
  okCount: number
  errorCount: number
}

export interface ImportCommitResult {
  count: number
  newPersons: number
  newEvents: number
  newBooks: number
}
