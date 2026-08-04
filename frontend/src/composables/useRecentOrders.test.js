import { describe, it, expect } from 'vitest'
import { useRecentOrders } from './useRecentOrders'

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
