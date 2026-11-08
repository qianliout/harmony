<script setup lang="ts">
import { back as goBack } from '@/utils/nav'

withDefaults(
  defineProps<{
    title?: string
    back?: boolean
    right?: string
    transparent?: boolean
  }>(),
  { title: '', back: true, right: '', transparent: false },
)

defineEmits<{ right: [] }>()

const statusBarHeight = uni.getSystemInfoSync().statusBarHeight ?? 0
</script>

<template>
  <view class="nav" :class="{ 'nav--transparent': transparent }">
    <view class="nav__status" :style="{ height: `${statusBarHeight}px` }" />
    <view class="nav__bar">
      <view class="nav__side" @click="goBack()">
        <text v-if="back" class="nav__back">‹</text>
      </view>
      <text class="nav__title">{{ title }}</text>
      <view class="nav__side nav__side--right" @click="$emit('right')">
        <text v-if="right" class="nav__right">{{ right }}</text>
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.nav {
  position: relative;
  background: var(--c-bg);
}

.nav--transparent {
  background: transparent;
}

.nav__bar {
  position: relative;
  display: flex;
  align-items: center;
  height: 44px;
  padding: 0 var(--gutter);
}

.nav__side {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  width: 64px;
  min-height: 44px;
}

.nav__side--right {
  justify-content: flex-end;
}

.nav__back {
  font-size: 30px;
  line-height: 1;
  color: var(--c-ink-1);
  padding-right: 12px;
}

.nav__title {
  flex: 1;
  text-align: center;
  font-size: 17px;
  line-height: 24px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.nav__right {
  font-size: 15px;
  color: var(--c-primary);
  font-weight: 500;
}
</style>
