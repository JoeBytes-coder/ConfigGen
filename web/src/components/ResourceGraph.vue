<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { DataSet } from 'vis-data'
import { Network } from 'vis-network'

import type { ResourceEdge, ResourceNode } from '@/api/types'
import { KIND_COLORS } from '@/constants/k8s'

const props = withDefaults(
  defineProps<{
    nodes: ResourceNode[]
    edges: ResourceEdge[]
    height?: number
  }>(),
  { height: 420 },
)

const container = ref<HTMLElement | null>(null)
let network: Network | null = null

function render() {
  if (!container.value || props.nodes.length === 0) return

  const kindToId = new Map<string, number>()
  const visNodes = props.nodes.map((n, i) => {
    kindToId.set(n.kind, i + 1)
    return {
      id: i + 1,
      label: `${n.kind}\n${n.name}`,
      title: `${n.kind}: ${n.name}`,
      shape: 'box',
      color: {
        background: KIND_COLORS[n.kind] ?? '#E8EEF6',
        border: '#6E9CD3',
        highlight: { background: KIND_COLORS[n.kind] ?? '#E8EEF6', border: '#7DAEE1' },
      },
      font: { multi: true, size: 13 },
      margin: 8,
    }
  })

  const visEdges = props.edges
    .map((e) => ({
      from: kindToId.get(e.source_kind),
      to: kindToId.get(e.target_kind),
      label: e.relation,
      arrows: 'to',
      font: { align: 'middle', size: 11, color: '#6B7F9C' },
      color: { color: '#93A9C4', highlight: '#7DAEE1' },
    }))
    .filter((e) => e.from !== undefined && e.to !== undefined)

  network?.destroy()
  network = new Network(
    container.value,
    { nodes: new DataSet(visNodes), edges: new DataSet(visEdges) },
    {
      layout: { hierarchical: { direction: 'LR', sortMethod: 'directed', levelSeparation: 110 } },
      physics: { enabled: false },
      interaction: { hover: true },
      edges: { smooth: { type: 'curvedCW', roundness: 0.2 } },
    },
  )
}

watch(() => [props.nodes, props.edges], render)
onMounted(render)
onBeforeUnmount(() => {
  network?.destroy()
  network = null
})
</script>

<template>
  <div
    ref="container"
    class="graph-container"
    :style="{ height: `${height}px` }"
  />
</template>
