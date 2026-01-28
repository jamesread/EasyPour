import { ref, onMounted } from 'vue'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { EasyPourService } from '../../gen/easypour/v1/easypour_pb.js'

const username = ref('')
const isAdmin = ref(false)
const oauthProviders = ref([])

const transport = createConnectTransport({
  baseUrl: `${window.location.protocol}//${window.location.hostname}:${window.location.port}`,
})
const client = createClient(EasyPourService, transport)

// Map backend OAuthProvider to Picocrank Login format: { id, name, authUrl, class }
function toLoginProvider(p) {
  if (!p || !p.id || !p.name || !p.authUrl) return null
  return { id: p.id, name: p.name, authUrl: p.authUrl, class: 'neutral' }
}

async function fetchCurrentUser() {
  try {
    const res = await client.getCurrentUser({})
    if (res?.isAuthenticated && res?.username) {
      username.value = res.username
      isAdmin.value = !!res?.isAdmin
    } else {
      username.value = ''
      isAdmin.value = false
    }
    const list = res?.oauthProviders ?? []
    oauthProviders.value = list.map(toLoginProvider).filter(Boolean)
  } catch {
    username.value = ''
    isAdmin.value = false
    oauthProviders.value = []
  }
}

async function logout() {
  const base = `${window.location.protocol}//${window.location.hostname}:${window.location.port}`
  try {
    await fetch(`${base}/logout`, {
      method: 'POST',
      credentials: 'include',
    })
  } finally {
    username.value = ''
    isAdmin.value = false
  }
}

export function useCurrentUser() {
  onMounted(fetchCurrentUser)
  return { username, isAdmin, oauthProviders, refresh: fetchCurrentUser, logout }
}
