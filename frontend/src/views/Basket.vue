<template>
  <main class="basket-page">
    <section class="padding">
      <div class="basket-header">
        <h2>Basket</h2>
        <button @click="$router.push('/')" aria-label="Close">Close</button>
      </div>
      
      <div v-if="basketItems.length === 0" class="empty-basket">
        <p>Your basket is empty</p>
        <button @click="$router.push('/')">Browse Menu</button>
      </div>
      
      <div v-else>
        <div v-if="error" class="basket-error" role="alert">
          {{ error }}
        </div>
        <div v-for="(item, index) in basketItems" :key="index" class="basket-item" @click="openEditDialog(index)">
          <div class="item-image">
            <img
              v-if="item.imageUrl"
              :src="item.imageUrl"
              :alt="item.name"
            />
            <HugeiconsIcon v-else :icon="DrinkIcon" class="item-icon-placeholder" aria-hidden="true" />
          </div>
          <div class="item-info">
            <h4>{{ item.name }}</h4>
            <div v-if="item.addSugar" class="item-detail">
              Sugar: {{ item.sugarType === 'diabetes' ? 'diabetes' : `${item.sugarAmount} ${item.sugarAmount === 1 ? 'spoon' : 'spoons'}` }}
            </div>
            <div v-else class="item-detail">
              No sugar
            </div>
            <div v-if="item.addMilk" class="item-detail">
              Milk
            </div>
          </div>
          <button @click.stop="removeFromBasket(index)" aria-label="Remove item">Remove</button>
        </div>
        
        <div class="basket-footer">
          <button @click="$router.push('/')">Continue Shopping</button>
          <button @click="placeOrder" :disabled="ordering">
            {{ ordering ? 'Placing Order...' : 'Place Order' }}
          </button>
        </div>
      </div>
    </section>

    <CustomizeItem
      :menu-item="editingMenuItem"
      :open="customizeOpen"
      :edit-basket-item="editingBasketItem"
      :edit-basket-index="editingBasketIndex"
      @close="closeCustomizeDialog"
      @add-to-basket="handleAddToBasket"
    />
  </main>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { EasyPourService } from '../../gen/easypour/v1/easypour_pb.js'
import { HugeiconsIcon } from '@hugeicons/vue'
import { DrinkIcon } from '@hugeicons/core-free-icons'
import CustomizeItem from '../CustomizeItem.vue'
import { useBasket } from '../composables/useBasket'
import { useRecentOrders } from '../composables/useRecentOrders'

const router = useRouter()
const { addOrderGroup } = useRecentOrders()

const transport = createConnectTransport({
  baseUrl: `${window.location.protocol}//${window.location.hostname}:${window.location.port}`,
})

const client = createClient(EasyPourService, transport)

const { basketItems, removeFromBasket, updateBasketItem, clearBasket } = useBasket()
const ordering = ref(false)
const error = ref(null)
const customizeOpen = ref(false)
const editingBasketIndex = ref(null)

const editingBasketItem = computed(() => {
  const i = editingBasketIndex.value
  if (i == null || i < 0 || i >= basketItems.value.length) return null
  return basketItems.value[i]
})

const editingMenuItem = computed(() => {
  const item = editingBasketItem.value
  if (!item) return null
  return {
    id: item.id,
    name: item.name,
    supportsSugar: item.supportsSugar ?? true,
    supportsMilk: item.supportsMilk ?? true,
    imageUrl: item.imageUrl,
  }
})

const openEditDialog = (index) => {
  editingBasketIndex.value = index
  customizeOpen.value = true
}

const closeCustomizeDialog = () => {
  customizeOpen.value = false
  editingBasketIndex.value = null
}

const handleAddToBasket = (item, index) => {
  if (index !== undefined && index !== null) {
    updateBasketItem(index, item)
  }
  closeCustomizeDialog()
}

const placeOrder = async () => {
  if (basketItems.value.length === 0) return

  ordering.value = true
  error.value = null

  try {
    // Place order for all items in basket
    const responses = []
    for (const item of basketItems.value) {
      const res = await client.orderDrink({
        menuItemId: item.id,
        addSugar: item.addSugar,
        addMilk: item.addMilk,
        sugarAmount: item.sugarAmount,
        milkAmount: item.milkAmount,
      })
      responses.push(res)
    }

    // Normalize responses (createdAt may be bigint; orderId may be under order_id)
    const orderIds = responses.map((r) => String(r?.orderId ?? r?.order_id ?? ''))
    const groupId = orderIds[0] ?? `ck-${Date.now()}`
    const items = basketItems.value.map((item, i) => ({
      orderId: orderIds[i] ?? '',
      name: item.name,
      imageUrl: item.imageUrl,
      addSugar: item.addSugar,
      addMilk: item.addMilk,
      sugarAmount: item.sugarAmount,
      sugarType: item.sugarType,
    }))

    addOrderGroup({
      groupId,
      orderIds,
      items,
      status: 'preparing',
      createdAt: Date.now(),
    })

    // Clear basket
    clearBasket()
    ordering.value = false

    if (groupId) {
      router.push({ name: 'OrderStatus', params: { orderId: groupId } })
    } else {
      router.push({ name: 'Orders' })
    }
  } catch (err) {
    error.value = `Failed to place order: ${err.message}`
    ordering.value = false
  }
}
</script>

<style scoped>
.basket-page {
  min-height: 100vh;
}

.basket-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
}

.empty-basket {
  padding: 4rem 2rem;
  text-align: center;
}

.empty-basket p {
  margin-bottom: 1rem;
  font-size: 1.2rem;
}

.basket-error {
  padding: 0.75rem 1rem;
  margin-bottom: 1rem;
  background-color: #fef2f2;
  color: #b91c1c;
  border-radius: 0.375rem;
  font-size: 0.95rem;
}

.basket-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  border-bottom: 1px solid #ccc;
  cursor: pointer;
}

.basket-item:hover {
  background-color: #f8f8f8;
}

.basket-item .item-image {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  min-width: 36px;
  border-radius: 0.375rem;
  background: #eee;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.basket-item .item-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.basket-item .item-icon-placeholder {
  width: 20px;
  height: 20px;
  color: #999;
}

.basket-item .item-info {
  flex: 1;
  min-width: 0;
  text-align: left;
}

.item-info h4 {
  margin: 0 0 0.5rem 0;
  text-align: left;
}

.item-detail {
  font-size: 0.9rem;
  color: #666;
  margin: 0.25rem 0;
}

.basket-footer {
  display: flex;
  gap: 1rem;
  margin-top: 2rem;
  padding-top: 2rem;
  border-top: 1px solid #ccc;
}

.basket-footer button {
  flex: 1;
}
</style>

