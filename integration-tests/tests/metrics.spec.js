import { describe, it } from 'mocha'
import assert from 'assert'

const baseUrl = process.env.EASYPOUR_BASE_URL || 'http://localhost:9654'

describe('Metrics endpoint', function () {
  it('serves prometheus metrics', async function () {
    const res = await fetch(`${baseUrl}/metrics`)
    assert.strictEqual(res.status, 200, '/metrics should return 200')
    const body = await res.text()
    assert.ok(body.includes('easypour_orders_created_total'), 'orders created counter should be exposed')
  })
})
