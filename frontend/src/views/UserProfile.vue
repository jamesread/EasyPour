<template>
  <main class="user-profile-page">
    <section class="with-header-and-content profile-section">
      <div class="section-header">
        <h2>Profile</h2>
        <button type="button" class="neutral" @click="$router.push('/')" aria-label="Back to menu">Back</button>
      </div>
      <div class="section-content padding">
        <div v-if="!username" class="profile-guest">
          <p>You are not logged in.</p>
          <p>Log in to see your profile.</p>
        </div>
        <template v-else>
          <dl class="profile-details">
            <dt>Username</dt>
            <dd>{{ username }}</dd>
            <dt v-if="isAdmin">Role</dt>
            <dd v-if="isAdmin">Admin</dd>
          </dl>
          <template v-if="isAdmin">
            <hr class="profile-divider" />
            <FormField label="Edit mode" fake>
              <RadioGroup
                v-model="editMode"
                name="edit-mode"
                variant="boolean"
                aria-label="Edit mode"
                :options="editModeOptions"
              />
              <p class="edit-mode-hint">When on, you can add, edit, and delete menu items on the Menu page.</p>
            </FormField>
          </template>
          <div class="profile-logout">
            <button type="button" class="neutral" @click="handleLogout">Log out</button>
          </div>
        </template>
      </div>
    </section>
    <section v-if="isAdmin" class="with-header-and-content admin-panel-section">
      <div class="section-header">
        <h2>Admin Control Panel</h2>
      </div>
      <div class="section-content padding">
        <Navigation ref="adminNavigation">
          <NavigationGrid compact />
        </Navigation>
      </div>
    </section>
  </main>
</template>

<script setup>
import { nextTick, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import FormField from 'picocrank/vue/components/FormField.vue'
import RadioGroup from 'picocrank/vue/components/RadioGroup.vue'
import Navigation from 'picocrank/vue/components/Navigation.vue'
import NavigationGrid from 'picocrank/vue/components/NavigationGrid.vue'
import { useCurrentUser } from '../composables/useCurrentUser'
import { useEditMode } from '../composables/useEditMode'

const router = useRouter()
const { username, isAdmin, logout } = useCurrentUser()
const { editMode } = useEditMode()
const adminNavigation = ref(null)

const editModeOptions = [
  { label: 'On', value: true },
  { label: 'Off', value: false },
]

async function setupAdminNavigation() {
  await nextTick()
  const nav = adminNavigation.value
  if (!nav || !isAdmin.value) return
  nav.clearNavigationLinks()
  nav.addRouterLink('AdminOrders', 'Admin Orders', {
    description: 'Acknowledge and manage all orders',
  })
  nav.addRouterLink('AdminKitchen', 'Kitchen View', {
    description: 'Item counts from open orders',
  })
  nav.addRouterLink('SettingsAdmin', 'Settings', {
    description: 'Site-wide configuration variables',
  })
  nav.addRouterLink('AdminApprise', 'Apprise', {
    description: 'Test Apprise notifications for new orders',
  })
}

async function handleLogout() {
  await logout()
  router.push('/')
}

watch([isAdmin, adminNavigation], () => setupAdminNavigation())
onMounted(() => setupAdminNavigation())
</script>

<style scoped>
.user-profile-page {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 1rem;
  gap: 1rem;
}

.profile-section,
.admin-panel-section {
  width: 100%;
  max-width: 36rem;
}

.profile-guest {
  color: var(--karma-info-fg, #666);
}

.profile-details {
  margin: 0 0 1rem;
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 0.25rem 1rem;
}

.profile-details dt {
  margin: 0;
  font-weight: 600;
}

.profile-details dd {
  margin: 0;
}

.profile-divider {
  margin: 1rem 0;
  border: none;
  border-top: 1px solid var(--border-color);
}

.edit-mode-hint {
  margin: 0.5rem 0 0;
  font-size: 0.9rem;
  color: var(--karma-info-fg, #666);
}

.profile-logout {
  margin-top: 1rem;
}
</style>
