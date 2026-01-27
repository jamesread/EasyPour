<template>
  <main class="order-status-page">
    <div v-if="!group" class="order-status-inner order-not-found">
      <div class="not-found-card">
        <p class="not-found-msg">We couldn’t find that order.</p>
        <div class="status-actions">
          <button class="btn-secondary" @click="$router.push('/orders')">View all orders</button>
          <button class="btn-primary" @click="$router.push('/')">Browse menu</button>
        </div>
      </div>
    </div>

    <div v-else class="order-status-inner" aria-live="polite">
      <!-- Confirmation hero -->
      <div class="confirmation-hero">
        <div class="confirmation-icon" aria-hidden="true">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
            <polyline points="22 4 12 14.01 9 11.01"/>
          </svg>
        </div>
        <h1 class="confirmation-title">Order confirmed!</h1>
        <p class="confirmation-subtitle">We’re getting your order ready.</p>
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

      <!-- Admin: mark as delivered -->
      <div v-if="isAdmin && stepIndex < 2" class="admin-actions">
        <button
          type="button"
          class="btn-delivered"
          :disabled="markingDelivered"
          @click="markAsDelivered"
        >
          {{ markingDelivered ? 'Updating…' : 'Mark as delivered' }}
        </button>
      </div>

      <!-- Actions -->
      <div class="status-actions">
        <button class="btn-primary" @click="$router.push('/')">
          Back to menu
        </button>
        <button class="btn-secondary" @click="$router.push('/orders')">
          View all orders
        </button>
      </div>
    </div>
  </main>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { HugeiconsIcon } from '@hugeicons/vue'
import { DrinkIcon } from '@hugeicons/core-free-icons'
import { useRecentOrders } from '../composables/useRecentOrders'
import { useCurrentUser } from '../composables/useCurrentUser'

const route = useRoute()
const { getRecentOrders, recentOrders, updateOrderStatus } = useRecentOrders()
const { isAdmin } = useCurrentUser()
const markingDelivered = ref(false)

const orderId = computed(() => route.params.orderId)

const group = computed(() => {
  const id = orderId.value
  if (!id) return null
  const list = recentOrders.value
  const byGroup = list.find((g) => g && g.groupId === id)
  if (byGroup && Array.isArray(byGroup.items)) return byGroup
  const byOrderId = list.find((g) => g && Array.isArray(g.orderIds) && g.orderIds.includes(id))
  if (byOrderId && Array.isArray(byOrderId.items)) return byOrderId
  return null
})

const displayOrderId = computed(() => {
  const g = group.value
  if (!g?.groupId) return '—'
  const s = String(g.groupId)
  return s.length > 8 ? s.slice(0, 8) : s
})

const statusToStep = {
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

const preparingMessage = computed(() => {
  const i = stepIndex.value
  if (i >= 2) return 'Your order is ready'
  if (i === 1) return 'Your order is being prepared'
  return 'We’ll start preparing soon'
})

function markAsDelivered() {
  const g = group.value
  if (!g?.groupId || markingDelivered.value) return
  markingDelivered.value = true
  updateOrderStatus(g.groupId, 'delivered')
  markingDelivered.value = false
}

onMounted(() => {
  getRecentOrders()
})
</script>

<style scoped>
.order-status-page {
  min-height: 100vh;
  background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%);
  padding: 1.5rem 1rem 2rem;
}

.order-status-inner {
  max-width: 28rem;
  margin: 0 auto;
}

/* Not found */
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
  background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
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

.btn-delivered {
  width: 100%;
  padding: 0.75rem 1.25rem;
  font-size: 0.95rem;
  font-weight: 600;
  color: #166534;
  background: #dcfce7;
  border: 1px solid #86efac;
  border-radius: 0.75rem;
  cursor: pointer;
}

.btn-delivered:hover:not(:disabled) {
  background: #bbf7d0;
  border-color: #4ade80;
}

.btn-delivered:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

/* Actions */
.status-actions {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.btn-primary {
  width: 100%;
  padding: 0.875rem 1.25rem;
  font-size: 1rem;
  font-weight: 600;
  color: #fff;
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
  border: none;
  border-radius: 0.75rem;
  cursor: pointer;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
}

.btn-primary:hover {
  background: linear-gradient(135deg, #1e293b 0%, #334155 100%);
}

.btn-secondary {
  width: 100%;
  padding: 0.75rem 1.25rem;
  font-size: 0.95rem;
  font-weight: 500;
  color: #475569;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 0.75rem;
  cursor: pointer;
}

.btn-secondary:hover {
  background: #f8fafc;
  border-color: #cbd5e1;
}
</style>
