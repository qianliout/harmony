<script setup lang="ts">
import { computed } from 'vue'

import { useWebUiStore } from '@/stores/web'
import WebIcon from '@/web/components/WebIcon.vue'

const ui = useWebUiStore()

const notes: Record<string, string[]> = {
  home: [
    '朱砂卡只显示当年收、送、净差额；未回礼胶囊进入回礼清单。',
    '右侧四个快捷入口：记一笔、连续录入、识别礼簿、导入导出。',
    '最近记录点行打开右侧详情抽屉；点姓名进入亲友详情。',
  ],
  records: [
    '筛选从左到右按主维度排列：礼簿类别、事项类型、亲友、时间；竖线后是方向、回礼状态、名目。',
    '只选回礼状态、不选方向时，按收礼记录处理。',
    '底部汇总来自筛选结果全量，不只是当前页。',
  ],
  people: ['搜索支持姓名与拼音首字母（如 zjg）。', '有未回礼的人，行首有琥珀色竖条。'],
  person: [
    '四个数字：总收、总送、净差额、未回礼。',
    '时间线按事项分组，组内跨礼簿、跨名目合计。',
    '勾选未回礼的收礼记录后，顶部“回礼”可用。',
  ],
  events: ['按日期倒序；自家办看收，别人家办看送。'],
  event: ['谁来了 / 我回了 / 还有谁没回，都按人合计。'],
  books: ['已合并的礼簿单独列出，只读；合并跨类别时名目不变。'],
  book: [
    '四个 tab：记录、连续录入、按页核对、整理纸本标注。',
    '连续录入按 Enter 记下，光标回到姓名；本页合计实时对比。',
  ],
  notes: ['列出抄了标注但还没转为回礼的收礼记录。', '一条标注可拆成现金与火炮多条送礼记录。'],
  pages: ['跨礼簿对比每页纸本合计与录入合计。'],
  repay: ['我该回的按人合计，最久远的在前。', '该回我的只算主动送礼，不含回礼生成记录。'],
  stats: ['年份切换在右上；全部年份隐藏按月图。', '合计 = 现金 + 礼品估价。'],
  recognize: [
    '一次一张：上传 → 识别 → 核对 → 写入。',
    '标黄行必须勾选确认后才能写入。',
    '左原图右表格，鼠标停在右侧某行，左图对应列高亮。',
  ],
  settings: ['礼簿整理、导入导出、名目与类别、登录设备集中在这里。'],
}

const current = computed(() => {
  const path = getCurrentPages().slice(-1)[0]?.route ?? ''
  const key = Object.keys(notes).find((item) => path.endsWith(`/${item}`))
  return notes[key ?? 'home'] ?? notes.home
})
</script>

<template>
  <aside class="web-drawer web-drawer--notes" role="complementary">
    <div class="web-drawer__head">
      <h3>交互说明</h3>
      <button class="web-btn web-btn--text" type="button" aria-label="关闭" @click="ui.close()">
        <WebIcon name="x" />
      </button>
    </div>
    <div class="web-drawer__body">
      <ol>
        <li v-for="item in current" :key="item">{{ item }}</li>
      </ol>
      <div class="web-field" style="margin-top: 16px">
        <label>全局快捷键</label>
        <div class="web-dl">
          <dt><kbd class="web-kbd">⌘K</kbd></dt>
          <dd>全局搜索（也可按 /）</dd>
          <dt><kbd class="web-kbd">N</kbd></dt>
          <dd>记一笔</dd>
          <dt><kbd class="web-kbd">Esc</kbd></dt>
          <dd>关闭抽屉、对话框</dd>
        </div>
      </div>
    </div>
  </aside>
</template>

<style scoped>
ol {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin: 0;
  padding-left: 20px;
  color: var(--c-ink-2);
  line-height: 22px;
}
</style>
