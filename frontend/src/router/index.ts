import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/views/RegisterView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('@/views/AppLayout.vue'),
      children: [
        { path: '', name: 'dashboard', component: () => import('@/views/DashboardView.vue') },
        { path: 'bills', name: 'bills', component: () => import('@/views/BillsView.vue') },
        { path: 'expenses', name: 'expenses', component: () => import('@/views/ExpensesView.vue') },
        { path: 'providers', name: 'providers', component: () => import('@/views/ProvidersView.vue') },
        { path: 'credentials', name: 'credentials', component: () => import('@/views/CredentialsView.vue') },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const authStore = useAuthStore()

  // Wait for Firebase to restore auth state on first load
  if (authStore.loading) {
    await new Promise<void>((resolve) => {
      const stop = authStore.$subscribe(() => {
        if (!authStore.loading) { stop(); resolve() }
      })
      // Also resolve if already done by the time we get here
      if (!authStore.loading) { stop(); resolve() }
    })
  }

  if (!to.meta.public && !authStore.isAuthenticated) return { name: 'login' }
  if (to.meta.public && authStore.isAuthenticated) return { name: 'dashboard' }
})

export default router
