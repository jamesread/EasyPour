<template>
  <main class="orders-page">
    <Section
      title="Admin Orders"
      subtitle="All orders. Acknowledge pending orders or mark as delivered."
      :padding="false"
    >
      <template #toolbar>
        <button type="button" class="neutral" @click="$router.push('/profile')" aria-label="Back">Back</button>
      </template>

      <div v-if="!isAdmin" class="orders-error padding" role="alert">
        Admin access required.
      </div>
      <div v-else-if="loadError" class="orders-error padding" role="alert">
        {{ loadError }}
      </div>
      <div v-else-if="loading" class="orders-loading padding" aria-live="polite">
        <p>Loading orders…</p>
      </div>
      <Table
        v-else
        :data="orderRows"
        :headers="headers"
        :show-pagination="orderRows.length > 10"
      >
        <template #cell-orderId="{ row, value }">
          <router-link :to="{ name: 'OrderStatus', params: { orderId: row.orderId } }">
            {{ displayOrderId(value) }}
          </router-link>
        </template>
        <template #cell-status="{ value }">
          <span class="tag" :class="statusKarmaClass(value)">{{ value }}</span>
        </template>
        <template #cell-created="{ value }">
          {{ formatDate(value) }}
        </template>
        <template #cell-actions="{ row }">
          <span class="order-actions">
            <button
              v-if="row.status === 'pending'"
              type="button"
              class="good"
              :disabled="updating === row.orderId"
              @click="updateStatus(row.orderId, 'preparing')"
            >
              {{ updating === row.orderId ? '…' : 'Acknowledge' }}
            </button>
            <button
              v-if="row.status === 'pending' || row.status === 'preparing'"
              type="button"
              class="good"
              :disabled="updating === row.orderId"
              @click="updateStatus(row.orderId, 'delivered')"
            >
              {{ updating === row.orderId ? '…' : 'Mark Delivered' }}
            </button>
            <span v-if="row.status === 'delivered'">—</span>
          </span>
        </template>
      </Table>
    </Section>
  </main>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import Section from 'picocrank/vue/components/Section.vue'
import Table from 'picocrank/vue/components/Table.vue'
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

const headers = [
  { key: 'orderId', label: 'Order ID', sortable: true },
  { key: 'username', label: 'Username', sortable: true },
  { key: 'status', label: 'Status', sortable: true },
  { key: 'created', label: 'Created', sortable: true },
  { key: 'actions', label: 'Actions', sortable: false },
]

const orderRows = computed(() =>
  (orders.value ?? []).map((o) => ({
    orderId: o.orderId,
    username: o.username || '—',
    status: o.status,
    created: createdAtMs(o.createdAt),
    actions: '',
  })),
)

function createdAtMs(ts) {
  if (ts == null) return null
  const n = typeof ts === 'bigint' ? Number(ts) : Number(ts ?? 0)
  if (!n) return null
  return n < 1e12 ? n * 1000 : n
}

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

function formatDate(ts) {
  if (ts == null) return '—'
  return new Date(ts).toLocaleString()
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

.orders-error {
  background: #fef2f2;
  color: #991b1b;
}

.orders-loading {
  text-align: center;
  color: #64748b;
}

.order-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
