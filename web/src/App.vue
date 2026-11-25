<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import {
  NConfigProvider,
  NGlobalStyle,
  NLayout,
  NLayoutContent,
  NLayoutHeader,
  NMessageProvider,
  dateZhCN,
  zhCN,
} from 'naive-ui'

import { themeOverrides } from '@/theme'
import { useMetaStore } from '@/stores/meta'

const route = useRoute()
const activeKey = computed(() => (route.path.startsWith('/history') ? 'history' : 'generator'))

// 应用挂载时拉取 K8s 资源目录(选择器分组/拓扑配色/replicas 门控依赖它)
onMounted(() => {
  useMetaStore().fetchMeta()
})
</script>

<template>
  <n-config-provider
    :theme-overrides="themeOverrides"
    :locale="zhCN"
    :date-locale="dateZhCN"
    inline-theme-disabled
  >
    <n-message-provider>
      <n-global-style />
      <n-layout class="app-shell">
        <n-layout-header bordered class="app-header">
          <div class="header-inner">
            <router-link to="/" class="brand">
              <span class="brand-logo">&#123;&#125;</span>
              <span>
                <span class="brand-name">ConfigGen</span>
                <div class="brand-sub">云原生配置生成器</div>
              </span>
            </router-link>
            <nav class="main-nav">
              <router-link to="/" :class="{ active: activeKey === 'generator' }">生成配置</router-link>
              <router-link to="/history" :class="{ active: activeKey === 'history' }">历史记录</router-link>
            </nav>
          </div>
        </n-layout-header>
        <n-layout-content class="app-content">
          <div class="page">
            <router-view />
          </div>
        </n-layout-content>
      </n-layout>
    </n-message-provider>
  </n-config-provider>
</template>

<style scoped>
.app-header {
  background: #fff;
}
</style>
