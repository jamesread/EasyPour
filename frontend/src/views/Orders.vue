<template>
  <main class="orders-page">
    <section class="padding">
      <div class="orders-header">
        <h2>My Orders</h2>
        <button type="button" class="neutral" @click="$router.push('/')" aria-label="Close">Close</button>
      </div>

      <p class="orders-intro">Your recent orders.</p>

      <div v-if="loadError" class="orders-error" role="alert">
        {{ loadError }}
      </div>
      <div v-else-if="loading" class="orders-loading" aria-live="polite">
        <p>Loading orders…</p>
      </div>
      <div v-else-if="orderGroups.length === 0" class="empty-orders">
        <p>No orders yet.</p>
        <button type="button" class="neutral" @click="$router.push('/')">Browse Menu</button>
      </div>

      <div v-else class="orders-table-wrap">
        <table class="orders-table" role="grid">
          <thead>
            <tr>
              <th>Order ID</th>
              <th>Items</th>
              <th>Status</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="group in orderGroups"
              :key="group.groupId"
              class="order-row"
              @click="$router.push({ name: 'OrderStatus', params: { orderId: group.groupId } })"
            >
              <td>{{ displayGroupId(group.groupId) }}</td>
              <td>{{ itemCountLabel(group) }}</td>
              <td>
                <span class="tag" :class="statusKarmaClass(group.status)">{{ group.status }}</span>
              </td>
              <td>{{ formatDate(group.createdAt) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </main>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useCurrentUser } from '../composables/useCurrentUser'
import { useOrderSync } from '../composables/useOrderSync.js'
import { groupOrders } from '../composables/useRecentOrders.js'

const { username } = useCurrentUser()
const { orders, refresh } = useOrderSync()

const loading = ref(true)
const loadError = ref(null)

const myOrders = computed(() => {
  const list = orders.value ?? []
  const me = username.value
  if (!me) return []
  return list.filter((o) => !o?.username || o.username === me)
})

const orderGroups = computed(() => groupOrders(myOrders.value))

function displayGroupId(id) {
  if (!id) return '—'
  const s = String(id)
  return s.length > 8 ? s.slice(0, 8) : s
}

function itemCountLabel(group) {
  const n = Array.isArray(group?.items) ? group.items.length : 0
  return n === 1 ? '1 item' : `${n} items`
}

function statusKarmaClass(status) {
  switch ((status ?? '').toLowerCase()) {
    case 'pending':
      return 'fg-warning'
    case 'preparing':
      return 'fg-note'
    case 'ready':
    case 'delivered':
    case 'completed':
    case 'done':
      return 'fg-good'
    default:
      return 'fg-info'
  }
}

function formatDate(ts) {
  if (ts == null) return '—'
  const n = typeof ts === 'bigint' ? Number(ts) : Number(ts ?? 0)
  const d = new Date(n < 1e12 ? n * 1000 : n)
  return d.toLocaleString()
}

onMounted(async () => {
  try {
    await refresh()
  } catch (e) {
    loadError.value = e?.message ?? 'Failed to load orders'
  } finally {
    loading.value = false
  }
})
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

.orders-error {
  padding: 1rem;
  background: #fef2f2;
  color: #991b1b;
  border-radius: 0.5rem;
  margin-bottom: 1rem;
}

.orders-loading,
.empty-orders {
  padding: 2rem;
  text-align: center;
  color: #64748b;
}

.empty-orders p {
  margin-bottom: 1rem;
  font-size: 1.1rem;
}

.orders-table-wrap {
  overflow-x: auto;
}

.orders-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
}

.orders-table th,
.orders-table td {
  padding: 0.75rem;
  text-align: left;
  border-bottom: 1px solid #e2e8f0;
}

.orders-table th {
  font-weight: 600;
  color: #0f172a;
  background: #f8fafc;
}

.orders-table td {
  color: #475569;
}

.order-row {
  cursor: pointer;
}

.order-row:hover {
  background-color: #f5f5f5;
}
</style>
