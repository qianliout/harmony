export const palette = {
  cinnabar: { main: '#C8372D', light: '#FBEDEA' },
  amber: { main: '#C27C0E', light: '#FBF0DB' },
  teal: { main: '#2E6E72', light: '#E5EFEF' },
  indigo: { main: '#3A63A0', light: '#E8EEF7' },
  rose: { main: '#C2577A', light: '#FAEBF0' },
  sage: { main: '#4F8A56', light: '#E8F1E7' },
  gold: { main: '#A9813A', light: '#F7F0E1' },
  stone: { main: '#6F665F', light: '#EFEBE6' },
} as const

export type PaletteName = keyof typeof palette

export const paletteNames = Object.keys(palette) as PaletteName[]

export function paletteOf(name: string): { main: string; light: string } {
  return palette[(name as PaletteName)] ?? palette.stone
}

/** 同一个人在任何地方颜色一致：按 personId % 8 取色。 */
export function avatarColor(id: number): { main: string; light: string } {
  const names = paletteNames
  return palette[names[Math.abs(id) % names.length]]
}

/** 礼簿种类：预置三类，前端允许输入自定义值。 */
export const bookCategories = [
  { code: 'renqing', label: '人情礼', color: 'cinnabar' },
  { code: 'huopao', label: '火炮', color: 'amber' },
  { code: 'other', label: '其他', color: 'stone' },
] as const

const CATEGORY_COLORS: Record<string, PaletteName> = {
  renqing: 'cinnabar',
  huopao: 'amber',
  other: 'stone',
}

/** 礼簿种类对应的取色名；自定义种类回落到 stone。 */
export function bookCategoryColor(code: string): PaletteName {
  return CATEGORY_COLORS[code] ?? 'stone'
}
