<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'

import { centToYuanText } from '@/utils/money'

const props = withDefaults(
  defineProps<{
    modelValue: string
    quickAmounts: number[]
    disabled?: boolean
  }>(),
  { disabled: false },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  save: []
}>()

const LONG_PRESS_MS = 450
let pressTimer: ReturnType<typeof setTimeout> | null = null
const showHint = ref(false)

const keys = ['1', '2', '3', '4', '5', '6', '7', '8', '9', '.', '0', '00']

function update(value: string): void {
  emit('update:modelValue', value)
}

function press(key: string): void {
  if (key === 'save') {
    emit('save')
    return
  }
  let value = props.modelValue ?? ''
  if (key === 'back') {
    update(value.slice(0, -1))
    return
  }
  if (key === '.') {
    if (value.includes('.')) return
    update(value ? `${value}.` : '0.')
    return
  }
  if (key === '00') {
    press('0')
    press('0')
    return
  }
  if (value.includes('.')) {
    const [, fraction = ''] = value.split('.')
    if (fraction.length >= 2) return
    update(value + key)
    return
  }
  const integerPart = value.split('.')[0]
  if (integerPart.length >= 8) return
  update(value === '0' ? key : value + key)
}

function pickQuick(cent: number): void {
  update(centToYuanText(cent))
}

function startLongPress(): void {
  pressTimer = setTimeout(() => {
    update('')
    showHint.value = true
    setTimeout(() => {
      showHint.value = false
    }, 800)
  }, LONG_PRESS_MS)
}

function cancelLongPress(): void {
  if (pressTimer) {
    clearTimeout(pressTimer)
    pressTimer = null
  }
}

function tapKey(key: string): void {
  cancelLongPress()
  if (props.disabled) {
    showHint.value = true
    setTimeout(() => {
      showHint.value = false
    }, 1200)
    return
  }
  press(key)
}

function onKeydown(event: KeyboardEvent): void {
  if (props.disabled) return
  const { key } = event
  if (/^[0-9]$/.test(key)) {
    press(key)
  } else if (key === '.') {
    press('.')
  } else if (key === 'Backspace') {
    press('back')
  } else if (key === 'Enter') {
    emit('save')
  } else {
    return
  }
  event.preventDefault()
}

onMounted(() => {
  // #ifdef H5
  window.addEventListener('keydown', onKeydown)
  // #endif
})

onUnmounted(() => {
  // #ifdef H5
  window.removeEventListener('keydown', onKeydown)
  // #endif
  if (pressTimer) clearTimeout(pressTimer)
})
</script>

<template>
  <view class="pad">
    <view class="pad__quick">
      <text v-for="cent in quickAmounts" :key="cent" class="pad__quick-btn" @click="pickQuick(cent)">
        {{ centToYuanText(cent) }}
      </text>
    </view>
    <view class="pad__body">
      <view class="pad__keys">
        <view v-for="key in keys" :key="key" class="pad__key" @click="tapKey(key)">
          <text>{{ key }}</text>
        </view>
      </view>
      <view class="pad__side">
        <view
          class="pad__key pad__key--del"
          @touchstart="startLongPress"
          @touchend="cancelLongPress"
          @touchcancel="cancelLongPress"
          @click="tapKey('back')"
        >
          <text>⌫</text>
        </view>
        <view class="pad__save" :class="{ 'pad__save--disabled': disabled }" @click="tapKey('save')">
          <text>保存</text>
        </view>
      </view>
    </view>
    <text v-if="showHint" class="pad__hint">{{ disabled ? '金额和礼品至少填一项' : '已清空金额' }}</text>
  </view>
</template>

<style lang="scss" scoped>
.pad {
  position: relative;
  flex: none;
  padding: 8px 6px calc(10px + env(safe-area-inset-bottom));
  background: #e9e2d8;
}

.pad__quick {
  display: flex;
  gap: 6px;
  padding: 0 2px 8px;
  overflow-x: auto;
}

.pad__quick-btn {
  flex: none;
  height: 30px;
  padding: 0 12px;
  border-radius: 8px;
  background: var(--c-surface);
  color: var(--c-ink-2);
  font-family: $font-num;
  font-size: 14px;
  line-height: 30px;
  text-align: center;
}

.pad__body {
  display: flex;
  gap: 6px;
}

.pad__keys {
  flex: 3;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  grid-auto-rows: 46px;
  gap: 6px;
}

.pad__side {
  flex: 1.1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.pad__key {
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: var(--c-surface);
  font-family: $font-num;
  font-size: 22px;
  color: var(--c-ink-1);
  box-shadow: 0 1px 0 rgba(0, 0, 0, 0.06);
}

.pad__key:active {
  background: var(--c-fill);
}

.pad__key--del {
  font-size: 18px;
  color: var(--c-ink-2);
}

.pad__save {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: var(--c-primary);
  color: var(--c-surface);
  font-size: 17px;
  font-weight: 600;
}

.pad__save--disabled {
  opacity: 0.45;
}

.pad__hint {
  position: absolute;
  left: 50%;
  bottom: 90px;
  transform: translateX(-50%);
  padding: 6px 12px;
  border-radius: var(--r-full);
  background: var(--c-ink-1);
  color: var(--c-surface);
  font-size: 12px;
  white-space: nowrap;
}
</style>
