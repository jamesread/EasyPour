<template>
  <Header
    title="EasyPour"
    logo-url="/logo.svg"
    :username="displayUsername"
    :sidebar-enabled="false"
  >
    <template #user-info>
      <div class="user-info">
        <button
          v-if="!username"
          type="button"
          class="guest-btn"
          aria-label="Log in"
          @click="showLoginForm = true"
        >
          guest
        </button>
        <button
          v-else
          type="button"
          class="username-btn"
          @click="$router.push('/profile')"
        >
          {{ username }}
        </button>
      </div>
    </template>
    <template #toolbar>
      <div class="header-actions">
        <template v-if="username">
          <button @click="$router.push('/')" class="menu-button" aria-label="View menu">
            <HugeiconsIcon :icon="SoftDrink01Icon" class="menu-icon" />
            <span class="header-btn-text">Menu</span>
          </button>
          <button @click="$router.push('/orders')" class="orders-button" aria-label="View orders">
            <HugeiconsIcon :icon="Invoice01Icon" class="orders-icon" />
            <span class="header-btn-text">Orders</span>
          </button>
        </template>
        <button v-if="username" @click="$router.push('/basket')" class="basket-button" aria-label="View basket">
          <HugeiconsIcon :icon="ShoppingCart01Icon" class="basket-icon" />
          <span class="header-btn-text">Basket</span>
          <span v-if="basketItems.length > 0" class="basket-count">{{ basketItems.length }}</span>
        </button>
      </div>
    </template>
  </Header>
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
  <dialog v-if="username" ref="loginDialogRef" class="login-dialog">
    <article class="login-article">
      <button type="button" class="close-btn" @click="showLoginForm = false" aria-label="Close">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M18 6L6 18M6 6l12 12" />
        </svg>
      </button>
      <Login
        ref="loginRef"
        :oauth-providers="oauthProviders"
        @local-login="handleLocalLogin"
        @oauth-login="handleOAuthLogin"
      />
    </article>
  </dialog>
  <footer>
	  <span>v{{ version }}</span>
  </footer>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { RouterView } from 'vue-router'
import { HugeiconsIcon } from '@hugeicons/vue'
import { Invoice01Icon, ShoppingCart01Icon, SoftDrink01Icon } from '@hugeicons/core-free-icons'
import Header from 'picocrank/vue/components/Header.vue'
import Login from 'picocrank/vue/components/Login.vue'
import { useBasket } from './composables/useBasket'
import { useCurrentUser } from './composables/useCurrentUser'

const version = __APP_VERSION__
const { basketItems } = useBasket()
const { username, refresh, oauthProviders } = useCurrentUser()
const showLoginForm = ref(false)
const loginDialogRef = ref(null)
const loginRef = ref(null)

const displayUsername = computed(() => username.value || 'guest')

watch(showLoginForm, (open) => {
  if (!loginDialogRef.value) return
  if (open) {
    loginDialogRef.value.showModal()
    loginRef.value?.resetLocalForm()
  } else {
    loginDialogRef.value.close()
  }
})

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
      showLoginForm.value = false
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
.header-actions {
  display: flex;
  gap: .2rem;
  align-items: center;
}

.menu-button,
.orders-button,
.basket-button {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border: none;
  background: none;
  cursor: pointer;
  color: white;
}

.menu-button:hover,
.orders-button:hover,
.basket-button:hover {
  color: #c2d7e4;
}

.menu-icon,
.orders-icon,
.basket-icon {
  width: 1.5rem;
  height: 1.5rem;
}

.basket-button {
  position: relative;
}

.basket-count {
  position: absolute;
  top: 0;
  right: 0;
  background: red;
  color: white;
  border-radius: 50%;
  width: 1.2rem;
  height: 1.2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.7rem;
  font-weight: bold;
}

@media (max-width: 767px) {
  .header-btn-text {
    display: none;
  }
}

.user-info {
  color: white;
}

.username-btn {
  background: none;
  border: none;
  padding: 0;
  color: white;
  cursor: pointer;
  font: inherit;
}

.username-btn:hover {
  color: #c2d7e4;
}

.guest-btn {
  background: none;
  border: none;
  padding: 0;
  color: white;
  cursor: pointer;
  font: inherit;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.guest-btn:hover {
  color: #c2d7e4;
}

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

.login-dialog {
  padding: 0;
  border: none;
  border-radius: 0.5rem;
  box-shadow: 0 0 1em rgba(0, 0, 0, 0.2);
}

.login-article {
  position: relative;
  padding: 2rem 2.5rem 1.5rem 1.5rem;
  max-width: 420px;
}

.login-article .close-btn {
  position: absolute;
  top: 0.75rem;
  right: 0.75rem;
  width: 2rem;
  height: 2rem;
  padding: 0;
  border: none;
  background: transparent;
  color: #666;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.25rem;
  z-index: 1;
}

.login-article .close-btn:hover {
  color: #000;
  background: #eee;
}

.login-article .close-btn svg {
  width: 1.25rem;
  height: 1.25rem;
}

</style>
