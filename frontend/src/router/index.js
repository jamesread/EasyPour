import { createRouter, createWebHistory } from 'vue-router'
import Menu from '../views/Menu.vue'
import Basket from '../views/Basket.vue'
import UserProfile from '../views/UserProfile.vue'
import Orders from '../views/Orders.vue'
import OrderStatus from '../views/OrderStatus.vue'

const routes = [
  {
    path: '/',
    name: 'Menu',
    component: Menu,
  },
  {
    path: '/basket',
    name: 'Basket',
    component: Basket,
  },
  {
    path: '/orders',
    name: 'Orders',
    component: Orders,
  },
  {
    path: '/orders/status/:orderId',
    name: 'OrderStatus',
    component: OrderStatus,
  },
  {
    path: '/profile',
    name: 'UserProfile',
    component: UserProfile,
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router


