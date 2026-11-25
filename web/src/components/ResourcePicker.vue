<script setup lang="ts">
import { computed } from 'vue'
import { NAlert, NButton, NSpin, NTag, NText } from 'naive-ui'

import { CATEGORY_LABELS } from '@/constants/k8s'
import { useMetaStore } from '@/stores/meta'

const props = defineProps<{ modelValue: string[] }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: string[]): void }>()

const meta = useMetaStore()
const total = computed(() => meta.kinds.length)

function toggle(kind: string) {
  const next = props.modelValue.includes(kind)
    ? props.modelValue.filter((item) => item !== kind)
    : [...props.modelValue, kind]
  emit('update:modelValue', next)
}

function selectAll() {
  emit('update:modelValue', meta.kinds.map((item) => item.kind))
}

function clear() {
  emit('update:modelValue', [])
}

function toggleGroup(kinds: string[], allSelected: boolean) {
  const next = allSelected
    ? props.modelValue.filter((kind) => !kinds.includes(kind))
    : [...new Set([...props.modelValue, ...kinds])]
  emit('update:modelValue', next)
}
</script>

<template>
  <div>
    <n-alert v-if="meta.error" type="error" :bordered="false" title="资源目录加载失败">
      <div class="alert-body">
        <span>{{ meta.error }}</span>
        <n-button size="tiny" @click="meta.fetchMeta()">重试</n-button>
      </div>
    </n-alert>

    <n-spin v-else-if="!meta.loaded" size="small" style="width: 100%; padding: 16px 0">
      <n-text depth="3" style="font-size: 12px">正在加载资源目录…</n-text>
    </n-spin>

    <template v-else>
      <div class="resource-actions">
        <n-button size="tiny" quaternary type="primary" @click="selectAll">全选</n-button>
        <n-button size="tiny" quaternary @click="clear">清空</n-button>
        <span class="picked-count">已选 {{ modelValue.length }} / {{ total }}</span>
      </div>

      <div v-for="(group, category) in meta.byCategory" :key="category" class="resource-group">
        <div class="group-title">
          <span>{{ CATEGORY_LABELS[category] ?? category }}</span>
          <n-button
            size="tiny"
            quaternary
            @click="toggleGroup(group.map((item) => item.kind), group.every((item) => modelValue.includes(item.kind)))"
          >
            {{ group.every((item) => modelValue.includes(item.kind)) ? '取消全组' : '全组' }}
          </n-button>
        </div>
        <div class="resource-tags">
          <n-tag
            v-for="item in group"
            :key="item.kind"
            checkable
            :checked="modelValue.includes(item.kind)"
            size="small"
            @update:checked="() => toggle(item.kind)"
          >
            {{ item.kind }}
          </n-tag>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.alert-body {
  display: flex;
  align-items: center;
  gap: 8px;
}
.resource-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.picked-count {
  font-size: 12px;
  color: var(--cg-text-muted);
  margin-left: auto;
}
.resource-group {
  margin-bottom: 10px;
}
.group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--cg-text-muted);
  margin-bottom: 6px;
}
.resource-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
