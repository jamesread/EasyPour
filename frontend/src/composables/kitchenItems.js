/**
 * @typedef {import('../../gen/easypour/v1/easypour_pb').Order} Order
 * @typedef {Object} KitchenItemCount
 * @property {string} menuItemId
 * @property {string} name
 * @property {string} [imageUrl]
 * @property {string} options
 * @property {number} count
 */

/**
 * @param {string | null | undefined} status
 * @returns {boolean}
 */
export function isActiveKitchenStatus(status) {
  const s = (status ?? '').toLowerCase()
  return s !== 'delivered' && s !== 'completed' && s !== 'done'
}

/**
 * @param {Order} order
 * @returns {string}
 */
export function formatKitchenOptions(order) {
  let sugar = 'No sugar'
  if (order?.addSugar) {
    const amount = Number(order.sugarAmount ?? 0)
    sugar = amount <= 0 ? 'Diabetic sugar' : `${amount} sugar`
  }
  const milk = order?.addMilk ? 'Milk' : 'No milk'
  return `${sugar}, ${milk}`
}

/**
 * Aggregate counts of items across non-delivered orders.
 * @param {Order[]} orders
 * @param {Map<string, { name?: string, imageUrl?: string }> | Record<string, { name?: string, imageUrl?: string }> | null | undefined} [menuById]
 * @returns {KitchenItemCount[]}
 */
export function countKitchenItems(orders, menuById) {
  /** @type {Map<string, KitchenItemCount>} */
  const byKey = new Map()

  for (const order of orders ?? []) {
    if (!isActiveKitchenStatus(order?.status)) continue
    const menuItemId = order.menuItemId || 'unknown'
    const options = formatKitchenOptions(order)
    const key = `${menuItemId}|${options}`
    const existing = byKey.get(key)
    if (existing) {
      existing.count += 1
      continue
    }
    let name = menuItemId
    let imageUrl
    if (menuById) {
      const entry = menuById instanceof Map ? menuById.get(menuItemId) : menuById[menuItemId]
      if (entry?.name) name = entry.name
      if (entry?.imageUrl) imageUrl = entry.imageUrl
    }
    byKey.set(key, {
      menuItemId,
      name,
      imageUrl,
      options,
      count: 1,
    })
  }

  return [...byKey.values()].sort((a, b) => {
    if (b.count !== a.count) return b.count - a.count
    return a.name.localeCompare(b.name) || a.options.localeCompare(b.options)
  })
}
