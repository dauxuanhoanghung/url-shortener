# Frontend — AI Agent Instructions

Scoped instructions for working inside `frontend/`. The root
[`../CLAUDE.md`](../CLAUDE.md) still applies; this file overrides or adds to
it for frontend tasks.

Authoritative docs:

- [docs/16-frontend-folder-structure.md](../docs/16-frontend-folder-structure.md) — folders, route groups, api.ts behaviour
- [docs/26-frontend-ui-stack.md](../docs/26-frontend-ui-stack.md) — UI libs, layouts, forms, Prettier config

---

## 1. Stack at a glance

| Layer     | What we use                                                                              |
| --------- | ---------------------------------------------------------------------------------------- |
| Framework | Vue 3.5 (Composition API, `<script setup lang="ts">`)                                    |
| Build     | Vite 8                                                                                   |
| Router    | Vue Router 4 — hand-written nested layout routes (no file-based plugin)                  |
| State     | Pinia 3                                                                                  |
| HTTP      | Axios via `src/services/api.ts` (shared instance with auth + 401 interceptors)           |
| UI        | shadcn-vue (copied into `src/components/ui/`) + Tailwind CSS v4                          |
| Icons     | `lucide-vue-next`                                                                        |
| Forms     | vee-validate + zod (via `@vee-validate/zod`)                                             |
| Format    | Prettier 3 + `@ianvs/prettier-plugin-sort-imports` + `prettier-plugin-tailwindcss@0.6.5` |
| Git hooks | husky 9 (repo root) + lint-staged 17 (frontend) → pre-commit Prettier on staged files    |

Not in use and not wanted: ESLint, Stylelint, Vuelidate, FormKit, GraphQL,
SWR/TanStack Query, Tailwind v3, Options API.

---

## 2. Folder layout

```text
src/
  layouts/        # one wrapper per route group (Marketing/Auth/Dashboard/Admin)
  views/
    (marketing)/  # parens = route group, NOT a URL segment
    (auth)/
    (dashboard)/
    (admin)/
  components/     # shared components (Navbar, PasswordInput, UrlForm, …)
    ui/          # shadcn-CLI-generated; do not hand-edit
  composables/    # reusable composition functions
  stores/         # Pinia stores
  services/       # api.ts + per-domain API clients
  router/         # hand-written route table
  types/          # TS interfaces mirroring backend DTOs
  lib/            # small utilities (cn(), etc.)
```

The `(group)/` parenthesised folders are a Next.js / TanStack convention
for **route groups** — they organise files but never appear in the URL.
`/login` lives at `views/(auth)/LoginView.vue` and stays `/login`.

---

## 3. Routing rules

- All routes are declared by hand in `src/router/index.ts`.
- Each `(group)/` maps to one layout under `src/layouts/`. Add a child route
  under that layout, never as a top-level sibling.
- `meta.requiresAuth` and `meta.requiresAdmin` go on the **layout route**,
  not each child. The guard uses `to.matched.some(...)` so children inherit.
- `meta.guest` (bounce authenticated users to `/dashboard`) goes on the
  individual auth routes.
- URLs are flat — no `/auth/login`, no `/app/dashboard`. The folder
  grouping is purely internal.

---

## 4. Component rules

### shadcn-vue (`src/components/ui/`)

- These files are CLI-generated. Don't hand-edit unless you've decided to
  fork. To upgrade: `npx shadcn-vue@latest add <name> --overwrite`.
- Prettier ignores this folder for the same reason (`/.prettierignore`).

### Project components (`src/components/`)

- Wrappers around shadcn primitives that add project-specific behaviour
  (e.g. `PasswordInput.vue` adds an eye-toggle to `Input`).
- Keep these small and composition-friendly. They should accept `v-model`
  and forward vee-validate `v-bind="...Attrs"`.

### Layouts (`src/layouts/`)

- A layout's template is shared chrome plus `<router-view />`. Nothing else.
- View files **do not** repeat what the layout already renders. Auth views
  start at `<CardHeader>` because `AuthLayout` owns the outer wrapper.
- The global `Navbar` lives in `App.vue` and renders above every layout.

---

## 5. State & data

- Pinia stores for cross-component state. Components consume them via
  `useXStore()` — never read `localStorage` directly outside of `authStore`.
- All HTTP calls go through `src/services/<domain>Service.ts`. Components
  and views do not import `axios` directly.
- The shared `api.ts` axios instance:
  - Request interceptor attaches `Authorization: Bearer …` from
    `localStorage`.
  - Response interceptor auto-redirects on 401 **except** for `/auth/*`
    requests, which let the calling view show its own error.
- `authStore.init()` runs once in `main.ts`. It decodes the JWT `exp` claim
  and clears the session if expired — never trust a stored token blindly.
- Refresh token: stored in `localStorage` but not yet exchanged. When
  `/auth/refresh` ships, wire it into both `api.ts` (retry 401 once) and
  `authStore.init()` (silent re-auth on load).

---

## 6. Forms

Every form uses vee-validate + zod. Pattern:

```ts
const schema = toTypedSchema(z.object({ ... }))
const { defineField, handleSubmit, errors, isSubmitting } = useForm({
  validationSchema: schema,
  initialValues: { ... },
})
const [email, emailAttrs] = defineField("email")
const onSubmit = handleSubmit(async (values) => { ... })
```

Rules:

1. **Schema is the source of truth.** No `required` / `minlength` HTML
   attributes alongside zod — pick one.
2. **`novalidate` on every `<form>`** so the native browser UI never shows.
3. **Never reset the form on server error.** vee-validate preserves values
   across re-renders by default; do not call `resetForm()` in a catch.
   Server errors (bad credentials, taken email) go to a separate
   `serverError` ref shown in an `<Alert variant="destructive">` above the
   form — they are not field errors.
4. **Password fields use `PasswordInput`**, not `Input type="password"`.
   Don't pass a `type` prop — the component manages it for the eye toggle.
5. **Cross-field rules** live in `.refine()` on the schema, not in the
   submit handler.

See [`src/views/(auth)/LoginView.vue`](<src/views/(auth)/LoginView.vue>) and
[`src/views/(auth)/RegisterView.vue`](<src/views/(auth)/RegisterView.vue>)
for canonical examples.

---

## 7. Styling

- Tailwind utility classes only. No `<style scoped>` blocks in views or
  project components. (`src/components/ui/` is exempt — generated.)
- Use design tokens, not hex values: `text-foreground`,
  `text-muted-foreground`, `text-primary`, `text-destructive`,
  `border-border`, `bg-background`. Dark mode (if added) hinges on this.
- Mobile-first responsive prefixes (default → `md:` → `lg:`).
- Class order is automatically sorted by `prettier-plugin-tailwindcss`.
  Don't fight it.

---

## 8. Imports & formatting

- All in-project imports use `@/`. Never `../../foo`.
- Imports are auto-grouped by `@ianvs/prettier-plugin-sort-imports`
  (builtins → third-party → `@/layouts` → `@/views` → `@/components` → …).
  See doc 26 for the full order.
- A husky `pre-commit` hook runs `lint-staged` → `prettier --write` on staged
  frontend files. You normally don't need to run `npm run format` manually —
  but it's fine to do so to format the whole tree at once.
- Don't bypass the hook with `git commit --no-verify` to "save time" on a
  formatting error — fix the code instead. If the hook itself is broken,
  fix the hook.
- First-time clone setup: `npm install` at the repo root (registers the
  hook), then `npm install` in `frontend/`. See doc 26 → Pre-commit hook.

---

## 9. Commands

```bash
npm run dev           # Vite dev server
npm run build         # vue-tsc -b && vite build
npm run preview       # serve dist/
npm run format        # write Prettier changes
npm run format:check  # verify only — non-zero exit if anything would change
```

For UI changes, start `npm run dev`, exercise the feature in a browser,
and confirm at least one golden-path + one edge case. Type checking proves
correctness, not feature-correctness.

---

## 10. Common pitfalls

- **Don't bypass `services/`** — calling `axios` directly in a component
  skips the 401 handler and the auth header.
- **Don't store context-of-render data in localStorage.** Anything you put
  there must survive a page reload and be safe to read from any tab.
- **Don't add a route without a layout.** A new top-level route needs
  either an existing layout child or a brand-new layout + group.
- **Don't import `Card` in an auth view** — `AuthLayout` already wraps the
  outlet in one. You'll get a card-in-a-card.
- **Don't pass `type="password"` to `PasswordInput`** — it manages its own
  type for the eye toggle.

---

## 11. Priority order

1. correctness
2. accessibility (forms, focus, aria)
3. consistency with the existing pattern
4. performance
5. developer ergonomics
