// 金额中文大写：详情页致敬纸质礼簿，如 50000 分 → 伍佰元整。

const DIGITS = ['零', '壹', '贰', '叁', '肆', '伍', '陆', '柒', '捌', '玖']
const UNITS = ['', '拾', '佰', '仟']
const GROUPS = ['', '万', '亿']

export function centToChinese(cent: number): string {
  if (!cent || cent <= 0) {
    return '零元整'
  }
  const yuan = Math.floor(cent / 100)
  const jiao = Math.floor((cent % 100) / 10)
  const fen = cent % 10
  let result = yuanToChinese(yuan)
  if (yuan > 0) {
    result += '元'
  }
  if (jiao > 0) {
    result += `${DIGITS[jiao]}角`
  }
  if (fen > 0) {
    result += `${DIGITS[fen]}分`
  }
  if (jiao === 0 && fen === 0) {
    result += '整'
  }
  return result
}

function yuanToChinese(yuan: number): string {
  if (yuan === 0) {
    return '零'
  }
  const groups: string[] = []
  let value = yuan
  let index = 0
  while (value > 0) {
    const section = value % 10000
    if (section > 0) {
      groups.unshift(sectionToChinese(section) + GROUPS[index])
    } else if (groups.length > 0 && !groups[0].startsWith('零')) {
      groups.unshift('零')
    }
    value = Math.floor(value / 10000)
    index++
  }
  return groups.join('').replace(/零+$/, '').replace(/零{2,}/g, '零')
}

function sectionToChinese(section: number): string {
  let result = ''
  let zero = false
  for (let position = 3; position >= 0; position--) {
    const digit = Math.floor(section / Math.pow(10, position)) % 10
    if (digit === 0) {
      if (result) {
        zero = true
      }
      continue
    }
    if (zero) {
      result += '零'
      zero = false
    }
    result += DIGITS[digit] + UNITS[position]
  }
  return result
}
