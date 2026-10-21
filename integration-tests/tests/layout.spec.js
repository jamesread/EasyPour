import { describe, it, before, after } from 'mocha'
import assert from 'assert'
import { Builder, By, until } from 'selenium-webdriver'

const baseUrl = process.env.EASYPOUR_BASE_URL || 'http://localhost:9654'

describe('Layout (selenium)', function () {
  let driver

  before(async function () {
    driver = await new Builder().forBrowser('chrome').build()
  })

  after(async function () {
    if (driver) await driver.quit()
  })

  it('does not hide page content under the header', async function () {
    await driver.get(baseUrl)
    const main = await driver.wait(until.elementLocated(By.css('main')), 10000)
    const header = await driver.findElement(By.css('header'))
    const headerRect = await header.getRect()
    const mainRect = await main.getRect()
    assert.ok(
      mainRect.y + 1 >= headerRect.y + headerRect.height,
      `main starts at ${mainRect.y}px but the header ends at ${headerRect.y + headerRect.height}px`,
    )
  })
})
