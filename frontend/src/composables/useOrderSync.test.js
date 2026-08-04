import { describe, it, expect } from 'vitest'
import { useOrderSync } from './useOrderSync'

describe('useOrderSync', () => {
  it('returns orders ref, isConnected ref, reconnect and refresh functions', () => {
    const { orders, isConnected, reconnect, refresh } = useOrderSync()
    expect(orders).toBeDefined()
    expect(Array.isArray(orders.value)).toBe(true)
    expect(isConnected).toBeDefined()
    expect(typeof reconnect).toBe('function')
    expect(typeof refresh).toBe('function')
  })
})
