<template>
  <main class="orders-page">
    <Section
      title="My Orders"
      subtitle="Your recent orders."
      :padding="false"
    >
      <template #toolbar>
        <button
          type="button"
          class="neutral"
          aria-label="Close"
          @click="$router.push('/')"
        >
          Close
        </button>
      </template>

      <div
        v-if="loadError"
        class="orders-error padding"
        role="alert"
      >
        {{ loadError }}
      </div>
      <div
        v-else-if="loading"
        class="orders-loading padding"
        aria-live="polite"
      >
        <p>Loading orders…</p>
      </div>
      <div
        v-else-if="orderRows.length === 0"
        class="orders-empty padding"
      >
        <p>No orders yet.</p>
        <button
          type="button"
          class="neutral"
          @click="$router.push('/')"
        >
          Browse Menu
        </button>
      </div>
      <Table
        v-else
        :data="orderRows"
        :headers="headers"
        :show-pagination="orderRows.length > 10"
      >
        <template #cell-groupId="{ row, value }">
          <router-link :to="{ name: 'OrderStatus', params: { orderId: row.groupId } }">
            {{ displayGroupId(value) }}
          </router-link>
        </template>
        <template #cell-itemCount="{ value }">
          {{ value === 1 ? '1 item' : `${value} items` }}
        </template>
        <template #cell-status="{ value }">
          <span
            class="tag"
            :class="statusKarmaClass(value)"
          >{{ value }}</span>
        </template>
        <template #cell-created="{ value }">
          {{ formatDate(value) }}
        </template>
      </Table>
    </Section>
  </main>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import Section from 'picocrank/vue/components/Section.vue'
import Table from 'picocrank/vue/components/Table.vue'
import { useCurrentUser } from '../composables/useCurrentUser'
import { useOrderSync } from '../composables/useOrderSync.js'
import { groupOrders } from '../composables/useRecentOrders.js'

const { username } = useCurrentUser()
const { orders, refresh } = useOrderSync()

const loading = ref(true)
const loadError = ref(null)

const headers = [
  { key: 'groupId', label: 'Order ID', sortable: true },
  { key: 'itemCount', label: 'Items', sortable: true },
  { key: 'status', label: 'Status', sortable: true },
  { key: 'created', label: 'Created', sortable: true },
]

const myOrders = computed(() => {
  const list = orders.value ?? []
  const me = username.value
  if (!me) return []
  return list.filter((o) => !o?.username || o.username === me)
})

const orderRows = computed(() =>
  groupOrders(myOrders.value).map((group) => ({
    groupId: group.groupId,
    itemCount: Array.isArray(group?.items) ? group.items.length : 0,
    status: group.status,
    created: createdAtMs(group.createdAt),
  })),
)

function createdAtMs(ts) {
  if (ts == null) return null
  const n = typeof ts === 'bigint' ? Number(ts) : Number(ts ?? 0)
  if (!n) return null
  return n < 1e12 ? n * 1000 : n
}

function displayGroupId(id) {
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

.orders-error {
  background: #fef2f2;
  color: #991b1b;
}

.orders-loading,
.orders-empty {
  text-align: center;
  color: #64748b;
}

.orders-empty p {
  margin-bottom: 1rem;
}
</style>
