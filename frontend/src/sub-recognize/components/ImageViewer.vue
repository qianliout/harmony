<script setup lang="ts">
// 原图查看器：滚轮以鼠标位置缩放、拖动平移、双击适应/100%、旋转 90 度。
// 缩放与旋转只改 CSS transform，不重新上传或识别；旋转角度按任务 id 存 localStorage。
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    src: string
    taskId?: number
    overlay?: boolean
  }>(),
  { taskId: 0, overlay: false },
)

const emit = defineEmits<{ close: [] }>()

const stage = ref<HTMLDivElement | null>(null)
const scale = ref(1)
const tx = ref(0)
const ty = ref(0)
const rotation = ref(0)
const failed = ref(false)
const autoFit = ref(true)
const natural = ref({ width: 0, height: 0 })

const MIN_SCALE = 0.05
const MAX_SCALE = 8
const FIT_RATIO = 0.94

let observer: ResizeObserver | null = null
let dragOrigin: { x: number; y: number; tx: number; ty: number } | null = null

const rotationKey = computed(() => `harmony:recognize:rotate:${props.taskId || 'draft'}`)

const imageStyle = computed(() => ({
  transform: `translate(${tx.value}px, ${ty.value}px) scale(${scale.value}) rotate(${rotation.value}deg)`,
}))

const fitScale = computed(() => {
  const element = stage.value
  if (!element || !natural.value.width || !natural.value.height) return 1
  const rect = element.getBoundingClientRect()
  if (!rect.width || !rect.height) return 1
  const swap = rotation.value % 180 !== 0
  const width = swap ? natural.value.height : natural.value.width
  const height = swap ? natural.value.width : natural.value.height
  return Math.min(rect.width / width, rect.height / height) * FIT_RATIO
})

function clamp(value: number): number {
  return Math.min(Math.max(value, MIN_SCALE), MAX_SCALE)
}

/** 适应窗口：居中并缩放到完整可见。 */
function fit(): void {
  scale.value = fitScale.value
  tx.value = 0
  ty.value = 0
  autoFit.value = true
}

function zoomAt(clientX: number, clientY: number, factor: number): void {
  const element = stage.value
  if (!element) return
  const rect = element.getBoundingClientRect()
  const centerX = rect.left + rect.width / 2
  const centerY = rect.top + rect.height / 2
  const next = clamp(scale.value * factor)
  if (next === scale.value) return
  const ratio = next / scale.value
  const mx = clientX - centerX
  const my = clientY - centerY
  tx.value = mx - (mx - tx.value) * ratio
  ty.value = my - (my - ty.value) * ratio
  scale.value = next
  autoFit.value = false
}

function zoomCenter(factor: number): void {
  const element = stage.value
  if (!element) return
  const rect = element.getBoundingClientRect()
  zoomAt(rect.left + rect.width / 2, rect.top + rect.height / 2, factor)
}

function zoomIn(): void {
  zoomCenter(1.2)
}

function zoomOut(): void {
  zoomCenter(1 / 1.2)
}

/** 双击在“适应窗口”和“100%”之间切换。 */
function toggleZoom(): void {
  if (!natural.value.width) return
  const nearFit = fitScale.value > 0 && Math.abs(scale.value - fitScale.value) / fitScale.value < 0.05
  if (nearFit) {
    scale.value = 1
    tx.value = 0
    ty.value = 0
    autoFit.value = false
    return
  }
  fit()
}

function persistRotation(): void {
  try {
    localStorage.setItem(rotationKey.value, String(rotation.value))
  } catch {
    // 隐私模式下 localStorage 不可写，旋转只保留在本次页面会话里。
  }
}

function loadRotation(): void {
  try {
    const saved = Number(localStorage.getItem(rotationKey.value) || 0)
    rotation.value = [0, 90, 180, 270].includes(saved) ? saved : 0
  } catch {
    rotation.value = 0
  }
}

function rotate(): void {
  rotation.value = (rotation.value + 90) % 360
  persistRotation()
  fit()
}

function onWheel(event: WheelEvent): void {
  zoomAt(event.clientX, event.clientY, event.deltaY < 0 ? 1.12 : 1 / 1.12)
}

function onPointerDown(event: PointerEvent): void {
  if (event.button !== 0) return
  dragOrigin = { x: event.clientX, y: event.clientY, tx: tx.value, ty: ty.value }
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}

function onPointerMove(event: PointerEvent): void {
  if (!dragOrigin) return
  tx.value = dragOrigin.tx + (event.clientX - dragOrigin.x)
  ty.value = dragOrigin.ty + (event.clientY - dragOrigin.y)
  autoFit.value = false
}

function onPointerUp(event: PointerEvent): void {
  if (!dragOrigin) return
  dragOrigin = null
  const target = event.currentTarget as HTMLElement
  if (target.hasPointerCapture(event.pointerId)) {
    target.releasePointerCapture(event.pointerId)
  }
}

function onImageLoad(event: Event): void {
  const image = event.target as HTMLImageElement
  natural.value = { width: image.naturalWidth, height: image.naturalHeight }
  failed.value = false
  fit()
}

watch(
  () => props.src,
  () => {
    failed.value = false
    natural.value = { width: 0, height: 0 }
    scale.value = 1
    tx.value = 0
    ty.value = 0
    autoFit.value = true
  },
)

watch(
  () => props.taskId,
  () => {
    loadRotation()
    fit()
  },
)

onMounted(() => {
  loadRotation()
  if (typeof ResizeObserver !== 'undefined' && stage.value) {
    observer = new ResizeObserver(() => {
      if (autoFit.value) fit()
    })
    observer.observe(stage.value)
  }
})

onUnmounted(() => {
  observer?.disconnect()
  observer = null
})

defineExpose({ zoomIn, zoomOut, fit, rotate, toggleZoom })
</script>

<template>
  <div class="viewer" :class="{ 'viewer--overlay': overlay }">
    <div class="viewer__bar">
      <text v-if="overlay" class="viewer__close" @click="emit('close')">关闭</text>
      <text v-else class="viewer__label">原图</text>
      <view class="viewer__tools">
        <text class="viewer__tool" @click="zoomOut">－</text>
        <text class="viewer__tool" @click="zoomIn">＋</text>
        <text class="viewer__tool" @click="fit">适应</text>
        <text class="viewer__tool" @click="rotate">⟲ 旋转</text>
      </view>
    </div>
    <div
      ref="stage"
      class="viewer__stage"
      @wheel.prevent="onWheel"
      @dblclick="toggleZoom"
      @pointerdown="onPointerDown"
      @pointermove="onPointerMove"
      @pointerup="onPointerUp"
      @pointercancel="onPointerUp"
    >
      <img
        v-if="src && !failed"
        class="viewer__image"
        :src="src"
        :style="imageStyle"
        draggable="false"
        @load="onImageLoad"
        @error="failed = true"
      />
      <text v-else-if="failed" class="viewer__tip">原图加载失败，签名链接可能已过期</text>
      <text v-else class="viewer__tip">暂无原图</text>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.viewer {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 240px;
  border-radius: var(--r-m);
  background: var(--c-ink-1);
  overflow: hidden;
}

.viewer--overlay {
  position: fixed;
  inset: 0;
  z-index: 60;
  height: 100vh;
  border-radius: 0;
  background: var(--c-mask);
}

.viewer__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  height: 40px;
  padding: 0 10px;
  background: rgba(34, 28, 24, 0.85);
  color: rgba(255, 255, 255, 0.86);
  flex: none;
}

.viewer__label {
  font-size: 12px;
}

.viewer__close {
  font-size: 14px;
  color: var(--c-surface);
  padding: 4px 6px;
}

.viewer__tools {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
}

.viewer__tool {
  padding: 4px 8px;
  border-radius: var(--r-xs);
  font-size: 12px;
  color: rgba(255, 255, 255, 0.9);
  background: rgba(255, 255, 255, 0.12);
}

.viewer__stage {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 200px;
  overflow: hidden;
  cursor: grab;
  touch-action: none;
  background: #1b1917;
}

.viewer__stage:active {
  cursor: grabbing;
}

.viewer__image {
  display: block;
  max-width: none;
  max-height: none;
  transform-origin: center center;
  user-select: none;
  -webkit-user-drag: none;
  will-change: transform;
}

.viewer__tip {
  padding: 0 20px;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.7);
  text-align: center;
}
</style>
