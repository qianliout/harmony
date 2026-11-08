<script setup lang="ts">
import { computed } from 'vue'

import { formatCent } from '@/utils/money'

const props = withDefaults(
  defineProps<{
    cent: number
    direction?: 'in' | 'out' | 'none'
    size?: 'display' | 'l' | 'm' | 's'
    sign?: boolean
    compact?: boolean
    tone?: 'pending'
  }>(),
  { direction: 'none', size: 'm', sign: false, compact: false, tone: undefined },
)

const parts = computed(() => {
  const text = formatCent(props.cent, { sign: props.sign, compact: props.compact })
  const index = text.indexOf('¥')
  return { prefix: text.slice(0, index), body: text.slice(index + 1) }
})

const toneClass = computed(() => (props.tone === 'pending' ? 'money--pending' : `money--${props.direction}`))
</script>

<template>
  <text class="money" :class="[`money--${size}`, toneClass]">
    <text class="money__prefix">{{ parts.prefix }}</text><text class="money__symbol">¥</text>{{ parts.body }}
  </text>
</template>

<style lang="scss" scoped>
.money {
  font-family: $font-num;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  white-space: nowrap;
}

.money__symbol {
  font-size: 0.7em;
  margin-right: 1px;
  font-weight: 600;
}

.money__prefix {
  font-size: 0.9em;
}

.money--in {
  color: var(--c-in);
}

.money--out {
  color: var(--c-out);
}

.money--none {
  color: var(--c-ink-1);
}

.money--pending {
  color: var(--c-pending);
}

.money--display {
  font-size: 32px;
  line-height: 40px;
}

.money--l {
  font-size: 22px;
  line-height: 28px;
}

.money--m {
  font-size: 17px;
  line-height: 24px;
}

.money--s {
  font-size: 14px;
  line-height: 20px;
  font-weight: 500;
}
</style>
