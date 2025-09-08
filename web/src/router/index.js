import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import HostDashboard from '@/components/views/HostDashboard.vue'
import VmView from '@/components/views/VmView.vue'
import Datacenter from '@/components/views/Datacenter.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      component: HomeView,
      children: [
        {
          path: '',
          name: 'datacenter',
          component: Datacenter, // Default view is now the datacenter
        },
        {
          path: 'hosts/:hostId',
          name: 'host-dashboard',
          component: HostDashboard,
          props: true,
        },
        {
          path: 'vms/:vmName',
          name: 'vm-view',
          component: VmView,
          props: true,
        },
      ],
    },
  ]
})

export default router

