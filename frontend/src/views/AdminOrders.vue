<template>
  <main class="orders-page">
    <section class="padding">
      <div class="orders-header">
        <h2>Admin Orders</h2>
        <button type="button" class="neutral" @click="$router.push('/profile')" aria-label="Back">Back</button>
      </div>

      <p class="orders-intro">All orders. Acknowledge pending orders or mark as delivered.</p>

      <div v-if="!isAdmin" class="orders-error" role="alert">
        Admin access required.
      </div>
      <div v-else-if="loadError" class="orders-error" role="alert">
        {{ loadError }}
      </div>
      <div v-else-if="loading" class="orders-loading" aria-live="polite">
        <p>Loading orders…</p>
      </div>
      <div v-else-if="orders.length === 0" class="empty-orders">
        <p>No orders yet.</p>
      </div>

      <div v-else class="orders-table-wrap">
        <table class="orders-table" role="grid">
          <thead>
            <tr>
              <th>Order ID</th>
              <th>Username</th>
              <th>Status</th>
              <th>Created</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="o in orders"
              :key="o.orderId"
              class="order-row"
              @click="openOrder(o.orderId)"
            >
              <td>{{ displayOrderId(o.orderId) }}</td>
              <td>{{ o.username || '—' }}</td>
              <td><span class="tag" :class="statusKarmaClass(o.status)">{{ o.status }}</span></td>
              <td>{{ formatDate(o.createdAt) }}</td>
              <td class="order-actions" @click.stop>
                <button
                  v-if="o.status === 'pending'"
                  type="button"
                  class="good"
                  :disabled="updating === o.orderId"
                  @click="updateStatus(o.orderId, 'preparing')"
                >
                  {{ updating === o.orderId ? '…' : 'Acknowledge' }}
                </button>
                <button
                  v-if="o.status === 'pending' || o.status === 'preparing'"
                  type="button"
                  class="good"
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
import { onMounted, ref } from 'vue'
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
const { isAdmin, refresh: refreshUser } = useCurrentUser()
const { orders, refresh } = useOrderSync()
const client = createOrderClient()

const loading = ref(true)
const loadError = ref(null)
const updating = ref(null)

function displayOrderId(id) {
  if (!id) return '—'
  const s = String(id)
  return s.length > 8 ? s.slice(0, 8) : s
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

function openOrder(orderId) {
  if (!orderId) return
  router.push({ name: 'OrderStatus', params: { orderId } })
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
  await refreshUser()
  if (!isAdmin.value) {
    router.replace('/orders')
    return
  }
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

.order-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
