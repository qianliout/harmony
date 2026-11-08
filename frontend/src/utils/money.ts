// 金额展示与输入换算：金额一律以分存储，仅展示时转元。规则见 docs/03-前端交互与UI规范.md。

export function formatCent(cent: number, opts: { sign?: boolean; compact?: boolean } = {}): string {
  const negative = cent < 0
  const abs = Math.abs(cent)
  let body: string
  if (opts.compact && abs >= 10_000_000) {
    body = `${trimZero((abs / 1_000_000).toFixed(2))}万`
  } else {
    const yuan = Math.floor(abs / 100)
    const fen = abs % 100
    body = yuan.toLocaleString('en-US') + (fen ? `.${String(fen).padStart(2, '0')}` : '')
  }
  const prefix = opts.sign ? (negative ? '−' : '+') : negative ? '−' : ''
  return `${prefix}¥${body}`
}

function trimZero(s: string): string {
  return s.replace(/\.?0+$/, '')
}

/** 元字符串转分：字符串拆分，避免浮点误差。 */
export function yuanToCent(input: string): number {
  const text = String(input ?? '').replace(/,/g, '').trim()
  if (!text) {
    return 0
  }
  const [y, f = ''] = text.split('.')
  const yuan = Number(y || 0)
  if (!Number.isFinite(yuan)) {
    return 0
  }
  return yuan * 100 + Number((f + '00').slice(0, 2) || 0)
}

/** 分转可编辑的元字符串（不带符号与千分位）。 */
export function centToYuanText(cent: number): string {
  const abs = Math.abs(cent)
  const yuan = Math.floor(abs / 100)
  const fen = abs % 100
  return fen ? `${yuan}.${String(fen).padStart(2, '0')}` : String(yuan)
}
