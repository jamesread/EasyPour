<template>
  <main class="order-status-page">
    <div v-if="loading" class="order-status-inner order-loading" aria-live="polite">
      <p class="loading-msg">Loading order…</p>
    </div>
    <div v-else-if="loadError" class="order-status-inner order-not-found">
      <div class="not-found-card">
        <p class="not-found-msg">{{ loadError }}</p>
        <div class="status-actions">
          <button type="button" class="neutral" @click="goToOrdersList">View all orders</button>
          <button type="button" class="good" @click="$router.push('/')">Browse menu</button>
        </div>
      </div>
    </div>
    <div v-else-if="!group" class="order-status-inner order-not-found">
      <div class="not-found-card">
        <p class="not-found-msg">We couldn’t find that order.</p>
        <div class="status-actions">
          <button type="button" class="neutral" @click="goToOrdersList">View all orders</button>
          <button type="button" class="good" @click="$router.push('/')">Browse menu</button>
        </div>
      </div>
    </div>

    <div v-else class="order-status-inner" aria-live="polite">
      <!-- Status hero -->
      <div class="confirmation-hero">
        <div class="confirmation-icon" :class="heroIconClass" aria-hidden="true">
          <svg v-if="stepIndex === 0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <polyline points="12 6 12 12 16 14"/>
          </svg>
          <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
            <polyline points="22 4 12 14.01 9 11.01"/>
          </svg>
        </div>
        <h1 class="confirmation-title">{{ heroTitle }}</h1>
        <p class="confirmation-subtitle">{{ heroSubtitle }}</p>
        <p class="order-number">Order #{{ displayOrderId }}</p>
      </div>

      <!-- Order items -->
      <div class="order-items-card">
        <h3 class="order-items-title">Your order</h3>
        <ul class="order-items-list">
          <li v-for="(item, idx) in group.items" :key="item.orderId || idx" class="order-item-row">
            <div class="order-item-image">
              <img v-if="item.imageUrl" :src="item.imageUrl" :alt="item.name || 'Item'" />
              <HugeiconsIcon v-else :icon="DrinkIcon" class="order-item-icon" aria-hidden="true" />
            </div>
            <div class="order-item-info">
              <span class="order-item-name">{{ item.name || 'Item' }}</span>
              <div class="order-item-options">
                <span v-if="item.addSugar" class="order-item-opt">
                  {{ item.sugarType === 'diabetes' ? 'Diabetic sugar' : `${item.sugarAmount ?? 1} sugar` }}
                </span>
                <span v-else class="order-item-opt">No sugar</span>
                <span v-if="item.addMilk" class="order-item-opt">Milk</span>
              </div>
            </div>
          </li>
        </ul>
      </div>

      <!-- Progress steps -->
      <div class="progress-card">
        <ol class="progress-steps" role="list">
          <li class="step" :class="{ active: stepIndex >= 0, done: stepIndex > 0 }">
            <span class="step-marker">
              <span v-if="stepIndex > 0" class="step-check">✓</span>
              <span v-else class="step-dot">1</span>
            </span>
            <div class="step-content">
              <span class="step-label">Order received</span>
              <span class="step-detail">We’ve got your order</span>
            </div>
          </li>
          <li class="step" :class="{ active: stepIndex >= 1, done: stepIndex > 1 }">
            <span class="step-marker">
              <span v-if="stepIndex > 1" class="step-check">✓</span>
              <span v-else class="step-dot">2</span>
            </span>
            <div class="step-content">
              <span class="step-label">Preparing</span>
              <span class="step-detail">{{ preparingMessage }}</span>
            </div>
          </li>
          <li class="step" :class="{ active: stepIndex >= 2, done: stepIndex > 2 }">
            <span class="step-marker">
              <span v-if="stepIndex > 2" class="step-check">✓</span>
              <span v-else class="step-dot">3</span>
            </span>
            <div class="step-content">
              <span class="step-label">Ready</span>
              <span class="step-detail">Ready for pickup or delivery</span>
            </div>
          </li>
        </ol>
      </div>

      <!-- Estimated time (when in "preparing" state) -->
      <p v-if="stepIndex === 1" class="eta-message">
        Usually ready in about <strong>5–10 minutes</strong>.
      </p>

      <!-- Mark as delivered: admin or order owner when preparing/pending -->
      <div v-if="canMarkDelivered && stepIndex < 2" class="admin-actions">
        <button
          type="button"
          class="good"
          :disabled="markingDelivered"
          @click="markAsDelivered"
        >
          {{ markingDelivered ? 'Updating…' : 'Mark as delivered' }}
        </button>
      </div>

      <!-- Actions -->
      <div class="status-actions">
        <button type="button" class="good" @click="$router.push('/')">
          Back to menu
        </button>
        <button type="button" class="neutral" @click="goToOrdersList">
          View all orders
        </button>
      </div>
    </div>
  </main>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { HugeiconsIcon } from '@hugeicons/vue'
import { DrinkIcon } from '@hugeicons/core-free-icons'
import { useRecentOrders } from '../composables/useRecentOrders'
import { useCurrentUser } from '../composables/useCurrentUser'

const route = useRoute()
const router = useRouter()
const { getOrderGroup, recentOrders, updateOrderStatus, refresh } = useRecentOrders()
const { isAdmin } = useCurrentUser()
const markingDelivered = ref(false)
const loading = ref(true)
const loadError = ref(null)
const group = ref(null)

const orderId = computed(() => route.params.orderId)

function goToOrdersList() {
  router.push({ name: isAdmin.value ? 'AdminOrders' : 'Orders' })
}

async function loadGroup() {
  const id = orderId.value
  if (!id) {
    group.value = null
    loading.value = false
    return
  }
  loading.value = true
  loadError.value = null
  try {
    const g = await getOrderGroup(id)
    group.value = g
    if (!g) loadError.value = 'Order not found.'
  } catch {
    loadError.value = 'Failed to load order.'
    group.value = null
  } finally {
    loading.value = false
  }
}

watch(orderId, loadGroup, { immediate: false })
watch(
  () => recentOrders.value,
  async (list) => {
    const id = orderId.value
    const current = group.value
    if (!id || !current || !list?.length) return
    const groupId = current.groupId
    const members = list.filter(
      (o) => o?.groupId === groupId || o?.orderId === groupId || current.orderIds?.includes(o?.orderId),
    )
    if (!members.length) return
    const statuses = members.map((o) => o.status ?? 'pending')
    let status = 'pending'
    if (statuses.every((s) => s === 'delivered')) status = 'delivered'
    else if (statuses.some((s) => s === 'preparing' || s === 'delivered')) status = 'preparing'
    if (status !== current.status || members.length !== current.items?.length) {
      const g = await getOrderGroup(id)
      if (g) group.value = g
    }
  },
  { deep: true }
)

const displayOrderId = computed(() => {
  const g = group.value
  if (!g?.groupId) return '—'
  const s = String(g.groupId)
  return s.length > 8 ? s.slice(0, 8) : s
})

const statusToStep = {
  pending: 0,
  preparing: 1,
  ready: 2,
  completed: 2,
  delivered: 2,
  done: 2,
}
const stepIndex = computed(() => {
  const s = (group.value?.status ?? '').toLowerCase()
  if (statusToStep[s] !== undefined) return statusToStep[s]
  return 1
})

const heroTitle = computed(() => {
  if (stepIndex.value === 0) return 'Order pending'
  if (stepIndex.value >= 2) return 'Order ready'
  return 'Preparing your order'
})

const heroSubtitle = computed(() => {
  if (stepIndex.value === 0) return 'Your order has been received and is awaiting acknowledgment.'
  if (stepIndex.value >= 2) return 'Your order is ready for pickup or delivery.'
  return 'We’re getting your order ready.'
})

const heroIconClass = computed(() => {
  if (stepIndex.value === 0) return 'pending'
  if (stepIndex.value >= 2) return 'ready'
  return 'confirmed'
})

const preparingMessage = computed(() => {
  const i = stepIndex.value
  if (i >= 2) return 'Your order is ready'
  if (i === 1) return 'Your order is being prepared'
  return 'We’ll start preparing soon'
})

const canMarkDelivered = computed(() => {
  const g = group.value
  if (!g?.groupId || stepIndex.value >= 2) return false
  return isAdmin.value || true
})

async function markAsDelivered() {
  const g = group.value
  if (!g?.groupId || markingDelivered.value) return
  markingDelivered.value = true
  try {
    const ok = await updateOrderStatus(g.groupId, 'delivered')
    if (ok) await loadGroup()
  } finally {
    markingDelivered.value = false
  }
}

onMounted(() => {
  refresh()
  loadGroup()
})
</script>

<style scoped>
.order-status-page {
  min-height: 100vh;
  background: transparent;
  padding: 1.5rem 1rem 2rem;
}

.order-status-inner {
  max-width: 28rem;
  margin: 0 auto;
}

/* Not found */
.order-loading {
  padding-top: 3rem;
  text-align: center;
}

.loading-msg {
  margin: 0;
  color: #64748b;
}

.order-not-found {
  padding-top: 3rem;
}

.not-found-card {
  background: #fff;
  border-radius: 1rem;
  padding: 2rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.not-found-msg {
  margin: 0 0 1.5rem 0;
  font-size: 1.1rem;
  color: #475569;
}

/* Confirmation hero */
.confirmation-hero {
  text-align: center;
  margin-bottom: 1.5rem;
}

.confirmation-icon {
  width: 4rem;
  height: 4rem;
  margin: 0 auto 1rem;
  border-radius: 50%;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}

.confirmation-icon.pending {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  box-shadow: 0 4px 14px rgba(245, 158, 11, 0.4);
}

.confirmation-icon.confirmed,
.confirmation-icon.ready {
  background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%);
  box-shadow: 0 4px 14px rgba(34, 197, 94, 0.4);
}

.confirmation-icon svg {
  width: 2.25rem;
  height: 2.25rem;
}

.confirmation-title {
  margin: 0 0 0.25rem 0;
  font-size: 1.75rem;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.02em;
}

.confirmation-subtitle {
  margin: 0 0 0.5rem 0;
  font-size: 1.05rem;
  color: #64748b;
}

.order-number {
  margin: 0;
  font-size: 0.9rem;
  color: #94a3b8;
  font-weight: 500;
}

/* Order items */
.order-items-card {
  background: #fff;
  border-radius: 1rem;
  padding: 1.25rem 1.5rem;
  margin-bottom: 1rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.order-items-title {
  margin: 0 0 1rem 0;
  font-size: 1rem;
  font-weight: 600;
  color: #0f172a;
}

.order-items-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.order-item-row {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 0;
  border-bottom: 1px solid #f1f5f9;
}

.order-item-row:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.order-item-image {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  border-radius: 0.5rem;
  background: #f1f5f9;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.order-item-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.order-item-icon {
  width: 1.5rem;
  height: 1.5rem;
  color: #94a3b8;
}

.order-item-info {
  flex: 1;
  min-width: 0;
}

.order-item-name {
  display: block;
  font-weight: 600;
  color: #0f172a;
  font-size: 0.95rem;
  margin-bottom: 0.2rem;
}

.order-item-options {
  font-size: 0.8rem;
  color: #64748b;
}

.order-item-opt + .order-item-opt::before {
  content: ' · ';
}

/* Progress card */
.progress-card {
  background: #fff;
  border-radius: 1rem;
  padding: 1.25rem 1.5rem;
  margin-bottom: 1rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.progress-steps {
  list-style: none;
  margin: 0;
  padding: 0;
}

.step {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  position: relative;
  padding-bottom: 1.25rem;
}

.step:last-child {
  padding-bottom: 0;
}

.step:not(:last-child)::before {
  content: '';
  position: absolute;
  left: 0.9375rem;
  top: 2rem;
  bottom: 0;
  width: 2px;
  background: #e2e8f0;
}

.step.done:not(:last-child)::before {
  background: #22c55e;
}

.step-marker {
  flex-shrink: 0;
  width: 2rem;
  height: 2rem;
  border-radius: 50%;
  background: #e2e8f0;
  color: #94a3b8;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
  font-weight: 600;
  position: relative;
  z-index: 1;
}

.step.active .step-marker {
  background: #22c55e;
  color: #fff;
}

.step.done .step-marker {
  background: #22c55e;
  color: #fff;
}

.step-check {
  font-size: 0.9rem;
  font-weight: 700;
}

.step-content {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.step-label {
  font-weight: 600;
  color: #0f172a;
  font-size: 0.95rem;
}

.step-detail {
  font-size: 0.85rem;
  color: #64748b;
}

.step:not(.active):not(.done) .step-label,
.step:not(.active):not(.done) .step-detail {
  color: #94a3b8;
}

/* ETA */
.eta-message {
  margin: 0 0 1.5rem 0;
  padding: 0 0.25rem;
  font-size: 0.95rem;
  color: #475569;
}

.eta-message strong {
  color: #0f172a;
}

/* Admin: Mark as delivered */
.admin-actions {
  margin-bottom: 1.25rem;
}

.admin-actions button,
.status-actions button {
  width: 100%;
}

/* Actions */
.status-actions {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
</style>
