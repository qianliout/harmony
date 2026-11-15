<script setup lang="ts">
import { useWebUiStore } from '@/stores/web'
import WebIcon from '@/web/components/WebIcon.vue'

const ui = useWebUiStore()
</script>

<template>
  <div v-if="ui.toastMessage" class="web-toast" role="status">
    <WebIcon name="check" :size="16" />
    {{ ui.toastMessage }}
  </div>
</template>
