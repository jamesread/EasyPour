<template>
  <main class="user-profile-page">
    <section class="with-header-and-content profile-section">
      <div class="section-header">
        <h2>Profile</h2>
        <button type="button" class="button" @click="$router.push('/')" aria-label="Back to menu">Back</button>
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
            <label class="edit-mode-toggle">
              <input v-model="editMode" type="checkbox" />
              <span>Edit mode</span>
            </label>
            <p class="edit-mode-hint">When on, you can add, edit, and delete menu items on the Menu page.</p>
          </template>
        </template>
      </div>
    </section>
    <section v-if="username" class="with-header-and-content profile-actions-section">
      <div class="section-header">
        <h2>Actions</h2>
      </div>
      <div class="section-content padding">
        <div class="profile-actions-buttons">
          <router-link v-if="isAdmin" to="/orders" class="button">Admin Control Panel</router-link>
          <button type="button" class="button" @click="handleLogout">Log out</button>
        </div>
      </div>
    </section>
  </main>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useCurrentUser } from '../composables/useCurrentUser'
import { useEditMode } from '../composables/useEditMode'

const router = useRouter()
const { username, isAdmin, logout } = useCurrentUser()
const { editMode } = useEditMode()

async function handleLogout() {
  await logout()
  router.push('/')
}
</script>

<style scoped>
.user-profile-page {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 1rem;
  gap: 1rem;
}

.profile-section {
  width: 100%;
  max-width: 36rem;
}

.profile-actions-section {
  width: 100%;
  max-width: 36rem;
}

.profile-actions-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
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

.edit-mode-toggle {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  user-select: none;
}

.edit-mode-toggle input {
  margin: 0;
}

.edit-mode-hint {
  margin: 0.25rem 0 0.5rem;
  font-size: 0.9rem;
  color: var(--karma-info-fg, #666);
}
</style>
