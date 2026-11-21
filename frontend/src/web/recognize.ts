// 识别核对的行模型：识别结果 → 可编辑行 → 写入参数。只在 H5 识别页使用。
import type { RecognizeCandidate, RecognizeCommitItem, RecognizeItem } from '@/api/types'
import { centToYuanText, yuanToCent } from '@/utils/money'

/** 置信度低于该值（百分比）的行标黄，提示对照原图。 */
export const LOW_CONFIDENCE = 60

/** personChoice：候选亲友 id；0 表示按姓名新建。 */
export interface ReviewRow {
  seq: number
  checked: boolean
  name: string
  remark: string
  personChoice: number
  candidates: RecognizeCandidate[]
  amountText: string
  note: string
  confidence: number
  raw: string
}

export function toReviewRows(items: RecognizeItem[]): ReviewRow[] {
  return items.map((item) => ({
    seq: item.seq,
    checked: true,
    name: item.name,
    remark: '',
    personChoice: item.match.status === 'single' ? item.match.personId : 0,
    candidates: item.match.candidates,
    amountText: item.amountCent ? centToYuanText(item.amountCent) : '',
    note: item.note,
    confidence: item.confidence,
    raw: item.raw,
  }))
}

/** 有同名亲友却没选、又没填备注区分时，写入会被后端判为歧义或重名。 */
export function rowProblem(row: ReviewRow): string {
  if (!row.name.trim()) return '缺姓名'
  const needChoice = row.candidates.length > 1 && !row.personChoice && !row.remark.trim()
  if (needChoice) return '同名多位，请选择或填备注'
  return ''
}

export function toCommitItem(row: ReviewRow): RecognizeCommitItem {
  const base = { amountCent: yuanToCent(row.amountText), note: row.note.trim() }
  if (row.personChoice) {
    return { ...base, personId: row.personChoice }
  }
  return { ...base, personName: row.name.trim(), personRemark: row.remark.trim() }
}
