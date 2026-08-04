import { describe, it, expect } from 'vitest'
import { useInit } from './useInit'

describe('useInit', () => {
  it('returns version and oauthProviders refs and refresh function', () => {
    const { version, oauthProviders, refresh } = useInit()
    expect(version).toBeDefined()
    expect(oauthProviders).toBeDefined()
    expect(typeof refresh).toBe('function')
  })
})
