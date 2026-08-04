import { describe, it } from 'mocha'
import assert from 'assert'

const baseUrl = process.env.EASYPOUR_BASE_URL || 'http://localhost:9654'
const initPath = '/easypour.v1.EasyPourService/Init'

describe('Init RPC', function () {
  it('returns version and oauth_providers', async function () {
    const res = await fetch(`${baseUrl}${initPath}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Connect-Protocol-Version': '1',
      },
      body: JSON.stringify({}),
    })
    assert.strictEqual(res.status, 200, 'Init should return 200')
    const body = await res.json()
    assert.ok(body.version, 'version should be set')
    const providers = body.oauthProviders ?? body.oauth_providers ?? []
    assert.ok(Array.isArray(providers), 'oauth_providers/oauthProviders should be array')
  })
})
