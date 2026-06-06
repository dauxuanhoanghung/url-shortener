import { createRouter, createWebHistory } from "vue-router";

import { useAuthStore } from "../stores/authStore";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      component: () => import("@/layouts/MarketingLayout.vue"),
      children: [
        {
          path: "",
          name: "landing",
          component: () => import("@/views/(marketing)/LandingView.vue"),
        },
        {
          path: "pricing",
          name: "pricing",
          component: () => import("@/views/(marketing)/PricingView.vue"),
        },
      ],
    },
    {
      path: "/",
      component: () => import("@/layouts/AuthLayout.vue"),
      children: [
        {
          path: "login",
          name: "login",
          component: () => import("@/views/(auth)/LoginView.vue"),
          meta: { guest: true },
        },
        {
          path: "register",
          name: "register",
          component: () => import("@/views/(auth)/RegisterView.vue"),
          meta: { guest: true },
        },
        {
          path: "forgot-password",
          name: "forgot-password",
          component: () => import("@/views/(auth)/ForgotPasswordView.vue"),
          meta: { guest: true },
        },
        {
          path: "reset-password",
          name: "reset-password",
          component: () => import("@/views/(auth)/ResetPasswordView.vue"),
          // not guest-only: someone whose session is still alive may follow
          // the reset link; don't bounce them back to dashboard.
        },
        {
          path: "verify-email",
          name: "verify-email",
          component: () => import("@/views/(auth)/VerifyEmailView.vue"),
          // verify link should work whether the user is logged in or not.
        },
      ],
    },
    {
      path: "/",
      component: () => import("@/layouts/DashboardLayout.vue"),
      meta: { requiresAuth: true },
      children: [
        {
          path: "dashboard",
          name: "dashboard",
          component: () => import("@/views/(dashboard)/DashboardView.vue"),
        },
      ],
    },
    {
      path: "/admin",
      component: () => import("@/layouts/AdminLayout.vue"),
      meta: { requiresAuth: true, requiresAdmin: true },
      redirect: { name: "admin-users" },
      children: [
        {
          path: "users",
          name: "admin-users",
          component: () => import("@/views/(admin)/AdminUsersView.vue"),
        },
        {
          path: "plans",
          name: "admin-plans",
          component: () => import("@/views/(admin)/AdminPlansView.vue"),
        },
        {
          path: "audit",
          name: "admin-audit",
          component: () => import("@/views/(admin)/AdminAuditView.vue"),
        },
      ],
    },
  ],
});

router.beforeEach((to) => {
  const auth = useAuthStore();
  if (to.matched.some((r) => r.meta.requiresAuth) && !auth.isAuthenticated) {
    return { name: "login" };
  }
  if (to.matched.some((r) => r.meta.requiresAdmin) && !auth.isAdmin) {
    return { name: "dashboard" };
  }
  if (to.meta.guest && auth.isAuthenticated) {
    return { name: "dashboard" };
  }
});

export default router;
