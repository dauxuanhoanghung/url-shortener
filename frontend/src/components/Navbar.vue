<script setup lang="ts">
import { useAuthStore } from '../stores/authStore'
import { useRouter } from 'vue-router'

const auth = useAuthStore()
const router = useRouter()

function handleLogout() {
  auth.logout()
  router.push({ name: 'landing' })
}
</script>

<template>
  <nav class="navbar">
    <div class="navbar-container">
      <router-link to="/" class="navbar-brand">Snip.ly</router-link>
      <div class="navbar-links">
        <template v-if="auth.isAuthenticated">
          <router-link to="/dashboard" class="nav-link">Dashboard</router-link>
          <span class="nav-user">{{ auth.user?.email }}</span>
          <button class="btn btn-outline" @click="handleLogout">Log out</button>
        </template>
        <template v-else>
          <router-link to="/login" class="nav-link">Log in</router-link>
          <router-link to="/register" class="btn btn-primary">Sign up free</router-link>
        </template>
      </div>
    </div>
  </nav>
</template>

<style scoped>
.navbar {
  background: #fff;
  border-bottom: 1px solid #e5e7eb;
  padding: 0 1.5rem;
  height: 64px;
  display: flex;
  align-items: center;
}

.navbar-container {
  max-width: 1200px;
  width: 100%;
  margin: 0 auto;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.navbar-brand {
  font-size: 1.5rem;
  font-weight: 700;
  color: #4f46e5;
  text-decoration: none;
}

.navbar-links {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.nav-link {
  color: #374151;
  text-decoration: none;
  font-weight: 500;
  font-size: 0.95rem;
}

.nav-link:hover {
  color: #4f46e5;
}

.nav-user {
  color: #6b7280;
  font-size: 0.9rem;
}

.btn {
  padding: 0.5rem 1.25rem;
  border-radius: 6px;
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  text-decoration: none;
  border: none;
  transition: background 0.15s, color 0.15s;
}

.btn-primary {
  background: #4f46e5;
  color: #fff;
}

.btn-primary:hover {
  background: #4338ca;
}

.btn-outline {
  background: transparent;
  color: #374151;
  border: 1px solid #d1d5db;
}

.btn-outline:hover {
  background: #f3f4f6;
}
</style>
