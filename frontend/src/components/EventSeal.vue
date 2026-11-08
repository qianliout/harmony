<script setup lang="ts">
import { computed } from 'vue'

import { paletteOf, sealOf } from '@/styles/palette'

const props = withDefaults(
  defineProps<{
    type: string
    size?: 's' | 'm' | 'l'
  }>(),
  { size: 'm' },
)

const seal = computed(() => sealOf(props.type))
const colors = computed(() => paletteOf(seal.value.color))
</script>

<template>
  <view
    class="seal"
    :class="`seal--${size}`"
    :style="{ background: colors.light, color: colors.main, borderColor: colors.main }"
  >
    <text class="seal__char">{{ seal.char }}</text>
  </view>
</template>

<style lang="scss" scoped>
.seal {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  border-radius: 10px;
}

.seal::after {
  content: '';
  position: absolute;
  inset: 2px;
  border: 1px solid currentColor;
  opacity: 0.25;
  border-radius: inherit;
}

.seal__char {
  font-family: $font-serif;
  font-weight: 600;
  line-height: 1;
}

.seal--s {
  width: 28px;
  height: 28px;
  border-radius: 7px;
}

.seal--s .seal__char {
  font-size: 15px;
}

.seal--m {
  width: 40px;
  height: 40px;
  border-radius: 10px;
}

.seal--m .seal__char {
  font-size: 20px;
}

.seal--l {
  width: 64px;
  height: 64px;
  border-radius: 16px;
}

.seal--l .seal__char {
  font-size: 32px;
}
</style>
