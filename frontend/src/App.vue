<template>
  <Navigation ref="navigation">
    <Header
      :title="siteTitle"
      logo-url="/logo.svg"
      :username="displayUsername"
      :sidebar-enabled="false"
      :top-bar-enabled="true"
      :navigation="navigation"
      @user-click="onUserClick"
    />
    <RouterView v-if="username" />
    <main v-else class="login-view">
      <article class="login-article-inline">
        <Login
          ref="loginRef"
          :oauth-providers="oauthProviders"
          @local-login="handleLocalLogin"
          @oauth-login="handleOAuthLogin"
        />
      </article>
    </main>
    <footer>
      <span>v{{ version }}</span>
    </footer>
  </Navigation>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { RouterView, useRouter } from 'vue-router'
import { Invoice01Icon, ShoppingCart01Icon, SoftDrink01Icon } from '@hugeicons/core-free-icons'
import Header from 'picocrank/vue/components/Header.vue'
import Login from 'picocrank/vue/components/Login.vue'
import Navigation from 'picocrank/vue/components/Navigation.vue'
import { useBasket } from './composables/useBasket'
import { useCurrentUser } from './composables/useCurrentUser'
import { useInit } from './composables/useInit'

const router = useRouter()
const { basketItems } = useBasket()
const { username, refresh } = useCurrentUser()
const { version, siteTitle, oauthProviders } = useInit()
const loginRef = ref(null)
const navigation = ref(null)

const displayUsername = computed(() => username.value || 'guest')
const basketCount = computed(() => basketItems.value.length)

function onUserClick() {
  if (username.value) {
    router.push('/profile')
  }
}

function setupNavigation() {
  if (!navigation.value) return
  navigation.value.clearNavigationLinks()
  if (!username.value) return
  navigation.value.addNavigationLink({
    name: 'Menu',
    title: 'Menu',
    path: '/',
    to: '/',
    icon: SoftDrink01Icon,
    type: 'route',
  })
  navigation.value.addNavigationLink({
    name: 'Orders',
    title: 'My Orders',
    path: '/orders',
    to: '/orders',
    icon: Invoice01Icon,
    type: 'route',
  })
  navigation.value.addNavigationLink({
    name: 'Basket',
    title: 'Basket',
    path: '/basket',
    to: '/basket',
    icon: ShoppingCart01Icon,
    type: 'route',
    count: basketCount.value > 0 ? basketCount.value : null,
  })
}

watch([username, basketCount], setupNavigation)
onMounted(setupNavigation)

async function handleLocalLogin(credentials) {
  try {
    const base = `${window.location.protocol}//${window.location.hostname}:${window.location.port}`
    const res = await fetch(`${base}/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({
        username: credentials.username,
        password: credentials.password,
      }),
    })
    if (res.ok) {
      refresh()
    } else {
      const msg = res.status === 401 ? 'Invalid username or password.' : 'Login failed.'
      loginRef.value?.setLocalLoginError(msg)
    }
  } catch {
    loginRef.value?.setLocalLoginError('Login failed.')
  }
}

function handleOAuthLogin(provider) {
  if (provider?.authUrl) {
    window.location.href = provider.authUrl
  }
}
</script>

<style scoped>
.login-view {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}

.login-article-inline {
  padding: 2rem 2.5rem;
  max-width: 420px;
  width: 100%;
  background: white;
  border-radius: 0.5rem;
  box-shadow: 0 0 1em rgba(0, 0, 0, 0.1);
}
</style>
