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

      <div class="dialog-content">
        <h3 v-if="hasCustomizationOptions">Customize your {{ itemName }}:</h3>

        <FormField v-if="menuItem?.supportsSugar" label="Sugar" fake>
          <RadioGroup
            v-model="sugarChoice"
            name="customize-sugar"
            aria-label="Sugar"
            :options="sugarOptions"
          />
        </FormField>

        <FormField v-if="menuItem?.supportsMilk" label="Milk" fake>
          <RadioGroup
            v-model="addMilk"
            name="customize-milk"
            variant="boolean"
            aria-label="Milk"
            :options="milkOptions"
          />
        </FormField>

        <div class="toolbar">
          <button type="button" class="good" @click="addToBasket" :disabled="adding">
            {{ adding ? 'Updating...' : (props.editBasketIndex != null ? 'Update in Basket' : 'Add to Basket') }}
          </button>
        </div>
      </div>
    </article>
  </dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import FormField from 'picocrank/vue/components/FormField.vue'
import RadioGroup from 'picocrank/vue/components/RadioGroup.vue'

const props = defineProps({
  menuItem: Object,
  open: Boolean,
  editBasketItem: { type: Object, default: null },
  editBasketIndex: { type: Number, default: undefined },
})

const emit = defineEmits(['close', 'add-to-basket'])

const dialogRef = ref(null)
const sugarChoice = ref('none')
const addMilk = ref(false)
const adding = ref(false)

const sugarOptions = [
  { label: 'None', value: 'none' },
  { label: '1 spoon', value: '1' },
  { label: '2 spoons', value: '2' },
  { label: 'Diabetes', value: 'diabetes' },
]

const milkOptions = [
  { label: 'No', value: false },
  { label: 'Yes', value: true },
]

const itemName = computed(() => {
  return props.menuItem?.name || 'Item'
})

const hasCustomizationOptions = computed(() => {
  return !!(props.menuItem?.supportsSugar || props.menuItem?.supportsMilk)
})

function sugarChoiceFromBasketItem(item) {
  if (!item?.addSugar) return 'none'
  if (item.sugarType === 'diabetes') return 'diabetes'
  return String(item.sugarAmount || 1)
}

watch(() => props.open, (newVal) => {
  if (newVal && dialogRef.value) {
    dialogRef.value.showModal()
    if (props.editBasketItem != null) {
      sugarChoice.value = sugarChoiceFromBasketItem(props.editBasketItem)
      addMilk.value = !!props.editBasketItem.addMilk
    } else {
      sugarChoice.value = 'none'
      addMilk.value = !!props.menuItem?.supportsMilk
    }
  } else if (!newVal && dialogRef.value) {
    dialogRef.value.close()
  }
})

const close = () => {
  emit('close')
}

const addToBasket = () => {
  if (!props.menuItem) return

  adding.value = true

  const choice = sugarChoice.value
  const withSugar = choice !== 'none'
  const isDiabetes = choice === 'diabetes'

  const basketItem = {
    id: props.menuItem.id,
    name: props.menuItem.name,
    imageUrl: props.menuItem?.imageUrl,
    addSugar: withSugar,
    addMilk: addMilk.value,
    sugarAmount: isDiabetes || !withSugar ? 0 : Number(choice),
    sugarType: isDiabetes ? 'diabetes' : 'normal',
    milkAmount: addMilk.value ? 50 : 0,
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
  max-width: 480px;
  box-sizing: border-box;
}

@media (max-width: 767px) {
  .customize-dialog {
    max-width: 100vw;
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

.dialog-content {
  padding: 0 1rem 1rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.toolbar {
  display: flex;
  gap: 1em;
}
</style>
