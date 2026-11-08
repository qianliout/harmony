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

const SEALS: Record<string, { char: string; color: PaletteName }> = {
  wedding: { char: '喜', color: 'cinnabar' },
  funeral: { char: '奠', color: 'stone' },
  fullmoon: { char: '满', color: 'rose' },
  housewarming: { char: '迁', color: 'amber' },
  school: { char: '学', color: 'indigo' },
  newyear: { char: '年', color: 'gold' },
  other: { char: '礼', color: 'stone' },
}

export function sealOf(type: string): { char: string; color: PaletteName } {
  return SEALS[type] ?? SEALS.other
}

export const relations = [
  { code: 'relative', label: '亲戚' },
  { code: 'friend', label: '朋友' },
  { code: 'colleague', label: '同事' },
  { code: 'neighbor', label: '邻居' },
  { code: 'other', label: '其他' },
]

export const eventTypes = [
  { code: 'wedding', label: '红事', char: '喜', color: 'cinnabar' },
  { code: 'funeral', label: '白事', char: '奠', color: 'stone' },
  { code: 'fullmoon', label: '满月', char: '满', color: 'rose' },
  { code: 'housewarming', label: '乔迁', char: '迁', color: 'amber' },
  { code: 'school', label: '升学', char: '学', color: 'indigo' },
  { code: 'newyear', label: '过年', char: '年', color: 'gold' },
  { code: 'other', label: '其他', char: '礼', color: 'stone' },
] as const
