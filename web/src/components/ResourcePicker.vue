<script setup lang="ts">
import { NButton, NTag } from 'naive-ui'

import { K8S_KINDS } from '@/constants/k8s'

const props = defineProps<{ modelValue: string[] }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: string[]): void }>()

function toggle(kind: string) {
  const next = props.modelValue.includes(kind)
    ? props.modelValue.filter((k) => k !== kind)
    : [...props.modelValue, kind]
  emit('update:modelValue', next)
}

function selectAll() {
  emit('update:modelValue', [...K8S_KINDS])
}

function clear() {
  emit('update:modelValue', [])
}
</script>

<template>
  <div>
    <div class="resource-actions">
      <n-button size="tiny" quaternary type="primary" @click="selectAll">全选</n-button>
      <n-button size="tiny" quaternary @click="clear">清空</n-button>
      <span class="picked-count">已选 {{ modelValue.length }} / {{ K8S_KINDS.length }}</span>
    </div>
    <div class="resource-tags">
      <n-tag
        v-for="kind in K8S_KINDS"
        :key="kind"
        checkable
        :checked="modelValue.includes(kind)"
        size="small"
        @update:checked="() => toggle(kind)"
      >
        {{ kind }}
      </n-tag>
    </div>
  </div>
</template>
