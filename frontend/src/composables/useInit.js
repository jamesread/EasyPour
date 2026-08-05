import { ref, onMounted } from 'vue'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { EasyPourService } from '../../gen/easypour/v1/easypour_pb.js'

const version = ref('')
const siteTitle = ref('EasyPour')
const features = ref({})
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

export async function loadInit() {
  try {
    const res = await client.init({})
    version.value = res?.version ?? ''
    siteTitle.value = res?.siteTitle || 'EasyPour'
    features.value = res?.features ?? {}
    const list = res?.oauthProviders ?? []
    oauthProviders.value = list.map(toLoginProvider).filter(Boolean)
    if (typeof document !== 'undefined' && siteTitle.value) {
      document.title = siteTitle.value
    }
  } catch {
    version.value = ''
    siteTitle.value = 'EasyPour'
    features.value = {}
    oauthProviders.value = []
  }
}

export function useInit() {
  onMounted(loadInit)
  return { version, siteTitle, features, oauthProviders, refresh: loadInit }
}
