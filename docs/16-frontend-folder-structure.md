# Frontend Folder Structure (Vue 3)

```text
frontend/
├── src/
│   ├── views/
│   │   ├── (marketing)/        # public site
│   │   │   ├── LandingView.vue
│   │   │   └── PricingView.vue
│   │   ├── (auth)/             # login / register / password / verify
│   │   │   ├── LoginView.vue
│   │   │   ├── RegisterView.vue
│   │   │   ├── ForgotPasswordView.vue
│   │   │   ├── ResetPasswordView.vue
│   │   │   └── VerifyEmailView.vue
│   │   ├── (dashboard)/        # authenticated user area
│   │   │   └── DashboardView.vue
│   │   └── (admin)/            # admin-only console pages
│   │       ├── AdminUsersView.vue
│   │       ├── AdminPlansView.vue
│   │       └── AdminAuditView.vue
│   │
│   ├── layouts/                # one <router-view/> wrapper per route group
│   │   ├── MarketingLayout.vue
│   │   ├── AuthLayout.vue      # centred card chrome shared by all (auth) views
│   │   ├── DashboardLayout.vue
│   │   └── AdminLayout.vue     # tabs + page header
│   │
│   ├── components/
│   │   ├── UrlForm.vue
│   │   ├── UrlList.vue
│   │   └── Navbar.vue          # rendered once in App.vue, above every layout
│   │
│   ├── composables/            # reusable composition functions (e.g. useSSE)
│   ├── stores/                 # Pinia stores (authStore, urlStore, …)
│   ├── services/               # axios + per-domain API clients
│   ├── router/                 # hand-written nested route table
│   └── types/                  # TS interfaces mirroring backend DTOs
```

### Route-group folders `(name)/`

The parenthesised folders inside `views/` are **route groups** in the
Next.js / TanStack File Router sense — they organise files but do **not**
appear in the URL. `/login` lives at `views/(auth)/LoginView.vue` and the
URL is still `/login`, not `/auth/login`. The group folder maps 1:1 to a
layout under `src/layouts/`.

We don't use a file-based routing plugin — the mapping is done by hand in
`router/index.ts`. The folder convention is purely for human navigation and
to keep group-shared chrome in one layout file.

When adding a new view:

1. Drop the file into the matching `(group)/` folder.
2. Add a child route under that group's layout in `router/index.ts`.
3. If the new view needs different chrome, create a new layout + group.

---

## `services/api.ts` — axios instance

A single shared `axios` instance with two interceptors:

1. **Request**: attaches `Authorization: Bearer <access_token>` from
   `localStorage` if present.
2. **Response (401 handler)**: on `401` for **non-auth** endpoints, clears
   `localStorage` and hard-redirects to `/login`. Requests to `/auth/*`
   (login, register, refresh, etc.) are **excluded** from the auto-redirect so
   the calling view can show a "wrong credentials" message without the page
   reloading and wiping the form.

When adding a new endpoint family that handles its own 401 (e.g. an optional preview API), extend the `isAuthEndpoint` check in `api.ts`.

---

## `stores/authStore.ts` — session bootstrap

`authStore.init()` is called once from `main.ts`. It restores the session from `localStorage` only if the access token's `exp` claim is still in the future;
expired tokens are cleared so the app loads as logged-out. There is no silent refresh yet — that will be added when `/auth/refresh` ships.

---

## Forms

All forms use **vee-validate + zod**. See
[26-frontend-ui-stack.md → Form validation](26-frontend-ui-stack.md#form-validation-vee-validate--zod)
for the canonical pattern. Key rules: schema-first validation, `novalidate` on
the `<form>`, never reset the form on server error.
