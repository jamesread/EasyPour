<template>
  <main class="orders-page">
    <section class="padding">
      <div class="orders-header">
        <h2>Orders</h2>
        <button @click="$router.push('/')" aria-label="Close">Close</button>
      </div>

      <p class="orders-intro">Recent order IDs stored on this device.</p>

      <div v-if="orderGroups.length === 0" class="empty-orders">
        <p>No orders yet.</p>
        <button @click="$router.push('/')">Browse Menu</button>
      </div>

      <ul v-else class="orders-list">
        <li
          v-for="group in orderGroups"
          :key="group.groupId"
          class="order-row"
          @click="$router.push({ name: 'OrderStatus', params: { orderId: group.groupId } })"
        >
          <span class="order-id">Order #{{ displayGroupId(group.groupId) }}</span>
          <span class="order-meta">{{ itemCountLabel(group) }}</span>
          <span class="order-status">{{ group.status }}</span>
          <span class="order-date">{{ formatDate(group.createdAt) }}</span>
        </li>
      </ul>
    </section>
  </main>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useRecentOrders } from '../composables/useRecentOrders'

const { recentOrders, getRecentOrders } = useRecentOrders()

const orderGroups = computed(() =>
  recentOrders.value.filter((g) => g && g.groupId && Array.isArray(g.items))
)

onMounted(() => {
  getRecentOrders()
})

function displayGroupId(id) {
  if (!id) return '—'
  const s = String(id)
  return s.length > 8 ? s.slice(0, 8) : s
}

function itemCountLabel(group) {
  const n = Array.isArray(group?.items) ? group.items.length : 0
  return n === 1 ? '1 item' : `${n} items`
}

function formatDate(ts) {
  if (ts == null) return '—'
  const d = new Date(typeof ts === 'number' && ts < 1e12 ? ts * 1000 : ts)
  return d.toLocaleString()
}
</script>

<style scoped>
.orders-page {
  min-height: 100vh;
}

.orders-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.orders-intro {
  color: #666;
  margin-bottom: 1.5rem;
  font-size: 0.95rem;
}

.empty-orders {
  padding: 3rem 2rem;
  text-align: center;
}

.empty-orders p {
  margin-bottom: 1rem;
  font-size: 1.1rem;
}

.orders-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.order-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem 1rem;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #e0e0e0;
  cursor: pointer;
}

.order-row:hover {
  background-color: #f5f5f5;
}

.order-id {
  font-weight: 500;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.order-meta {
  flex-shrink: 0;
  font-size: 0.9rem;
  color: #64748b;
}

.order-status {
  flex-shrink: 0;
  color: #555;
  font-size: 0.9rem;
}

.order-date {
  flex-shrink: 0;
  margin-left: auto;
  color: #777;
  font-size: 0.85rem;
}
</style>
