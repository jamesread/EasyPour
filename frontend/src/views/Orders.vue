<template>
  <main class="orders-page">
    <section class="padding">
      <div class="orders-header">
        <h2>Orders</h2>
        <button @click="goBack" aria-label="Back">{{ isAdmin ? 'Back' : 'Close' }}</button>
      </div>

      <p class="orders-intro">
        {{ isAdmin ? 'All orders. Acknowledge pending orders or mark as delivered.' : 'Your recent orders from the server.' }}
      </p>

      <div v-if="loadError" class="orders-error" role="alert">
        {{ loadError }}
      </div>
      <div v-else-if="loading" class="orders-loading" aria-live="polite">
        <p>Loading orders…</p>
      </div>
      <div v-else-if="!isAdmin && orderGroups.length === 0" class="empty-orders">
        <p>No orders yet.</p>
        <button @click="$router.push('/')">Browse Menu</button>
      </div>
      <div v-else-if="isAdmin && orders.length === 0" class="empty-orders">
        <p>No orders yet.</p>
      </div>

      <ul v-else-if="!isAdmin" class="orders-list">
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

      <div v-else class="orders-table-wrap">
        <table class="orders-table" role="grid">
          <thead>
            <tr>
              <th>Order ID</th>
              <th>Username</th>
              <th>Menu item</th>
              <th>Status</th>
              <th>Created</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in orders" :key="o.orderId">
              <td>{{ displayOrderId(o.orderId) }}</td>
              <td>{{ o.username || '—' }}</td>
              <td>{{ o.menuItemId }}</td>
              <td>{{ o.status }}</td>
              <td>{{ formatDate(o.createdAt) }}</td>
              <td>
                <button
                  v-if="o.status === 'pending'"
                  type="button"
                  class="btn-ack"
                  :disabled="updating === o.orderId"
                  @click="updateStatus(o.orderId, 'preparing')"
                >
                  {{ updating === o.orderId ? '…' : 'Acknowledge' }}
                </button>
                <button
                  v-if="o.status === 'pending' || o.status === 'preparing'"
                  type="button"
                  class="btn-delivered"
                  :disabled="updating === o.orderId"
                  @click="updateStatus(o.orderId, 'delivered')"
                >
                  {{ updating === o.orderId ? '…' : 'Mark Delivered' }}
                </button>
                <span v-if="o.status === 'delivered'">—</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </main>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { EasyPourService } from '../../gen/easypour/v1/easypour_pb.js'
import { useCurrentUser } from '../composables/useCurrentUser'
import { useOrderSync } from '../composables/useOrderSync.js'

function createOrderClient() {
  const transport = createConnectTransport({
    baseUrl: `${window.location.protocol}//${window.location.hostname}:${window.location.port}`,
  })
  return createClient(EasyPourService, transport)
}

const router = useRouter()
const { isAdmin } = useCurrentUser()
const { orders, refresh } = useOrderSync()
const client = createOrderClient()

const loading = ref(true)
const loadError = ref(null)
const updating = ref(null)

const orderGroups = computed(() => {
  const list = orders.value ?? []
  return list.map((o) => {
    const createdAt = typeof o?.createdAt === 'bigint' ? Number(o.createdAt) : Number(o?.createdAt ?? 0)
    return {
      groupId: o?.orderId ?? '',
      orderIds: [o?.orderId ?? ''],
      items: o?.orderId ? [{ orderId: o.orderId, name: o.menuItemId }] : [],
      status: o?.status ?? 'pending',
      createdAt,
    }
  }).filter((g) => g.groupId)
})

function goBack() {
  if (isAdmin) {
    router.push('/profile')
  } else {
    router.push('/')
  }
}

function displayGroupId(id) {
  if (!id) return '—'
  const s = String(id)
  return s.length > 8 ? s.slice(0, 8) : s
}

function displayOrderId(id) {
  return displayGroupId(id)
}

function itemCountLabel(group) {
  const n = Array.isArray(group?.items) ? group.items.length : 0
  return n === 1 ? '1 item' : `${n} items`
}

function formatDate(ts) {
  if (ts == null) return '—'
  const n = typeof ts === 'bigint' ? Number(ts) : Number(ts ?? 0)
  const d = new Date(n < 1e12 ? n * 1000 : n)
  return d.toLocaleString()
}

async function updateStatus(orderId, status) {
  if (!orderId || updating.value) return
  updating.value = orderId
  try {
    await client.updateOrderStatus({ orderId, status })
    await refresh()
  } catch (e) {
    loadError.value = e?.message ?? 'Update failed'
  } finally {
    updating.value = null
  }
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

.orders-table .btn-ack,
.orders-table .btn-delivered {
  margin-right: 0.5rem;
  padding: 0.35rem 0.6rem;
  font-size: 0.85rem;
  border-radius: 0.375rem;
  cursor: pointer;
  border: 1px solid #e2e8f0;
  background: #fff;
  color: #475569;
}

.orders-table .btn-ack:hover:not(:disabled),
.orders-table .btn-delivered:hover:not(:disabled) {
  background: #f1f5f9;
}

.orders-table .btn-delivered {
  border-color: #86efac;
  background: #dcfce7;
  color: #166534;
}

.orders-table .btn-delivered:hover:not(:disabled) {
  background: #bbf7d0;
}

.orders-table button:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}
</style>
