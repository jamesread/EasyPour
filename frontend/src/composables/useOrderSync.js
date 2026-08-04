import { ref, onUnmounted } from 'vue'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { EasyPourService } from '../../gen/easypour/v1/easypour_pb.js'

const POLL_INTERVAL_MS = 5000

function createOrderClient() {
  const transport = createConnectTransport({
    baseUrl: `${window.location.protocol}//${window.location.hostname}:${window.location.port}`,
  })
  return createClient(EasyPourService, transport)
}

const orders = ref(/** @type {import('../../gen/easypour/v1/easypour_pb').Order[]} */ ([]))
const isConnected = ref(false)
let client = null
let eventSource = null
let pollTimer = null
let unmountCount = 0

async function refresh() {
  if (!client) client = createOrderClient()
  try {
    const res = await client.listOrders({})
    orders.value = res.orders ?? []
  } catch {
    orders.value = []
  }
}

function mergeStatusUpdate(orderId, status) {
  const list = orders.value
  const idx = list.findIndex((o) => o?.orderId === orderId)
  if (idx >= 0 && list[idx]) {
    const copy = [...list]
    copy[idx] = { ...copy[idx], status }
    orders.value = copy
  }
}

function startPolling() {
  if (pollTimer) return
  pollTimer = setInterval(refresh, POLL_INTERVAL_MS)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function connectSSE() {
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
  stopPolling()
  const url = '/orders/events'
  const es = new EventSource(url)
  eventSource = es
  es.onopen = () => {
    isConnected.value = true
  }
  es.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data ?? '{}')
      if (data.type === 'status_update' && data.order_id && data.status) {
        mergeStatusUpdate(data.order_id, data.status)
      }
    } catch {
      // ignore parse errors
    }
  }
  es.onerror = () => {
    isConnected.value = false
    es.close()
    eventSource = null
    startPolling()
  }
}

function reconnect() {
  isConnected.value = false
  stopPolling()
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
  refresh().finally(() => connectSSE())
}

function teardown() {
  unmountCount--
  if (unmountCount <= 0) {
    unmountCount = 0
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
    stopPolling()
    isConnected.value = false
  }
}

export function useOrderSync() {
  unmountCount++
  if (unmountCount === 1) {
    refresh().finally(() => connectSSE())
  }

  onUnmounted(teardown)

  return { orders, isConnected, reconnect, refresh }
}
