import { describe, it, expect } from 'vitest'
import { countKitchenItems, formatKitchenOptions, isActiveKitchenStatus } from './kitchenItems'

describe('isActiveKitchenStatus', () => {
  it('excludes delivered-like statuses', () => {
    expect(isActiveKitchenStatus('delivered')).toBe(false)
    expect(isActiveKitchenStatus('completed')).toBe(false)
    expect(isActiveKitchenStatus('done')).toBe(false)
  })

  it('includes pending and preparing', () => {
    expect(isActiveKitchenStatus('pending')).toBe(true)
    expect(isActiveKitchenStatus('preparing')).toBe(true)
  })
})

describe('formatKitchenOptions', () => {
  it('formats sugar and milk options', () => {
    expect(formatKitchenOptions({ addSugar: false, addMilk: false })).toBe('No sugar, No milk')
    expect(formatKitchenOptions({ addSugar: true, sugarAmount: 1, addMilk: true })).toBe('1 sugar, Milk')
    expect(formatKitchenOptions({ addSugar: true, sugarAmount: 0, addMilk: false })).toBe('Diabetic sugar, No milk')
  })
})

describe('countKitchenItems', () => {
  it('counts only non-delivered orders and groups by item+options', () => {
    const menuById = new Map([
      ['latte', { name: 'Latte' }],
      ['toast', { name: 'Marmite toast' }],
    ])
    const rows = countKitchenItems(
      [
        { menuItemId: 'latte', status: 'pending', addSugar: true, sugarAmount: 1, addMilk: true },
        { menuItemId: 'latte', status: 'preparing', addSugar: true, sugarAmount: 1, addMilk: true },
        { menuItemId: 'latte', status: 'pending', addSugar: false, addMilk: true },
        { menuItemId: 'toast', status: 'pending', addSugar: false, addMilk: false },
        { menuItemId: 'toast', status: 'delivered', addSugar: false, addMilk: false },
      ],
      menuById,
    )
    expect(rows).toEqual([
      {
        menuItemId: 'latte',
        name: 'Latte',
        imageUrl: undefined,
        options: '1 sugar, Milk',
        count: 2,
      },
      {
        menuItemId: 'latte',
        name: 'Latte',
        imageUrl: undefined,
        options: 'No sugar, Milk',
        count: 1,
      },
      {
        menuItemId: 'toast',
        name: 'Marmite toast',
        imageUrl: undefined,
        options: 'No sugar, No milk',
        count: 1,
      },
    ])
  })

  it('returns empty when there are no active orders', () => {
    expect(countKitchenItems([{ menuItemId: 'latte', status: 'delivered' }])).toEqual([])
  })
})
