import { ref } from 'vue'

const STORAGE_KEY = 'easypour-recent-orders'
const MAX_GROUPS = 50

/**
 * @typedef {Object} OrderItem
 * @property {string} orderId
 * @property {string} [name]
 * @property {string} [imageUrl]
 * @property {boolean} [addSugar]
 * @property {boolean} [addMilk]
 * @property {number} [sugarAmount]
 * @property {string} [sugarType]
 */

/**
 * @typedef {Object} OrderGroup
 * @property {string} groupId
 * @property {string[]} orderIds
 * @property {OrderItem[]} items
 * @property {string} status
 * @property {number} createdAt
 */

function loadFromStorage() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function saveToStorage(groups) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(groups))
  } catch (_) {}
}

/**
 * @param {OrderGroup} group
 * @returns {OrderGroup[]}
 */
function mergeGroupAndSave(group) {
  if (!group || !group.groupId) return loadFromStorage()
  const existing = loadFromStorage()
  const byGroupId = new Map()
  for (const g of existing) {
    if (g && g.groupId && Array.isArray(g.items)) {
      byGroupId.set(g.groupId, g)
    }
  }
  byGroupId.set(group.groupId, {
    groupId: group.groupId,
    orderIds: group.orderIds ?? [],
    items: group.items ?? [],
    status: group.status ?? 'preparing',
    createdAt: group.createdAt ?? Date.now(),
  })
  const merged = Array.from(byGroupId.values()).sort(
    (a, b) => (b.createdAt ?? 0) - (a.createdAt ?? 0)
  )
  const trimmed = merged.slice(0, MAX_GROUPS)
  saveToStorage(trimmed)
  return trimmed
}

export function useRecentOrders() {
  const recentOrders = ref(loadFromStorage())

  function getRecentOrders() {
    recentOrders.value = loadFromStorage()
    return recentOrders.value
  }

  /**
   * Add an order group (one checkout = multiple items). Persists to localStorage.
   * @param {OrderGroup} group - { groupId, orderIds, items: [{ orderId, name, imageUrl, addSugar, addMilk, ... }], status, createdAt }
   */
  function addOrderGroup(group) {
    if (!group?.groupId) return recentOrders.value
    recentOrders.value = mergeGroupAndSave(group)
    return recentOrders.value
  }

  /**
   * Find an order group by groupId or by any orderId in the group.
   * @param {string} id - groupId or one of the orderIds in the group
   * @returns {OrderGroup | null}
   */
  function getOrderGroup(id) {
    if (!id) return null
    const list = loadFromStorage()
    const groupById = list.find((g) => g && g.groupId === id)
    if (groupById && Array.isArray(groupById.items)) return groupById
    const groupByOrderId = list.find(
      (g) => g && Array.isArray(g.orderIds) && g.orderIds.includes(id)
    )
    if (groupByOrderId && Array.isArray(groupByOrderId.items)) return groupByOrderId
    return null
  }

  /**
   * Update a group's status by groupId. Persists to localStorage.
   * @param {string} id - groupId
   * @param {string} status - e.g. 'delivered', 'ready', 'preparing'
   * @returns {boolean}
   */
  function updateOrderStatus(id, status) {
    if (!id || !status) return false
    const list = loadFromStorage()
    const idx = list.findIndex((g) => g && g.groupId === id)
    if (idx < 0) return false
    list[idx] = { ...list[idx], status }
    saveToStorage(list)
    recentOrders.value = list
    return true
  }

  return {
    recentOrders,
    getRecentOrders,
    addOrderGroup,
    getOrderGroup,
    updateOrderStatus,
  }
}
