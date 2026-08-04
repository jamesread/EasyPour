import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { EasyPourService } from '../../gen/easypour/v1/easypour_pb.js'
import { useOrderSync } from './useOrderSync.js'

/**
 * @typedef {import('../../gen/easypour/v1/easypour_pb').Order} Order
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

function orderToGroup(/** @type {Order} */ order) {
  if (!order?.orderId) return null
  return {
    groupId: order.orderId,
    orderIds: [order.orderId],
    items: [
      {
        orderId: order.orderId,
        name: order.menuItemId,
        addSugar: order.addSugar,
        addMilk: order.addMilk,
        sugarAmount: order.sugarAmount,
        milkAmount: order.milkAmount,
      },
    ],
    status: order.status ?? 'pending',
    createdAt: typeof order.createdAt === 'bigint' ? Number(order.createdAt) : Number(order.createdAt ?? 0),
  }
}

export function useRecentOrders() {
  const { orders: ordersRef, refresh, isConnected } = useOrderSync()
  const client = createOrderClient()

  const recentOrders = ordersRef

  function getRecentOrders() {
    return refresh().then(() => recentOrders.value)
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
    refresh()
  }

  /**
   * Find an order group by groupId or orderId. Returns group-like shape for OrderStatus/Orders views.
   * @param {string} id - order_id (or groupId)
   * @returns {Promise<OrderGroup | null>}
   */
  async function getOrderGroup(id) {
    if (!id) return null
    const order = await getOrder(id)
    return order ? orderToGroup(order) : null
  }

  /**
   * Update order status via RPC. Returns true if successful.
   * @param {string} id - order_id
   * @param {string} status
   * @returns {Promise<boolean>}
   */
  async function updateOrderStatus(id, status) {
    if (!id || !status) return false
    try {
      await client.updateOrderStatus({ orderId: id, status })
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
  }
}
