import { createRouter, createWebHistory } from 'vue-router'
import { CookingPotIcon, Invoice01Icon, Notification03Icon, Configuration01Icon, ShoppingCart01Icon, SoftDrink01Icon } from '@hugeicons/core-free-icons'
import Menu from '../views/Menu.vue'
import Basket from '../views/Basket.vue'
import UserProfile from '../views/UserProfile.vue'
import Orders from '../views/Orders.vue'
import AdminOrders from '../views/AdminOrders.vue'
import AdminApprise from '../views/AdminApprise.vue'
import AdminKitchen from '../views/AdminKitchen.vue'
import SettingsAdmin from '../views/SettingsAdmin.vue'
import OrderStatus from '../views/OrderStatus.vue'

const routes = [
  {
    path: '/',
    name: 'Menu',
    component: Menu,
    meta: { title: 'Menu', icon: SoftDrink01Icon },
  },
  {
    path: '/login',
    name: 'Login',
    component: Menu,
  },
  {
    path: '/basket',
    name: 'Basket',
    component: Basket,
    meta: { title: 'Basket', icon: ShoppingCart01Icon },
  },
  {
    path: '/orders',
    name: 'Orders',
    component: Orders,
    meta: { title: 'My Orders', icon: Invoice01Icon },
  },
  {
    path: '/orders/status/:orderId',
    name: 'OrderStatus',
    component: OrderStatus,
  },
  {
    path: '/admin/orders',
    name: 'AdminOrders',
    component: AdminOrders,
    meta: { title: 'Admin Orders', icon: Invoice01Icon, description: 'Acknowledge and manage all orders' },
  },
  {
    path: '/admin/kitchen',
    name: 'AdminKitchen',
    component: AdminKitchen,
    meta: { title: 'Kitchen View', icon: CookingPotIcon, description: 'Item counts from open orders' },
  },
  {
    path: '/admin/apprise',
    name: 'AdminApprise',
    component: AdminApprise,
    meta: { title: 'Apprise', icon: Notification03Icon, description: 'Configure notifications for new orders' },
  },
  {
    path: '/admin/settings',
    name: 'SettingsAdmin',
    component: SettingsAdmin,
    meta: { title: 'Settings', icon: Configuration01Icon, description: 'Site-wide configuration variables' },
  },
  {
    path: '/profile',
    name: 'UserProfile',
    component: UserProfile,
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

export default router
