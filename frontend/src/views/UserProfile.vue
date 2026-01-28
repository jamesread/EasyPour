<template>
  <main class="user-profile-page">
    <section class="padding">
      <div class="profile-header">
        <h2>Profile</h2>
        <button @click="$router.push('/')" aria-label="Back to menu">Back</button>
      </div>

      <div v-if="!username" class="profile-guest">
        <p>You are not logged in.</p>
        <p>Log in to see your profile.</p>
      </div>

      <div v-else class="profile-content">
        <dl class="profile-details">
          <dt>Username</dt>
          <dd>{{ username }}</dd>
          <dt v-if="isAdmin">Role</dt>
          <dd v-if="isAdmin">Admin</dd>
        </dl>
        <div class="logout-section">
          <button type="button" class="logout-btn" @click="handleLogout">Log out</button>
        </div>
        <div v-if="isAdmin" class="edit-mode-section">
          <label class="edit-mode-toggle">
            <input v-model="editMode" type="checkbox" />
            <span>Edit mode</span>
          </label>
          <p class="edit-mode-hint">When on, you can add, edit, and delete menu items on the Menu page.</p>
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
.profile-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.profile-header h2 {
  margin: 0;
}

.profile-guest {
  color: var(--text-muted, #666);
}

.profile-content {
  max-width: 400px;
}

.profile-details {
  margin: 0;
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 0.5rem 1.5rem;
}

.profile-details dt {
  margin: 0;
  font-weight: 600;
}

.profile-details dd {
  margin: 0;
}

.logout-section {
  margin-top: 1.5rem;
}

.logout-btn {
  padding: 0.5rem 1rem;
  font-size: 1rem;
  cursor: pointer;
  background: var(--button-secondary-bg, #f0f0f0);
  color: var(--text, #333);
  border: 1px solid var(--border-color, #ccc);
  border-radius: 4px;
}

.logout-btn:hover {
  background: var(--button-secondary-hover-bg, #e5e5e5);
}

.edit-mode-section {
  margin-top: 1.5rem;
  padding-top: 1.5rem;
  border-top: 1px solid var(--border-color, #eee);
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
  margin: 0.5rem 0 0 0;
  font-size: 0.9rem;
  color: var(--text-muted, #666);
}
</style>
