import { describe, it, expect } from 'vitest'
import { groupOrders, useRecentOrders } from './useRecentOrders'

describe('useRecentOrders', () => {
  it('returns recentOrders, getRecentOrders, getOrder, addOrderGroup, getOrderGroup, updateOrderStatus, isConnected, refresh', () => {
    const {
      recentOrders,
      getRecentOrders,
      getOrder,
      addOrderGroup,
      getOrderGroup,
      updateOrderStatus,
      isConnected,
      refresh,
    } = useRecentOrders()
    expect(recentOrders).toBeDefined()
    expect(typeof getRecentOrders).toBe('function')
    expect(typeof getOrder).toBe('function')
    expect(typeof addOrderGroup).toBe('function')
    expect(typeof getOrderGroup).toBe('function')
    expect(typeof updateOrderStatus).toBe('function')
    expect(isConnected).toBeDefined()
    expect(typeof refresh).toBe('function')
  })
})

describe('groupOrders', () => {
  it('groups multiple line items that share a groupId into one order group', () => {
    const groups = groupOrders([
      { orderId: 'a', groupId: 'g1', menuItemId: 'latte', status: 'pending', createdAt: 100 },
      { orderId: 'b', groupId: 'g1', menuItemId: 'tea', status: 'pending', createdAt: 101 },
      { orderId: 'c', groupId: 'g2', menuItemId: 'mocha', status: 'delivered', createdAt: 200 },
    ])
    expect(groups).toHaveLength(2)
    const multi = groups.find((g) => g.groupId === 'g1')
    expect(multi.items).toHaveLength(2)
    expect(multi.orderIds).toEqual(['a', 'b'])
    expect(multi.items.map((i) => i.name)).toEqual(['latte', 'tea'])
  })

  it('treats missing groupId as the order id', () => {
    const groups = groupOrders([
      { orderId: 'solo', menuItemId: 'water', status: 'pending', createdAt: 1 },
    ])
    expect(groups).toHaveLength(1)
    expect(groups[0].groupId).toBe('solo')
    expect(groups[0].items).toHaveLength(1)
  })

  it('resolves menu item ids to display names and images', () => {
    const menuById = new Map([
      ['item-3ab22e63', { name: 'marmite toast', imageUrl: '/images/toast.jpg' }],
    ])
    const groups = groupOrders(
      [{ orderId: 'a0f5be28', groupId: 'a0f5be28', menuItemId: 'item-3ab22e63', status: 'pending', createdAt: 1 }],
      menuById,
    )
    expect(groups).toHaveLength(1)
    expect(groups[0].items[0].name).toBe('marmite toast')
    expect(groups[0].items[0].imageUrl).toBe('/images/toast.jpg')
  })
})
