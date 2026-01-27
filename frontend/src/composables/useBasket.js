import { ref } from 'vue'

const basketItems = ref([])

export function useBasket() {
  const addToBasket = (item) => {
    basketItems.value.push(item)
  }

  const removeFromBasket = (index) => {
    basketItems.value.splice(index, 1)
  }

  const updateBasketItem = (index, item) => {
    if (index >= 0 && index < basketItems.value.length) {
      basketItems.value[index] = item
    }
  }

  const clearBasket = () => {
    basketItems.value = []
  }

  return {
    basketItems,
    addToBasket,
    removeFromBasket,
    updateBasketItem,
    clearBasket,
  }
}


