<script setup lang="ts">
import { computed } from 'vue'

import { avatarColor } from '@/styles/palette'

const props = withDefaults(
  defineProps<{
    name: string
    id: number
    size?: number
  }>(),
  { size: 40 },
)

const colors = computed(() => avatarColor(props.id))
const initial = computed(() => (props.name ? Array.from(props.name)[0] : '?'))
const fontSize = computed(() => Math.round(props.size * 0.4))
</script>

<template>
  <view
    class="avatar"
    :style="{
      width: `${size}px`,
      height: `${size}px`,
      background: colors.light,
      color: colors.main,
      fontSize: `${fontSize}px`,
    }"
  >
    <text>{{ initial }}</text>
  </view>
</template>

<style lang="scss" scoped>
.avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  border-radius: var(--r-full);
  font-weight: 600;
  line-height: 1;
}
</style>
