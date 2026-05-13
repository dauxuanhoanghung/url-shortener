<script setup lang="ts">
import { useAuthStore } from '../stores/authStore'

const auth = useAuthStore()
</script>

<template>
  <div class="landing">
    <section class="hero">
      <h1>Shorten your links,<br />expand your reach</h1>
      <p class="hero-subtitle">
        Create short, memorable links in seconds. Track clicks and manage all
        your URLs from one dashboard.
      </p>
      <div class="hero-actions">
        <router-link
          v-if="!auth.isAuthenticated"
          to="/register"
          class="btn btn-primary btn-lg"
        >
          Get started free
        </router-link>
        <router-link v-else to="/dashboard" class="btn btn-primary btn-lg">
          Go to Dashboard
        </router-link>
      </div>
    </section>

    <section class="features">
      <div class="feature-card">
        <div class="feature-icon">&#9889;</div>
        <h3>Lightning Fast</h3>
        <p>Redirects in under 100ms with Redis-powered caching.</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon">&#128274;</div>
        <h3>Secure</h3>
        <p>URL validation, rate limiting, and JWT authentication.</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon">&#128200;</div>
        <h3>Scalable</h3>
        <p>Free plan with 100 URLs. Pro plan with 10,000 URLs.</p>
      </div>
    </section>

    <section class="plans">
      <h2>Simple pricing</h2>
      <div class="plan-cards">
        <div class="plan-card">
          <h3>Free</h3>
          <div class="plan-price">$0<span>/month</span></div>
          <ul>
            <li>Up to 100 short URLs</li>
            <li>Public redirect</li>
            <li>Click tracking</li>
          </ul>
          <router-link
            v-if="!auth.isAuthenticated"
            to="/register"
            class="btn btn-outline btn-block"
          >
            Sign up
          </router-link>
        </div>
        <div class="plan-card plan-card--featured">
          <h3>Pro</h3>
          <div class="plan-price">$9<span>/month</span></div>
          <ul>
            <li>Up to 10,000 short URLs</li>
            <li>Public redirect</li>
            <li>Click tracking</li>
            <li>Priority support</li>
          </ul>
          <router-link
            v-if="!auth.isAuthenticated"
            to="/register"
            class="btn btn-primary btn-block"
          >
            Get started
          </router-link>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.landing {
  max-width: 1000px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}

.hero {
  text-align: center;
  padding: 4rem 0 3rem;
}

.hero h1 {
  font-size: 3rem;
  font-weight: 800;
  color: #111827;
  line-height: 1.15;
  margin-bottom: 1rem;
}

.hero-subtitle {
  font-size: 1.2rem;
  color: #6b7280;
  max-width: 520px;
  margin: 0 auto 2rem;
  line-height: 1.6;
}

.hero-actions {
  display: flex;
  justify-content: center;
  gap: 1rem;
}

.btn {
  padding: 0.6rem 1.5rem;
  border-radius: 8px;
  font-weight: 600;
  font-size: 1rem;
  cursor: pointer;
  text-decoration: none;
  border: none;
  transition: background 0.15s;
  display: inline-block;
}

.btn-lg {
  padding: 0.85rem 2rem;
  font-size: 1.1rem;
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
  background: #f9fafb;
}

.btn-block {
  display: block;
  width: 100%;
  text-align: center;
}

/* Features */
.features {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 2rem;
  padding: 3rem 0;
}

.feature-card {
  text-align: center;
  padding: 2rem 1.5rem;
  border-radius: 12px;
  background: #f9fafb;
}

.feature-icon {
  font-size: 2rem;
  margin-bottom: 0.75rem;
}

.feature-card h3 {
  font-size: 1.15rem;
  color: #111827;
  margin-bottom: 0.5rem;
}

.feature-card p {
  color: #6b7280;
  font-size: 0.95rem;
  line-height: 1.5;
}

/* Plans */
.plans {
  text-align: center;
  padding: 3rem 0;
}

.plans h2 {
  font-size: 2rem;
  font-weight: 700;
  margin-bottom: 2rem;
  color: #111827;
}

.plan-cards {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 2rem;
  max-width: 700px;
  margin: 0 auto;
}

.plan-card {
  padding: 2rem;
  border-radius: 12px;
  border: 1px solid #e5e7eb;
  text-align: left;
}

.plan-card--featured {
  border-color: #4f46e5;
  box-shadow: 0 0 0 2px #4f46e540;
}

.plan-card h3 {
  font-size: 1.2rem;
  color: #111827;
  margin-bottom: 0.5rem;
}

.plan-price {
  font-size: 2.5rem;
  font-weight: 800;
  color: #111827;
  margin-bottom: 1.5rem;
}

.plan-price span {
  font-size: 1rem;
  font-weight: 400;
  color: #6b7280;
}

.plan-card ul {
  list-style: none;
  padding: 0;
  margin-bottom: 1.5rem;
}

.plan-card li {
  padding: 0.4rem 0;
  color: #374151;
  font-size: 0.95rem;
}

.plan-card li::before {
  content: '\2713  ';
  color: #4f46e5;
  font-weight: 700;
}
</style>
