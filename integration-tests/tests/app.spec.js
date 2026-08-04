import { describe, it, before, after } from 'mocha'
import assert from 'assert'
import { Builder } from 'selenium-webdriver'

const baseUrl = process.env.EASYPOUR_BASE_URL || 'http://localhost:9654'

describe('App (selenium)', function () {
  let driver

  before(async function () {
    driver = await new Builder().forBrowser('chrome').build()
  })

  after(async function () {
    if (driver) await driver.quit()
  })

  it('loads backend root', async function () {
    await driver.get(baseUrl)
    const title = await driver.getTitle()
    assert.ok(title !== undefined, 'page should have a title')
  })
})
