import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'generator',
      component: () => import('@/views/GeneratorView.vue'),
      meta: { title: '生成配置' },
    },
    {
      path: '/history',
      name: 'history',
      component: () => import('@/views/HistoryView.vue'),
      meta: { title: '历史记录' },
    },
    {
      path: '/history/:id(\\d+)',
      name: 'record-detail',
      component: () => import('@/views/RecordDetailView.vue'),
      meta: { title: '记录详情' },
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
  scrollBehavior: () => ({ top: 0 }),
})

export default router
