import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { EasyPourService } from '../../gen/easypour/v1/easypour_pb.js'
import { useOrderSync } from './useOrderSync.js'

/**
 * @typedef {import('../../gen/easypour/v1/easypour_pb').Order} Order
 * @typedef {Object} MenuLookupEntry
 * @property {string} [name]
 * @property {string} [imageUrl]
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

function createOrderClient() {
  const transport = createConnectTransport({
    baseUrl: `${window.location.protocol}//${window.location.hostname}:${window.location.port}`,
  })
  return createClient(EasyPourService, transport)
}

function orderGroupKey(/** @type {Order} */ order) {
  return order?.groupId || order?.orderId || ''
}

/**
 * @param {Array<{ id?: string, name?: string, imageUrl?: string }>} menuItems
 * @returns {Map<string, MenuLookupEntry>}
 */
export function buildMenuLookup(menuItems) {
  /** @type {Map<string, MenuLookupEntry>} */
  const map = new Map()
  for (const item of menuItems ?? []) {
    if (!item?.id) continue
    map.set(item.id, { name: item.name, imageUrl: item.imageUrl })
  }
  return map
}

/**
 * @param {string} menuItemId
 * @param {Map<string, MenuLookupEntry> | Record<string, MenuLookupEntry> | null | undefined} menuById
 * @returns {MenuLookupEntry}
 */
function resolveMenuItem(menuItemId, menuById) {
  if (!menuItemId || !menuById) {
    return { name: menuItemId || 'Item' }
  }
  const entry = menuById instanceof Map ? menuById.get(menuItemId) : menuById[menuItemId]
  if (!entry) return { name: menuItemId }
  return {
    name: entry.name || menuItemId,
    imageUrl: entry.imageUrl,
  }
}

/**
 * @param {Order[]} members
 * @param {Map<string, MenuLookupEntry> | Record<string, MenuLookupEntry> | null | undefined} [menuById]
 * @returns {OrderGroup | null}
 */
function ordersToGroup(members, menuById) {
  if (!members?.length) return null
  const sorted = [...members].sort((a, b) => {
    const ac = typeof a.createdAt === 'bigint' ? Number(a.createdAt) : Number(a.createdAt ?? 0)
    const bc = typeof b.createdAt === 'bigint' ? Number(b.createdAt) : Number(b.createdAt ?? 0)
    return ac - bc
  })
  const first = sorted[0]
  const createdAt = typeof first.createdAt === 'bigint' ? Number(first.createdAt) : Number(first.createdAt ?? 0)
  const statuses = sorted.map((o) => o.status ?? 'pending')
  let status = 'pending'
  if (statuses.every((s) => s === 'delivered')) status = 'delivered'
  else if (statuses.some((s) => s === 'preparing' || s === 'delivered')) status = 'preparing'

  return {
    groupId: orderGroupKey(first),
    orderIds: sorted.map((o) => o.orderId),
    items: sorted.map((o) => {
      const resolved = resolveMenuItem(o.menuItemId, menuById)
      return {
        orderId: o.orderId,
        name: resolved.name,
        imageUrl: resolved.imageUrl,
        addSugar: o.addSugar,
        addMilk: o.addMilk,
        sugarAmount: o.sugarAmount,
        milkAmount: o.milkAmount,
      }
    }),
    status,
    createdAt,
  }
}

/**
 * @param {Order[]} orders
 * @param {Map<string, MenuLookupEntry> | Record<string, MenuLookupEntry> | null | undefined} [menuById]
 * @returns {OrderGroup[]}
 */
export function groupOrders(orders, menuById) {
  /** @type {Map<string, Order[]>} */
  const byGroup = new Map()
  for (const order of orders ?? []) {
    const key = orderGroupKey(order)
    if (!key) continue
    const list = byGroup.get(key) ?? []
    list.push(order)
    byGroup.set(key, list)
  }
  return [...byGroup.values()]
    .map((members) => ordersToGroup(members, menuById))
    .filter(Boolean)
    .sort((a, b) => b.createdAt - a.createdAt)
}

export function useRecentOrders() {
  const { orders: ordersRef, refresh, isConnected } = useOrderSync()
  const client = createOrderClient()

  const recentOrders = ordersRef

  function getRecentOrders() {
    return refresh().then(() => recentOrders.value)
  }

  /**
   * @returns {Promise<Map<string, MenuLookupEntry>>}
   */
  async function loadMenuLookup() {
    try {
      const res = await client.getMenu({})
      return buildMenuLookup(res.items ?? [])
    } catch {
      return new Map()
    }
  }

  /**
   * Get a single order by id. Returns null if not found or error.
   * @param {string} orderId
   * @returns {Promise<Order | null>}
   */
  async function getOrder(orderId) {
    if (!orderId) return null
    try {
      const res = await client.getOrder({ orderId })
      return res.order ?? null
    } catch {
      return null
    }
  }

  /**
   * Add an order group (for backward compat after Basket placeOrder). Backend already has orders; just refresh list.
   * @param {OrderGroup} _group - ignored; we refresh from backend
   */
  function addOrderGroup(_group) {
    return refresh()
  }

  /**
   * Find an order group by groupId or any member orderId.
   * @param {string} id - group_id or order_id
   * @returns {Promise<OrderGroup | null>}
   */
  async function getOrderGroup(id) {
    if (!id) return null
    await refresh()
    let members = (recentOrders.value ?? []).filter(
      (o) => o?.orderId === id || orderGroupKey(o) === id,
    )
    if (!members.length) {
      const order = await getOrder(id)
      if (!order) return null
      await refresh()
      const groupId = orderGroupKey(order)
      members = (recentOrders.value ?? []).filter(
        (o) => orderGroupKey(o) === groupId,
      )
      if (!members.length) {
        members = [order]
      }
    } else {
      const groupId = orderGroupKey(members[0])
      members = (recentOrders.value ?? []).filter((o) => orderGroupKey(o) === groupId)
    }
    const menuById = await loadMenuLookup()
    return ordersToGroup(members, menuById)
  }

  /**
   * Update status for every order in the group (id may be group_id or order_id).
   * @param {string} id
   * @param {string} status
   * @returns {Promise<boolean>}
   */
  async function updateOrderStatus(id, status) {
    if (!id || !status) return false
    const group = await getOrderGroup(id)
    const orderIds = group?.orderIds?.length ? group.orderIds : [id]
    try {
      for (const orderId of orderIds) {
        await client.updateOrderStatus({ orderId, status })
      }
      await refresh()
      return true
    } catch {
      return false
    }
  }

  return {
    recentOrders,
    getRecentOrders,
    getOrder,
    addOrderGroup,
    getOrderGroup,
    updateOrderStatus,
    isConnected,
    refresh,
    groupOrders,
  }
}
