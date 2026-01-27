import { ref, onMounted } from 'vue'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { EasyPourService } from '../../gen/easypour/v1/easypour_pb.js'

const username = ref('')
const isAdmin = ref(false)

const transport = createConnectTransport({
  baseUrl: `${window.location.protocol}//${window.location.hostname}:${window.location.port}`,
})
const client = createClient(EasyPourService, transport)

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
  } catch {
    username.value = ''
    isAdmin.value = false
  }
}

export function useCurrentUser() {
  onMounted(fetchCurrentUser)
  return { username, isAdmin, refresh: fetchCurrentUser }
}
