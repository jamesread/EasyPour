<template>
  <dialog ref="dialogRef" class="customize-dialog">
    <article class="dialog-article">
      <button type="button" class="close-btn" @click="close" aria-label="Close">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M18 6L6 18M6 6l12 12" />
        </svg>
      </button>
      <div class="dialog-header">
        <div class="item-image">
          <img
            v-if="menuItem?.imageUrl"
            :src="menuItem.imageUrl"
            :alt="itemName"
          />
          <svg v-else class="placeholder-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
            <path d="M12 18h24v18H12z" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M16 18V14a4 4 0 0 1 8 0v4M20 10v2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </div>
      </div>
      
	  <div class = "dialog-content">
        <h3>Customize your {{ itemName }}:</h3>
      <div v-if="menuItem?.supportsSugar" class="option-group">
        <div class="option-row">
          <button type="button" class="icon-toggle" :class="{ active: addSugar }" :aria-pressed="addSugar" aria-label="Add Sugar" @click="toggleSugar">
            <svg v-if="!addSugar" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M12 5v14M5 12h14" /></svg>
            <svg v-else xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M5 12h14" /></svg>
          </button>
          <span :class="{ checked: addSugar }">Add Sugar</span>
        </div>
        <div v-if="addSugar" class="sugar-control">
          <input
            type="range"
            v-model.number="sugarAmount"
            min="1"
            max="3"
            step="1"
            class="sugar-slider"
          />
          <div class="sugar-labels">
            <span :class="{ active: sugarAmount === 1 }">1 spoon</span>
            <span :class="{ active: sugarAmount === 2 }">2 spoons</span>
            <span :class="{ active: sugarAmount === 3 }">diabetes</span>
          </div>
        </div>
      </div>

      <div v-if="menuItem?.supportsMilk" class="option-group">
        <div class="option-row">
          <button type="button" class="icon-toggle" :class="{ active: addMilk }" :aria-pressed="addMilk" aria-label="Add Milk" @click="addMilk = !addMilk">
            <svg v-if="!addMilk" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M12 5v14M5 12h14" /></svg>
            <svg v-else xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M5 12h14" /></svg>
          </button>
          <span :class="{ checked: addMilk }">Add Milk</span>
        </div>
      </div>

      <div class="toolbar" style="display: flex; gap: 1em">
        <button class="good" @click="addToBasket" :disabled="adding">
          {{ adding ? 'Updating...' : (props.editBasketIndex != null ? 'Update in Basket' : 'Add to Basket') }}
        </button>
      </div>

	  </div>
    </article>
  </dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  menuItem: Object,
  open: Boolean,
  editBasketItem: { type: Object, default: null },
  editBasketIndex: { type: Number, default: undefined },
})

const emit = defineEmits(['close', 'add-to-basket'])

const dialogRef = ref(null)
const addSugar = ref(false)
const addMilk = ref(false)
const sugarAmount = ref(1) // 1 = 1 spoon, 2 = 2 spoons, 3 = diabetes
const adding = ref(false)

const itemName = computed(() => {
  return props.menuItem?.name || 'Item'
})

watch(() => props.open, (newVal) => {
  if (newVal && dialogRef.value) {
    dialogRef.value.showModal()
    if (props.editBasketItem != null) {
      addSugar.value = !!props.editBasketItem.addSugar
      addMilk.value = !!props.editBasketItem.addMilk
      sugarAmount.value = props.editBasketItem.sugarType === 'diabetes' ? 3 : (props.editBasketItem.sugarAmount || 1)
    } else {
      addSugar.value = false
      addMilk.value = false
      sugarAmount.value = 1
    }
  } else if (!newVal && dialogRef.value) {
    dialogRef.value.close()
  }
})

const toggleSugar = () => {
  addSugar.value = !addSugar.value
  updateSugar()
}

const updateSugar = () => {
  if (!addSugar.value) {
    sugarAmount.value = 1 // Reset to default when unchecked
  }
}

const close = () => {
  emit('close')
}

const addToBasket = () => {
  if (!props.menuItem) return
  
  adding.value = true
  
  // sugarAmount: 1 = 1 spoon, 2 = 2 spoons, 3 = diabetes
  const sugarValue = addSugar.value ? sugarAmount.value : 0
  const isDiabetes = sugarValue === 3
  
  const basketItem = {
    id: props.menuItem.id,
    name: props.menuItem.name,
    imageUrl: props.menuItem?.imageUrl,
    addSugar: addSugar.value,
    addMilk: addMilk.value,
    sugarAmount: isDiabetes ? 0 : sugarValue,
    sugarType: isDiabetes ? 'diabetes' : 'normal',
    milkAmount: addMilk.value ? 50 : 0, // Default milk amount when checked
    supportsSugar: props.menuItem.supportsSugar,
    supportsMilk: props.menuItem.supportsMilk,
  }
  
  const editIndex = props.editBasketIndex
  if (editIndex !== undefined && editIndex !== null) {
    emit('add-to-basket', basketItem, editIndex)
  } else {
    emit('add-to-basket', basketItem)
  }
  adding.value = false
  close()
}
</script>

<style scoped>
.customize-dialog {
  padding: 0;
  border: none;
  max-width: 100vw;
  box-sizing: border-box;
}

@media (max-width: 767px) {
  .customize-dialog {
    margin-top: 1em;
    margin-bottom: auto;
    max-height: calc(100vh - 1em);
    overflow-y: auto;
  }
}

.dialog-article {
  position: relative;
  max-width: 100%;
  min-width: 0;
}

.close-btn {
  position: absolute;
  top: 0.75rem;
  right: 0.75rem;
  width: 2rem;
  height: 2rem;
  padding: 0;
  border: none;
  background: transparent;
  color: #666;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.25rem;
  z-index: 1;
}

.close-btn:hover {
  color: #000;
  background: #eee;
}

.close-btn svg {
  width: 1.25rem;
  height: 1.25rem;
}

.dialog-header {
  margin-bottom: 1rem;
}

.dialog-header .item-image {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  aspect-ratio: 2 / 1;
  border-radius: 0.375rem;
  background: #eee;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  margin-bottom: 1rem;
}

.dialog-header .item-image img {
  max-width: 100%;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.dialog-header .placeholder-icon {
  width: 64px;
  height: 64px;
  color: #999;
}

.dialog-header h3 {
  margin: 0;
}

.option-group {
  margin-bottom: 1rem;
}

.option-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.icon-toggle {
  width: 1.75rem;
  height: 1.75rem;
  padding: 0;
  border: 1px solid #ccc;
  background: #fff;
  color: #333;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.25rem;
  flex-shrink: 0;
}

.icon-toggle:hover {
  border-color: #999;
  background: #f5f5f5;
}

.icon-toggle.active {
  border-color: #333;
  background: #eee;
  color: #000;
}

.icon-toggle svg {
  width: 1rem;
  height: 1rem;
}

.option-row span.checked {
  font-weight: 600;
  color: #000;
}

.sugar-control {
  margin-top: 0.5rem;
}

.sugar-slider {
  width: 100%;
  margin-bottom: 0.5rem;
}

.sugar-labels {
  display: flex;
  justify-content: space-between;
  font-size: 0.9rem;
}

.sugar-labels span {
  color: #666;
}

.sugar-labels span.active {
  color: #000;
  font-weight: bold;
}

.dialog-content {
  padding: 0 1rem 1rem 1rem;
}
</style>

