<template>
  <main class="apprise-page">
    <Section
      title="Apprise"
      subtitle="Configure Apprise notifications for new orders"
    >
      <template #toolbar>
        <button type="button" class="neutral" @click="$router.push('/profile')" aria-label="Back">Back</button>
      </template>

      <div v-if="!isAdmin" class="apprise-error" role="alert">
        Admin access required.
      </div>
      <div v-else-if="loadError" class="apprise-error" role="alert">
        {{ loadError }}
      </div>
      <div v-else-if="loading" class="apprise-loading" aria-live="polite">
        <p>Loading…</p>
      </div>
      <template v-else>
        <FormField label="Apprise URL">
          <input
            v-model="appriseUrl"
            type="text"
            class="apprise-url-input"
            placeholder="http://localhost:8000/notify/"
            autocomplete="off"
          />
          <p class="apprise-hint">
            Apprise API notify URL. When set, new orders POST a notification here
            (e.g. http://apprise:8000/notify/ or http://apprise:8000/notify/mytag).
          </p>
        </FormField>
        <div class="apprise-actions">
          <button
            type="button"
            class="good"
            :disabled="saving"
            @click="saveSettings"
          >
            {{ saving ? 'Saving…' : 'Save' }}
          </button>
          <button
            type="button"
            class="neutral"
            :disabled="testing || !appriseUrl.trim()"
            @click="sendTestNotification"
          >
            {{ testing ? 'Sending…' : 'Send test notification' }}
          </button>
          <p v-if="saveMessage" class="apprise-message" :class="{ error: saveError }">
            {{ saveMessage }}
          </p>
        </div>
      </template>
    </Section>
  </main>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import FormField from 'picocrank/vue/components/FormField.vue'
import Section from 'picocrank/vue/components/Section.vue'
import { EasyPourService } from '../../gen/easypour/v1/easypour_pb.js'
import { useCurrentUser } from '../composables/useCurrentUser'

function createSettingsClient() {
  const transport = createConnectTransport({
    baseUrl: `${window.location.protocol}//${window.location.hostname}:${window.location.port}`,
  })
  return createClient(EasyPourService, transport)
}

const router = useRouter()
const { isAdmin, refresh: refreshUser } = useCurrentUser()
const client = createSettingsClient()

const loading = ref(true)
const loadError = ref(null)
const appriseUrl = ref('')
const saving = ref(false)
const testing = ref(false)
const saveMessage = ref('')
const saveError = ref(false)

async function loadSettings() {
  loadError.value = null
  try {
    const res = await client.getSettings({})
    appriseUrl.value = res.settings?.appriseUrl ?? ''
  } catch (e) {
    loadError.value = e?.message ?? 'Failed to load settings'
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  if (!isAdmin.value || saving.value) return
  saving.value = true
  saveMessage.value = ''
  saveError.value = false
  try {
    const res = await client.updateSettings({
      settings: { appriseUrl: appriseUrl.value.trim() },
    })
    appriseUrl.value = res.settings?.appriseUrl ?? ''
    saveMessage.value = 'Saved.'
  } catch (e) {
    saveError.value = true
    saveMessage.value = e?.message ?? 'Failed to save settings'
  } finally {
    saving.value = false
  }
}

async function sendTestNotification() {
  if (!isAdmin.value || testing.value) return
  const url = appriseUrl.value.trim()
  if (!url) return
  testing.value = true
  saveMessage.value = ''
  saveError.value = false
  try {
    const res = await client.testAppriseNotification({ appriseUrl: url })
    saveMessage.value = res.message || 'Test notification sent.'
  } catch (e) {
    saveError.value = true
    saveMessage.value = e?.message ?? 'Failed to send test notification'
  } finally {
    testing.value = false
  }
}

onMounted(async () => {
  await refreshUser()
  if (!isAdmin.value) {
    router.replace('/profile')
    return
  }
  await loadSettings()
})
</script>

<style scoped>
.apprise-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 1rem;
}

.apprise-page :deep(section) {
  width: 100%;
  max-width: 36rem;
}

.apprise-error {
  padding: 1rem;
  background: #fef2f2;
  color: #991b1b;
  border-radius: 0.5rem;
  margin-bottom: 1rem;
}

.apprise-loading {
  padding: 2rem;
  text-align: center;
  color: #64748b;
}

.apprise-url-input {
  width: 100%;
  box-sizing: border-box;
}

.apprise-hint {
  margin: 0.5rem 0 0;
  font-size: 0.9rem;
  color: var(--karma-info-fg, #666);
}

.apprise-actions {
  margin-top: 0.75rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
}

.apprise-message {
  margin: 0;
  flex-basis: 100%;
  font-size: 0.9rem;
  color: var(--karma-good-fg, #166534);
}

.apprise-message.error {
  color: var(--karma-bad-fg, #991b1b);
}
</style>
