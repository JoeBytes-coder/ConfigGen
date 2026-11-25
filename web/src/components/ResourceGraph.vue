<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { DataSet } from 'vis-data'
import { Network } from 'vis-network'

import type { PlanEdge, PlanNode } from '@/api/types'
import { CATEGORY_COLORS } from '@/constants/k8s'
import { useMetaStore } from '@/stores/meta'

const props = defineProps<{ nodes: PlanNode[]; edges: PlanEdge[] }>()
const container = ref<HTMLDivElement | null>(null)
const meta = useMetaStore()

let network: Network | null = null

function buildData() {
  const present = new Set(props.nodes.map((node) => node.kind))
  const nodes = new DataSet(
    props.nodes.map((node) => {
      const category = meta.kindToMeta.get(node.kind)?.category
      return {
        id: node.kind,
        label: node.kind + '\n' + node.name,
        shape: 'box',
        margin: 10,
        color: {
          background: CATEGORY_COLORS[category ?? ''] ?? '#E7F0FA',
          border: node.auto_added ? '#D99A4E' : '#6E9CD3',
          highlight: { background: '#DCEAF8', border: '#3D6EA8' },
        },
        font: { color: '#2C3E5D', size: 12, face: 'Inter, sans-serif' },
        title: node.auto_added ? '自动补充的资源' : node.name,
      }
    }),
  )
  const edges = new DataSet(
    props.edges
      .filter((edge) => present.has(edge.from) && present.has(edge.to))
      .map((edge, index) => ({
        id: index,
        from: edge.from,
        to: edge.to,
        label: edge.relation,
        arrows: 'to',
        color: { color: '#A8BCD5' },
        font: { size: 10, color: '#8296B3', align: 'middle' },
        smooth: { enabled: true, type: 'dynamic' },
      })),
  )
  return { nodes, edges }
}

function render() {
  if (!container.value) return
  const data = buildData()
  if (network) {
    network.setData(data)
    return
  }
  network = new Network(container.value, data, {
    height: '280px',
    physics: { stabilization: true, barnesHut: { gravitationalConstant: -6000, springLength: 140 } },
    interaction: { hover: true, dragNodes: true },
    layout: { improvedLayout: true },
  })
}

onMounted(render)
watch(() => [props.nodes, props.edges], render, { deep: true })
onBeforeUnmount(() => {
  network?.destroy()
  network = null
})
</script>

<template>
  <div ref="container" class="graph-container"></div>
</template>

<style scoped>
.graph-container {
  width: 100%;
  height: 280px;
  background: #fbfdff;
  border: 1px solid var(--cg-border);
  border-radius: 8px;
}
</style>
