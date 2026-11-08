<script setup lang="ts">
import { computed } from 'vue'

import { confirmState, resolveConfirm } from '@/composables/useConfirmSubmit'

const visible = computed({
  get: () => confirmState.value.show,
  set: (value: boolean) => {
    if (!value) resolveConfirm(false)
  },
})

const payload = computed(() => confirmState.value.payload)
</script>

<template>
  <wd-popup
    v-model="visible"
    position="bottom"
    :safe-area-inset-bottom="true"
    custom-style="border-radius: 20px 20px 0 0;"
  >
    <view v-if="payload" class="confirm">
      <text class="confirm__title">{{ payload.title }}</text>
      <text class="confirm__message">{{ payload.message }}</text>
      <view v-if="payload.conflicts.length" class="confirm__list">
        <view v-for="conflict in payload.conflicts" :key="`${conflict.id ?? 0}-${conflict.summary}`" class="confirm__item">
          <text>{{ conflict.summary }}</text>
        </view>
      </view>
      <view class="confirm__actions">
        <view class="btn btn--secondary confirm__btn" @click="resolveConfirm(false)">取消</view>
        <view class="btn btn--primary confirm__btn" @click="resolveConfirm(true)">
          {{ payload.confirmText || '仍然保存' }}
        </view>
      </view>
    </view>
  </wd-popup>
</template>

<style lang="scss" scoped>
.confirm {
  padding: 20px 16px calc(16px + env(safe-area-inset-bottom));
}

.confirm__title {
  display: block;
  font-size: 17px;
  line-height: 24px;
  font-weight: 600;
}

.confirm__message {
  display: block;
  margin-top: 8px;
  font-size: 15px;
  line-height: 22px;
  color: var(--c-ink-2);
}

.confirm__list {
  margin-top: 12px;
  padding: 8px 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
}

.confirm__item {
  font-size: 13px;
  line-height: 20px;
  color: var(--c-ink-2);
}

.confirm__actions {
  display: flex;
  gap: 10px;
  margin-top: 18px;
}

.confirm__btn {
  flex: 1;
}
</style>
