import { ref, onMounted } from 'vue'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { EasyPourService } from '../../gen/easypour/v1/easypour_pb.js'

const version = ref('')
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

async function fetchInit() {
  try {
    const res = await client.init({})
    version.value = res?.version ?? ''
    const list = res?.oauthProviders ?? []
    oauthProviders.value = list.map(toLoginProvider).filter(Boolean)
  } catch {
    version.value = ''
    oauthProviders.value = []
  }
}

export function useInit() {
  onMounted(fetchInit)
  return { version, oauthProviders, refresh: fetchInit }
}
