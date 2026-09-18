<script setup lang="ts">
import { NDynamicInput, NText } from 'naive-ui'

import type { EnvPair } from '@/api/types'

defineProps<{ value: EnvPair[] }>()
const emit = defineEmits<{ (e: 'update:value', v: EnvPair[]): void }>()

function onUpdate(v: EnvPair[]) {
  emit('update:value', v)
}
</script>

<template>
  <div>
    <n-dynamic-input :value="value" preset="pair" key-placeholder="KEY" value-placeholder="VALUE" @update:value="onUpdate" />
    <n-text depth="3" style="font-size: 12px">可选，按 KEY = VALUE 注入环境变量，生成时按键名排序</n-text>
  </div>
</template>
