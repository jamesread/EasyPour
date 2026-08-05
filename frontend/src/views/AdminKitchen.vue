<template>
  <main class="kitchen-page">
    <Section
      title="Kitchen View"
      subtitle="Item counts from all non-delivered orders"
    >
      <template #toolbar>
        <button type="button" class="neutral" @click="$router.push('/profile')" aria-label="Back">Back</button>
      </template>

      <div v-if="!isAdmin" class="kitchen-error" role="alert">
        Admin access required.
      </div>
      <div v-else-if="loadError" class="kitchen-error" role="alert">
        {{ loadError }}
      </div>
      <div v-else-if="loading" class="kitchen-loading" aria-live="polite">
        <p>Loading…</p>
      </div>
      <div v-else-if="kitchenItems.length === 0" class="kitchen-empty">
        <p>No open orders.</p>
      </div>
      <div v-else class="kitchen-table-wrap">
        <table class="kitchen-table" role="grid">
          <thead>
            <tr>
              <th>Item</th>
              <th>Options</th>
              <th>Count</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in kitchenItems" :key="`${row.menuItemId}|${row.options}`">
              <td class="kitchen-item-cell">
                <img
                  v-if="row.imageUrl"
                  :src="row.imageUrl"
                  :alt="row.name"
                  class="kitchen-item-image"
                />
                <span>{{ row.name }}</span>
              </td>
              <td>{{ row.options }}</td>
              <td class="kitchen-count">{{ row.count }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </Section>
  </main>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import Section from 'picocrank/vue/components/Section.vue'
import { EasyPourService } from '../../gen/easypour/v1/easypour_pb.js'
import { buildMenuLookup } from '../composables/useRecentOrders.js'
import { countKitchenItems } from '../composables/kitchenItems.js'
import { useCurrentUser } from '../composables/useCurrentUser'
import { useOrderSync } from '../composables/useOrderSync.js'

function createMenuClient() {
  const transport = createConnectTransport({
    baseUrl: `${window.location.protocol}//${window.location.hostname}:${window.location.port}`,
  })
  return createClient(EasyPourService, transport)
}

const router = useRouter()
const { isAdmin, refresh: refreshUser } = useCurrentUser()
const { orders, refresh } = useOrderSync()
const client = createMenuClient()

const loading = ref(true)
const loadError = ref(null)
const menuById = ref(new Map())

const kitchenItems = computed(() => countKitchenItems(orders.value, menuById.value))

onMounted(async () => {
  await refreshUser()
  if (!isAdmin.value) {
    router.replace('/profile')
    return
  }
  try {
    const [menuRes] = await Promise.all([
      client.getMenu({}),
      refresh(),
    ])
    menuById.value = buildMenuLookup(menuRes.items ?? [])
  } catch (e) {
    loadError.value = e?.message ?? 'Failed to load kitchen view'
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.kitchen-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 1rem;
}

.kitchen-page :deep(section) {
  width: 100%;
  max-width: 48rem;
}

.kitchen-error {
  padding: 1rem;
  background: #fef2f2;
  color: #991b1b;
  border-radius: 0.5rem;
  margin-bottom: 1rem;
}

.kitchen-loading,
.kitchen-empty {
  padding: 2rem;
  text-align: center;
  color: #64748b;
}

.kitchen-table-wrap {
  overflow-x: auto;
}

.kitchen-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.95rem;
}

.kitchen-table th,
.kitchen-table td {
  padding: 0.75rem;
  text-align: left;
  border-bottom: 1px solid #e2e8f0;
}

.kitchen-table th {
  font-weight: 600;
  color: #0f172a;
  background: #f8fafc;
}

.kitchen-table td {
  color: #475569;
}

.kitchen-item-cell {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-weight: 600;
  color: #0f172a;
}

.kitchen-item-image {
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 0.375rem;
  object-fit: cover;
  background: #f1f5f9;
}

.kitchen-count {
  font-size: 1.25rem;
  font-weight: 700;
  color: #0f172a;
}
</style>
