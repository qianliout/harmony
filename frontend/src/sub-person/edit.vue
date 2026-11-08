<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { createPerson, deletePerson, fetchPersonDetail, updatePerson } from '@/api/person'
import AppNav from '@/components/AppNav.vue'
import ConfirmSheet from '@/components/ConfirmSheet.vue'
import ListCell from '@/components/ListCell.vue'
import Skeleton from '@/components/Skeleton.vue'
import { submitWithConfirm } from '@/composables/useConfirmSubmit'
import { relations } from '@/styles/palette'
import { themeVars } from '@/styles/theme'
import { back, confirmModal, toast } from '@/utils/nav'

const personId = ref(0)
const isEdit = computed(() => personId.value > 0)

const form = ref({
  name: '',
  relation: 'other',
  phone: '',
  remark: '',
})

const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)
const error = ref('')

async function loadDetail(): Promise<void> {
  if (!personId.value) return
  loading.value = true
  error.value = ''
  try {
    const detail = await fetchPersonDetail(personId.value)
    form.value = {
      name: detail.name,
      relation: detail.relation || 'other',
      phone: detail.phone,
      remark: detail.remark,
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
    toast(error.value)
  } finally {
    loading.value = false
  }
}

async function save(): Promise<void> {
  if (saving.value) return
  const name = form.value.name.trim()
  if (!name) {
    toast('请填写姓名')
    return
  }
  saving.value = true
  try {
    const payload = {
      name,
      relation: form.value.relation || 'other',
      phone: form.value.phone.trim(),
      remark: form.value.remark.trim(),
    }
    const saved = await submitWithConfirm((confirm) =>
      personId.value ? updatePerson(personId.value, { ...payload, confirm }) : createPerson({ ...payload, confirm }),
    )
    if (!saved) return
    toast('已保存', 'success')
    back()
  } catch (err) {
    toast(err instanceof Error ? err.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function remove(): Promise<void> {
  if (deleting.value) return
  const ok = await confirmModal('删除后，这位亲友的资料将不再保留。', '删除亲友')
  if (!ok) return
  deleting.value = true
  try {
    await deletePerson(personId.value)
    toast('已删除', 'success')
    back(2)
  } catch (err) {
    toast(err instanceof Error ? err.message : '删除失败')
  } finally {
    deleting.value = false
  }
}

onLoad((query) => {
  personId.value = Number(query?.id ?? 0)
  if (personId.value) {
    loadDetail()
  }
})
</script>

<template>
  <wd-config-provider :theme-vars="themeVars">
    <view class="screen screen--form">
      <AppNav :title="isEdit ? '编辑亲友' : '新建亲友'" />

      <view class="page">
        <Skeleton v-if="loading" type="list" />

        <view v-else-if="error" class="load-error">
          <text class="load-error__text">{{ error }}</text>
          <view class="btn btn--secondary btn--small" @click="loadDetail">重新加载</view>
        </view>

        <template v-else>
          <view class="form-card">
            <view class="field">
              <text class="field__label">姓名</text>
              <input
                v-model="form.name"
                class="field__input"
                maxlength="32"
                placeholder="怎么称呼"
                placeholder-class="input-placeholder"
              />
            </view>
            <view class="field">
              <text class="field__label">电话</text>
              <input
                v-model="form.phone"
                class="field__input"
                maxlength="30"
                placeholder="选填，方便联系"
                placeholder-class="input-placeholder"
              />
            </view>
          </view>

          <view class="form-card relation-card">
            <text class="relation-card__label">关系</text>
            <view class="relation-card__chips">
              <text
                v-for="item in relations"
                :key="item.code"
                class="chip"
                :class="{ 'chip--on': form.relation === item.code }"
                @click="form.relation = item.code"
              >
                {{ item.label }}
              </text>
            </view>
          </view>

          <view class="form-card">
            <view class="field field--stack">
              <text class="field__label">备注</text>
              <textarea
                v-model="form.remark"
                class="textarea"
                maxlength="200"
                placeholder="记下称呼、住址之类"
                placeholder-class="input-placeholder"
              />
            </view>
          </view>

          <view v-if="isEdit" class="card delete-card">
            <ListCell title="删除这位亲友" :arrow="false" danger @click="remove" />
          </view>
        </template>
      </view>

      <view v-if="!loading && !error" class="bottom-bar">
        <view class="btn btn--primary bottom-bar__btn" :class="{ 'btn--disabled': saving }" @click="save">
          {{ saving ? '保存中…' : '保存' }}
        </view>
      </view>

      <ConfirmSheet />
    </view>
  </wd-config-provider>
</template>

<style lang="scss" scoped>
.screen--form {
  padding-bottom: calc(88px + env(safe-area-inset-bottom));
}

.load-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 40px 16px;
}

.load-error__text {
  font-size: 13px;
  color: var(--c-ink-3);
}

.relation-card {
  padding-bottom: 16px;
}

.relation-card__label {
  display: block;
  padding: 12px 16px 8px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.relation-card__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 0 16px;
}

.field--stack {
  align-items: flex-start;
  padding-top: 12px;
  padding-bottom: 12px;
}

.textarea {
  flex: 1;
  width: auto;
  height: 72px;
  font-size: 15px;
  line-height: 22px;
  color: var(--c-ink-1);
}

.delete-card {
  margin-top: 20px;
  overflow: hidden;
}
</style>
