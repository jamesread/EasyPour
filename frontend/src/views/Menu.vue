<template>
  <main class="menu-page">
    <div class="menu-main">
      <div class="">
        <div v-if="loading" class="loading">Loading menu...</div>
        <div v-else-if="error" class="error">{{ error }}</div>
        <div v-else class="menu-container">
          <div v-if="isAdmin" class="admin-bar">
            <button v-if="editMode" type="button" class="good" @click="openAddItem">Add item</button>
          </div>
          <section
            v-for="group in categoriesWithItems"
            :key="group.category"
            class="menu-category"
          >
            <h2 class="category-title">{{ group.categoryDisplay }}</h2>
            <div class="menu-items">
              <div
                v-for="item in group.items"
                :key="item.id"
                class="menu-item"
                @click="selectItem(item)"
              >
              <div class="item-image">
                <img
                  v-if="item.imageUrl"
                  :src="item.imageUrl"
                  :alt="item.name"
                />
                <svg v-else class="placeholder-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
                  <path d="M12 18h24v18H12z" stroke-linecap="round" stroke-linejoin="round"/>
                  <path d="M16 18V14a4 4 0 0 1 8 0v4M20 10v2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
              </div>
              <div class="item-content">
                <h3>{{ item.name }}</h3>
                <p>{{ item.description }}</p>
                <div v-if="isAdmin && editMode" class="item-actions" @click.stop>
                  <button type="button" class="neutral" aria-label="Edit" @click="openEditItem(item)">
                    <HugeiconsIcon :icon="Edit01Icon" class="item-action-icon" />
                  </button>
                  <button type="button" class="bad" aria-label="Delete" @click="confirmDelete(item)">
                    <HugeiconsIcon :icon="Delete01Icon" class="item-action-icon" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </section>

          <div v-if="lastOrder" class="order-success">
            <h3>✅ Order Placed!</h3>
            <p><strong>Order ID:</strong> {{ lastOrder.orderId }}</p>
            <p><strong>Status:</strong> {{ lastOrder.status }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Desktop: right sidebar -->
    <aside class="basket-sidebar" aria-label="Your order">
      <div class="basket-preview-header">
        <h3>Your order</h3>
        <span v-if="basketItems.length > 0" class="basket-count-badge">{{ basketItems.length }}</span>
      </div>
      <div v-if="basketItems.length === 0" class="basket-preview-empty">
        <p>Basket is empty</p>
        <p class="basket-preview-hint">Tap a menu item to add it</p>
      </div>
      <ul v-else class="basket-preview-list">
        <li v-for="(item, index) in basketItems" :key="index" class="basket-preview-item">
          <div class="basket-preview-item-image">
            <img v-if="item.imageUrl" :src="item.imageUrl" :alt="item.name" />
            <HugeiconsIcon v-else :icon="DrinkIcon" class="basket-preview-icon" aria-hidden="true" />
          </div>
          <div class="basket-preview-item-info">
            <span class="basket-preview-item-name">{{ item.name }}</span>
            <span class="basket-preview-item-opts">
              {{ item.addSugar ? (item.sugarType === 'diabetes' ? 'Diabetic sugar' : `${item.sugarAmount ?? 1} sugar`) : 'No sugar' }}{{ item.addMilk ? ' · Milk' : '' }}
            </span>
          </div>
          <div class="basket-preview-item-actions">
            <button type="button" class="basket-preview-btn-edit" aria-label="Edit" @click="openEditBasketItem(index)">
              <HugeiconsIcon :icon="Edit01Icon" class="basket-preview-action-icon" />
            </button>
            <button type="button" class="basket-preview-btn-remove" aria-label="Remove" @click="removeFromBasket(index)">
              <HugeiconsIcon :icon="Delete01Icon" class="basket-preview-action-icon" />
            </button>
          </div>
        </li>
      </ul>
      <div v-if="basketItems.length > 0" class="basket-preview-footer">
        <button type="button" class="basket-preview-checkout" @click="$router.push('/basket')">
          View basket ({{ basketItems.length }})
        </button>
      </div>
    </aside>

    <!-- Mobile: bottom drawer toggle + drawer (teleported to body so fixed positioning is viewport-relative) -->
    <Teleport to="body">
      <button
        type="button"
        class="basket-drawer-toggle"
        :class="{ 'has-items': basketItems.length > 0 }"
        :aria-expanded="basketDrawerOpen"
        aria-label="Open order"
        @click="basketDrawerOpen = !basketDrawerOpen"
      >
        <HugeiconsIcon :icon="ShoppingCart01Icon" class="basket-drawer-toggle-icon" />
        <span>Basket</span>
        <span v-if="basketItems.length > 0" class="basket-drawer-toggle-count">{{ basketItems.length }}</span>
      </button>
      <div class="basket-drawer-backdrop" :class="{ open: basketDrawerOpen }" aria-hidden="true" @click="basketDrawerOpen = false" />
      <div class="basket-drawer" :class="{ open: basketDrawerOpen }" role="dialog" aria-label="Your order">
      <div class="basket-drawer-handle" aria-hidden="true" />
      <div class="basket-drawer-header">
        <h3>Your order</h3>
        <button type="button" class="basket-drawer-close" aria-label="Close" @click="basketDrawerOpen = false">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
        </button>
      </div>
      <div v-if="basketItems.length === 0" class="basket-preview-empty">
        <p>Basket is empty</p>
        <p class="basket-preview-hint">Tap a menu item to add it</p>
      </div>
      <ul v-else class="basket-preview-list basket-drawer-list">
        <li v-for="(item, index) in basketItems" :key="index" class="basket-preview-item">
          <div class="basket-preview-item-image">
            <img v-if="item.imageUrl" :src="item.imageUrl" :alt="item.name" />
            <HugeiconsIcon v-else :icon="DrinkIcon" class="basket-preview-icon" aria-hidden="true" />
          </div>
          <div class="basket-preview-item-info">
            <span class="basket-preview-item-name">{{ item.name }}</span>
            <span class="basket-preview-item-opts">
              {{ item.addSugar ? (item.sugarType === 'diabetes' ? 'Diabetic sugar' : `${item.sugarAmount ?? 1} sugar`) : 'No sugar' }}{{ item.addMilk ? ' · Milk' : '' }}
            </span>
          </div>
          <div class="basket-preview-item-actions">
            <button type="button" class="basket-preview-btn-edit" aria-label="Edit" @click="openEditBasketItem(index); basketDrawerOpen = false">
              <HugeiconsIcon :icon="Edit01Icon" class="basket-preview-action-icon" />
            </button>
            <button type="button" class="basket-preview-btn-remove" aria-label="Remove" @click="removeFromBasket(index)">
              <HugeiconsIcon :icon="Delete01Icon" class="basket-preview-action-icon" />
            </button>
          </div>
        </li>
      </ul>
      <div v-if="basketItems.length > 0" class="basket-drawer-footer">
        <button type="button" class="basket-preview-checkout" @click="$router.push('/basket'); basketDrawerOpen = false">
          View basket ({{ basketItems.length }})
        </button>
      </div>
    </div>
    </Teleport>
  </main>

  <CustomizeItem
    :menu-item="selectedMenuItemForCustomize"
    :open="customizeDialogOpen"
    :edit-basket-item="editingBasketItem"
    :edit-basket-index="editingBasketIndex"
    @close="closeCustomizeDialog"
    @add-to-basket="handleAddToBasket"
  />

  <dialog ref="itemDialogRef" class="item-dialog">
    <article class="item-article">
      <h2>{{ editingItem ? 'Edit item' : 'Add item' }}</h2>
      <form @submit.prevent="saveItem" class="item-form">
        <label>Image</label>
        <div class="item-image-section">
          <div
            class="image-drop-zone"
            :class="{ 'drop-zone-active': dropZoneActive, 'drop-zone-uploading': imageUploading }"
            @dragover.prevent="dropZoneActive = true"
            @dragleave="dropZoneActive = false"
            @drop.prevent="handleImageDrop"
            @click="imageFileInputRef?.click()"
          >
            <img
              v-if="itemForm.imageUrl"
              :src="itemForm.imageUrl"
              :alt="itemForm.name || 'Item'"
              class="item-form-image"
            />
            <svg v-else class="image-placeholder" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
              <path d="M12 18h24v18H12z" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M16 18V14a4 4 0 0 1 8 0v4M20 10v2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            <span class="drop-zone-hint">{{ imageUploading ? 'Uploading…' : 'Drop image here or click to choose' }}</span>
          </div>
          <input
            ref="imageFileInputRef"
            type="file"
            accept="image/*"
            class="image-file-input"
            @change="handleImageFileSelect"
          />
        </div>

        <label for="item-name">Name</label>
        <input
          id="item-name"
          v-model="itemForm.name"
          type="text"
          placeholder="e.g. Espresso, Sandwich, …"
          required
        />

        <label for="item-description">Description</label>
        <input
          id="item-description"
          v-model="itemForm.description"
          type="text"
          placeholder="e.g. Freshly brewed"
          required
        />

        <label for="item-category">Category</label>
        <input
          id="item-category"
          v-model="itemForm.category"
          type="text"
          list="category-list"
          placeholder="e.g. Drinks, Food (type to add new)"
        />
        <datalist id="category-list">
          <option v-for="c in existingCategories" :key="c" :value="c" />
        </datalist>

        <label for="item-supports-sugar">Supports sugar</label>
        <input
          id="item-supports-sugar"
          v-model="itemForm.supportsSugar"
          type="checkbox"
        />

        <label for="item-supports-milk">Supports milk</label>
        <input
          id="item-supports-milk"
          v-model="itemForm.supportsMilk"
          type="checkbox"
        />

        <p v-if="itemFormError" class="form-error">{{ itemFormError }}</p>

        <fieldset>
          <button type="submit" class="good" :disabled="itemFormSaving">Save</button>
          <button type="button" class="neutral" @click="closeItemDialog">Cancel</button>
        </fieldset>
      </form>
    </article>
  </dialog>

  <dialog ref="deleteDialogRef" class="delete-dialog">
    <article class="delete-article">
      <h2>Delete item?</h2>
      <p v-if="itemToDelete">Delete “{{ itemToDelete.name }}”?</p>
      <div class="form-actions">
        <button type="button" class="neutral" @click="closeDeleteDialog">Cancel</button>
        <button type="button" class="bad" @click="doDelete">Delete</button>
      </div>
    </article>
  </dialog>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { HugeiconsIcon } from '@hugeicons/vue'
import { Delete01Icon, DrinkIcon, Edit01Icon, ShoppingCart01Icon } from '@hugeicons/core-free-icons'
import { EasyPourService } from '../../gen/easypour/v1/easypour_pb.js'
import CustomizeItem from '../CustomizeItem.vue'
import { useBasket } from '../composables/useBasket'
import { useCurrentUser } from '../composables/useCurrentUser'
import { useEditMode } from '../composables/useEditMode'

const transport = createConnectTransport({
  baseUrl: `${window.location.protocol}//${window.location.hostname}:${window.location.port}`,
})
const client = createClient(EasyPourService, transport)

const { addToBasket, basketItems, removeFromBasket, updateBasketItem } = useBasket()
const { isAdmin } = useCurrentUser()
const { editMode } = useEditMode()

const loading = ref(true)
const error = ref(null)
const menuItems = ref([])
const selectedItemId = ref(null)
const customizeDialogOpen = ref(false)
const lastOrder = ref(null)

const itemDialogRef = ref(null)
const deleteDialogRef = ref(null)
const editingItem = ref(null)
const itemForm = ref({
  name: '',
  description: '',
  imageUrl: '',
  category: '',
  supportsSugar: true,
  supportsMilk: true,
})
const itemFormError = ref('')
const itemFormSaving = ref(false)
const itemToDelete = ref(null)
const dropZoneActive = ref(false)
const imageUploading = ref(false)
const imageFileInputRef = ref(null)

const selectedMenuItem = computed(() => {
  return menuItems.value.find(item => item.id === selectedItemId.value)
})

const editingBasketIndex = ref(null)
const basketDrawerOpen = ref(false)

const editingBasketItem = computed(() => {
  const i = editingBasketIndex.value
  if (i == null || i < 0 || i >= basketItems.value.length) return null
  return basketItems.value[i]
})

const selectedMenuItemForCustomize = computed(() => {
  if (editingBasketIndex.value != null && editingBasketItem.value) {
    const b = editingBasketItem.value
    return { id: b.id, name: b.name, supportsSugar: b.supportsSugar ?? true, supportsMilk: b.supportsMilk ?? true, imageUrl: b.imageUrl }
  }
  return selectedMenuItem.value
})

const UNCategorized = 'Uncategorized'

const categoriesWithItems = computed(() => {
  const byCat = new Map()
  for (const item of menuItems.value) {
    const cat = (item.category && item.category.trim()) || UNCategorized
    if (!byCat.has(cat)) byCat.set(cat, [])
    byCat.get(cat).push(item)
  }
  const cats = [...byCat.keys()].sort((a, b) => {
    if (a === UNCategorized) return 1
    if (b === UNCategorized) return -1
    return a.localeCompare(b)
  })
  return cats.map(category => ({
    category,
    categoryDisplay: category === UNCategorized ? UNCategorized : category,
    items: byCat.get(category),
  }))
})

const existingCategories = computed(() => {
  const set = new Set()
  for (const item of menuItems.value) {
    const c = (item.category && item.category.trim()) || ''
    if (c) set.add(c)
  }
  return [...set].sort()
})

async function fetchMenu() {
  try {
    const response = await client.getMenu({})
    menuItems.value = response.items ?? []
    error.value = null
  } catch (err) {
    error.value = `Failed to load menu: ${err.message}`
  }
}

function selectItem(item) {
  selectedItemId.value = item.id
  customizeDialogOpen.value = true
}

function closeCustomizeDialog() {
  customizeDialogOpen.value = false
  selectedItemId.value = null
  editingBasketIndex.value = null
}

function openEditBasketItem(index) {
  editingBasketIndex.value = index
  customizeDialogOpen.value = true
}

function handleAddToBasket(item, index) {
  if (index !== undefined && index !== null) {
    updateBasketItem(index, item)
  } else {
    addToBasket(item)
  }
  closeCustomizeDialog()
}

function openAddItem() {
  editingItem.value = null
  itemForm.value = {
    name: '',
    description: '',
    imageUrl: '',
    category: 'Drinks',
    supportsSugar: true,
    supportsMilk: true,
  }
  itemFormError.value = ''
  itemDialogRef.value?.showModal()
}

function openEditItem(item) {
  editingItem.value = item
  itemForm.value = {
    name: item.name,
    description: item.description,
    imageUrl: item.imageUrl || '',
    category: item.category || '',
    supportsSugar: item.supportsSugar,
    supportsMilk: item.supportsMilk,
  }
  itemFormError.value = ''
  itemDialogRef.value?.showModal()
}

function getUploadUrl() {
  return `${window.location.protocol}//${window.location.hostname}:${window.location.port}/upload`
}

async function uploadImage(file) {
  if (!file || !file.type.startsWith('image/')) return
  imageUploading.value = true
  itemFormError.value = ''
  try {
    const form = new FormData()
    form.append('image', file)
    const res = await fetch(getUploadUrl(), {
      method: 'POST',
      credentials: 'include',
      body: form,
    })
    if (!res.ok) {
      const text = await res.text()
      throw new Error(text || `Upload failed (${res.status})`)
    }
    const data = await res.json()
    const url = data?.url
    if (url) itemForm.value = { ...itemForm.value, imageUrl: url }
  } catch (err) {
    itemFormError.value = err.message || 'Image upload failed'
  } finally {
    imageUploading.value = false
    dropZoneActive.value = false
  }
}

function handleImageDrop(e) {
  const file = e.dataTransfer?.files?.[0]
  if (file) uploadImage(file)
}

function handleImageFileSelect(e) {
  const file = e.target.files?.[0]
  if (file) uploadImage(file)
  e.target.value = ''
}


function closeItemDialog() {
  itemDialogRef.value?.close()
}

async function saveItem() {
  itemFormError.value = ''
  itemFormSaving.value = true
  try {
    if (editingItem.value) {
      await client.updateMenuItem({
        item: {
          id: editingItem.value.id,
          name: itemForm.value.name,
          description: itemForm.value.description,
          imageUrl: itemForm.value.imageUrl || '',
          category: (itemForm.value.category && itemForm.value.category.trim()) || '',
          supportsSugar: itemForm.value.supportsSugar,
          supportsMilk: itemForm.value.supportsMilk,
        },
      })
    } else {
      await client.createMenuItem({
        item: {
          name: itemForm.value.name,
          description: itemForm.value.description,
          imageUrl: itemForm.value.imageUrl || '',
          category: (itemForm.value.category && itemForm.value.category.trim()) || '',
          supportsSugar: itemForm.value.supportsSugar,
          supportsMilk: itemForm.value.supportsMilk,
        },
      })
    }
    closeItemDialog()
    await fetchMenu()
  } catch (err) {
    itemFormError.value = err.message || 'Save failed'
  } finally {
    itemFormSaving.value = false
  }
}

function confirmDelete(item) {
  itemToDelete.value = item
  deleteDialogRef.value?.showModal()
}

function closeDeleteDialog() {
  itemToDelete.value = null
  deleteDialogRef.value?.close()
}

async function doDelete() {
  if (!itemToDelete.value?.id) return
  try {
    await client.deleteMenuItem({ id: itemToDelete.value.id })
    closeDeleteDialog()
    await fetchMenu()
  } catch (err) {
    error.value = err.message || 'Delete failed'
  }
}

onMounted(async () => {
  await fetchMenu()
  loading.value = false
})
</script>

<style scoped>
.admin-bar {
  margin-bottom: 1rem;
}

.menu-category {
  margin-bottom: 2rem;
}

.menu-category:last-of-type {
  margin-bottom: 0;
}

.category-title {
  margin: 0 0 0.75rem 0;
  font-size: 1.25rem;
  font-weight: 600;
}

.menu-items {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1rem;
}

.menu-item {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 1rem;
  border: 1px solid #ccc;
  padding: 1rem;
  border-radius: 0.5rem;
  cursor: pointer;
}

.menu-item:hover {
  background-color: #f0f0f0;
}

.menu-item .item-image {
  flex-shrink: 0;
  width: 56px;
  height: 56px;
  border-radius: 0.375rem;
  background: #eee;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.menu-item .item-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.menu-item .placeholder-icon {
  width: 32px;
  height: 32px;
  color: #999;
}

.menu-item .item-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.menu-item h3 {
  margin: 0;
}

.menu-item p {
  margin: 0;
}

.menu-item .item-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.menu-item .item-actions button {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.menu-item .item-action-icon {
  width: 1rem;
  height: 1rem;
}

.item-dialog,
.delete-dialog {
  padding: 0;
  border: none;
  border-radius: 0.5rem;
  box-shadow: 0 0 1em rgba(0, 0, 0, 0.2);
}

.item-article,
.delete-article {
  padding: 1.5rem;
  min-width: 320px;
}

.item-image-section {
  margin-bottom: 1rem;
  position: relative;
}

.image-drop-zone {
  border: 2px dashed var(--border-color, #ccc);
  border-radius: 0.5rem;
  min-height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  overflow: hidden;
  position: relative;
  background: var(--input-bg, #fafafa);
}

.image-drop-zone.drop-zone-active {
  border-color: var(--primary-color, #333);
  background: #eee;
}

.image-drop-zone.drop-zone-uploading {
  pointer-events: none;
  opacity: 0.8;
}

.image-drop-zone .item-form-image {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-drop-zone .image-placeholder {
  width: 48px;
  height: 48px;
  color: #999;
}

.drop-zone-hint {
  position: relative;
  z-index: 1;
  padding: 0.5rem 1rem;
  background: rgba(255, 255, 255, 0.9);
  border-radius: 0.25rem;
  font-size: 0.9rem;
  color: var(--text-muted, #666);
}

.image-drop-zone:has(.item-form-image) .drop-zone-hint {
  opacity: 0.9;
}

.image-file-input {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
  pointer-events: none;
}

/* Form structure matches picocrank FormExample.vue: label + control siblings, actions in fieldset */
.item-form label {
  display: block;
  margin-bottom: 0.25rem;
}

.item-form input[type="text"],
.item-form input[type="checkbox"],
.item-form select {
  display: block;
  margin-bottom: 1rem;
}

.item-form input[type="checkbox"] {
  display: inline-block;
  margin-bottom: 0.5rem;
  margin-right: 0.5rem;
}

.item-form input[type="text"],
.item-form select {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--border-color, #ccc);
  border-radius: 0.25rem;
  width: 100%;
  box-sizing: border-box;
}

.item-form fieldset {
  margin-top: 1rem;
  padding: 0.75rem 0 0 0;
  border: none;
  display: flex;
  gap: 0.5rem;
}

.form-error {
  color: #c00;
  font-size: 0.9rem;
  margin: 0 0 0.5rem 0;
}

/* Menu page layout + live basket preview */
.menu-page {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.menu-main {
  flex: 1;
  min-width: 0;
}

/* Right sidebar: visible on desktop (768px+), hidden on mobile */
/* Override femtocrank's visibility: hidden so the sidebar is visible */
.basket-sidebar {
  visibility: visible;
  border-left: 2px solid #94a3b8;
  box-shadow: -2px 0 8px rgba(0, 0, 0, 0.08);
  box-sizing: border-box;
}

@media (max-width: 767px) {
  .basket-sidebar {
    display: none !important;
  }
}

@media (min-width: 768px) {
  .menu-page {
    flex-direction: row;
  }

  .basket-sidebar {
    position: sticky;
    top: 0;
    align-self: flex-start;
    max-height: 100vh;
    overflow: hidden;
  }

  .basket-drawer-toggle,
  .basket-drawer-backdrop,
  .basket-drawer {
    display: none !important;
  }
}

.basket-preview-header {
  padding: 1rem 1.25rem;
  border-bottom: 1px solid #e2e8f0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.basket-preview-header h3 {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 600;
}

.basket-count-badge {
  background: #0f172a;
  color: #fff;
  font-size: 0.8rem;
  font-weight: 600;
  padding: 0.2rem 0.5rem;
  border-radius: 999px;
}

.basket-preview-empty {
  padding: 1.5rem 1.25rem;
  text-align: center;
  color: #64748b;
  font-size: 0.95rem;
}

.basket-preview-empty p {
  margin: 0 0 0.25rem 0;
}

.basket-preview-hint {
  font-size: 0.85rem !important;
  color: #94a3b8 !important;
}

.basket-preview-list {
  list-style: none;
  margin: 0;
  padding: 0.75rem 0;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}

.basket-preview-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.6rem 1.25rem;
  border-bottom: 1px solid #f1f5f9;
}

.basket-preview-item:last-child {
  border-bottom: none;
}

.basket-preview-item-image {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: 0.375rem;
  background: #e2e8f0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.basket-preview-item-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.basket-preview-icon {
  width: 1.25rem;
  height: 1.25rem;
  color: #94a3b8;
}

.basket-preview-item-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
}

.basket-preview-item-name {
  font-weight: 600;
  font-size: 0.9rem;
  color: #0f172a;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.basket-preview-item-opts {
  font-size: 0.75rem;
  color: #64748b;
}

.basket-preview-item-actions {
  display: flex;
  gap: 0.25rem;
  flex-shrink: 0;
}

.basket-preview-btn-edit,
.basket-preview-btn-remove {
  padding: 0.35rem;
  border: none;
  background: transparent;
  border-radius: 0.25rem;
  cursor: pointer;
  color: #64748b;
}

.basket-preview-btn-edit:hover {
  background: #e2e8f0;
  color: #0f172a;
}

.basket-preview-btn-remove:hover {
  background: #fee2e2;
  color: #b91c1c;
}

.basket-preview-action-icon {
  width: 1rem;
  height: 1rem;
}

.basket-preview-footer {
  padding: 1rem 1.25rem;
  border-top: 1px solid #e2e8f0;
}

.basket-preview-checkout {
  width: 100%;
  padding: 0.65rem 1rem;
  font-size: 0.95rem;
  font-weight: 600;
  color: #fff;
  background: #0f172a;
  border: none;
  border-radius: 0.5rem;
  cursor: pointer;
}

.basket-preview-checkout:hover {
  background: #1e293b;
}

/* Mobile: bottom drawer (teleported to body so position:fixed is viewport-relative) */
.basket-drawer-toggle {
  display: sticky;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  position: fixed;
  bottom: 2em;
  left: 1em;
  padding: 0.75rem 1.25rem;
  font-size: 0.95rem;
  font-weight: 600;
  color: #0f172a;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 999px;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.15);
  cursor: pointer;
  z-index: 40;
}

.basket-drawer-toggle.has-items {
  background: #0f172a;
  color: #fff;
  border-color: #0f172a;
}

.basket-drawer-toggle-icon {
  width: 1.25rem;
  height: 1.25rem;
}

.basket-drawer-toggle-count {
  background: rgba(255, 255, 255, 0.25);
  padding: 0.15rem 0.45rem;
  border-radius: 999px;
  font-size: 0.85rem;
}

.basket-drawer-toggle.has-items .basket-drawer-toggle-count {
  background: rgba(255, 255, 255, 0.3);
}

.basket-drawer-backdrop {
  display: none;
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  z-index: 50;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.basket-drawer-backdrop.open {
  display: block;
  opacity: 1;
}

.basket-drawer {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 51;
  background: #fff;
  border-radius: 1rem 1rem 0 0;
  box-shadow: 0 -4px 20px rgba(0, 0, 0, 0.15);
  max-height: 70vh;
  display: flex;
  flex-direction: column;
  transform: translateY(100%);
  transition: transform 0.25s ease;
}

.basket-drawer.open {
  transform: translateY(0);
}

.basket-drawer-handle {
  width: 2.5rem;
  height: 0.25rem;
  background: #cbd5e1;
  border-radius: 999px;
  margin: 0.5rem auto 0;
}

.basket-drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1.25rem 0.5rem;
  border-bottom: 1px solid #e2e8f0;
}

.basket-drawer-header h3 {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 600;
}

.basket-drawer-close {
  padding: 0.5rem;
  border: none;
  background: transparent;
  border-radius: 0.25rem;
  cursor: pointer;
  color: #64748b;
}

.basket-drawer-close:hover {
  background: #f1f5f9;
  color: #0f172a;
}

.basket-drawer-close svg {
  width: 1.25rem;
  height: 1.25rem;
}

.basket-drawer-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-bottom: 0.5rem;
}

.basket-drawer-footer {
  padding: 1rem 1.25rem;
  padding-bottom: max(1rem, env(safe-area-inset-bottom));
  border-top: 1px solid #e2e8f0;
}

@media (min-width: 768px) {
  .basket-drawer-toggle {
    display: none;
  }
}
</style>
