import { describe, it, expect } from 'vitest'
import { useInit } from './useInit'

describe('useInit', () => {
  it('returns version, siteTitle, features, oauthProviders refs and refresh function', () => {
    const { version, siteTitle, features, oauthProviders, refresh } = useInit()
    expect(version).toBeDefined()
    expect(siteTitle).toBeDefined()
    expect(features).toBeDefined()
    expect(oauthProviders).toBeDefined()
    expect(typeof refresh).toBe('function')
  })
})
