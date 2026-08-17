/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import assert from 'node:assert/strict'
import { after, describe, test } from 'node:test'

import { Window } from 'happy-dom'
import type { ReactNode } from 'react'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLButtonElement',
  'HTMLAnchorElement',
  'HTMLImageElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
  'ResizeObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const noopScrollTo = () => {}
Object.defineProperty(globalThis, 'scrollTo', {
  configurable: true,
  value: noopScrollTo,
})
Object.defineProperty(domWindow, 'scrollTo', {
  configurable: true,
  value: noopScrollTo,
})

const { act, createElement } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} = await import('@tanstack/react-router')
const { LandingPage } = await import('../components/landing-page')
const {
  LANDING_LAYOUT,
  LANDING_LOGO_SRC,
  LANDING_ROUTES,
  LANDING_SECTION_IDS,
} = await import('../lib/landing')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Ni Tuan AI': 'Ni Tuan AI',
        'How to use': 'How to use',
        'Generation samples': 'Generation samples',
        'Who uses it': 'Who uses it',
        FAQ: 'FAQ',
        'Go to Dashboard': 'Go to Dashboard',
        'One gateway, many models': 'One gateway, many models',
        'One key, every model you want': 'One key, every model you want',
        'Use Claude, OpenAI, and more with your existing SDK — just change the base_url.':
          'Use Claude, OpenAI, and more with your existing SDK — just change the base_url.',
        'Get a free key →': 'Get a free key →',
        'See how to connect': 'See how to connect',
        '1:1 top-up': '1:1 top-up',
        'Balance never resets or expires': 'Balance never resets or expires',
        'Change one line to connect': 'Change one line to connect',
        'Balance never resets': 'Balance never resets',
        'Per-request records': 'Per-request records',
        'Your tools work — just change the address':
          'Your tools work — just change the address',
        'Most of what you want is here': 'Most of what you want is here',
        'Billed at official price ratios, with a public price list.':
          'Billed at official price ratios, with a public price list.',
        'Browse the model list': 'Browse the model list',
        'Three steps to connect': 'Three steps to connect',
        'Create an account': 'Create an account',
        'Sign up and get your API key in the console.':
          'Sign up and get your API key in the console.',
        'Change the address': 'Change the address',
        'Point your tool or SDK base_url to this site.':
          'Point your tool or SDK base_url to this site.',
        'Call models as usual': 'Call models as usual',
        'Keep using OpenAI, Anthropic, or your current client.':
          'Keep using OpenAI, Anthropic, or your current client.',
        'Example calls you can run after changing base_url.':
          'Example calls you can run after changing base_url.',
        'Chat completion': 'Chat completion',
        'Claude messages': 'Claude messages',
        'Image generation': 'Image generation',
        'If the tool can set an API address, it can connect here.':
          'If the tool can set an API address, it can connect here.',
        'Why people stay': 'Why people stay',
        'Clear pricing and a key that just works.':
          'Clear pricing and a key that just works.',
        'Official-ratio billing': 'Official-ratio billing',
        'Charges follow the public model price list.':
          'Charges follow the public model price list.',
        'Balance keeps working': 'Balance keeps working',
        'Unused balance does not reset or expire.':
          'Unused balance does not reset or expire.',
        'Request-level records': 'Request-level records',
        'Every call is logged so you can audit usage.':
          'Every call is logged so you can audit usage.',
        'Works with your stack': 'Works with your stack',
        'OpenAI SDK, Claude Code, Cherry Studio, and more.':
          'OpenAI SDK, Claude Code, Cherry Studio, and more.',
        'Common questions': 'Common questions',
        'How do I get a key?': 'How do I get a key?',
        'Create a free account, open the console, and generate an API key.':
          'Create a free account, open the console, and generate an API key.',
        'How do I switch from OpenAI?': 'How do I switch from OpenAI?',
        'Keep your SDK or client. Change base_url to this site and use your new key.':
          'Keep your SDK or client. Change base_url to this site and use your new key.',
        'Does unused balance expire?': 'Does unused balance expire?',
        'No. Your balance stays until you use it.':
          'No. Your balance stays until you use it.',
        'How is usage priced?': 'How is usage priced?',
        'Models are billed using official price ratios. See the public model list.':
          'Models are billed using official price ratios. See the public model list.',
        'Which tools are supported?': 'Which tools are supported?',
        'Any client that lets you set an OpenAI- or Anthropic-compatible API address.':
          'Any client that lets you set an OpenAI- or Anthropic-compatible API address.',
        'Ready to connect your models?': 'Ready to connect your models?',
        'Create an account, copy your key, and change one line.':
          'Create an account, copy your key, and change one line.',
        'View Pricing': 'View Pricing',
      },
    },
  },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

function createLandingRouter(isAuthenticated: boolean) {
  const rootRoute = createRootRoute({
    component: () =>
      createElement(
        I18nextProvider,
        { i18n },
        createElement(LandingPage, { isAuthenticated })
      ),
  })
  const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
  })
  const signUpRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/sign-up',
  })
  const pricingRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/pricing',
  })
  const dashboardRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/dashboard',
  })

  return createRouter({
    routeTree: rootRoute.addChildren([
      indexRoute,
      signUpRoute,
      pricingRoute,
      dashboardRoute,
    ]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
}

async function renderLanding(isAuthenticated = false) {
  const host = document.createElement('div')
  document.body.appendChild(host)
  const root = createRoot(host)
  const router = createLandingRouter(isAuthenticated)
  await act(async () => {
    root.render(createElement(RouterProvider, { router }) as ReactNode)
  })
  return { host, root }
}

let rendered: {
  host: HTMLDivElement
  root: ReturnType<typeof createRoot>
} | null = null

after(() => {
  if (rendered) {
    rendered.root.unmount()
    rendered.host.remove()
  }
})

describe('landing canvas contract', () => {
  test('uses a clipped warm shell with a two-column hero and wrapping tool row', () => {
    assert.match(LANDING_LAYOUT.shell, /overflow-x-clip/)
    assert.match(LANDING_LAYOUT.shell, /nt-landing/)
    assert.match(LANDING_LAYOUT.heroGrid, /grid-cols-1/)
    assert.match(LANDING_LAYOUT.heroGrid, /lg:grid-cols-12/)
    assert.match(LANDING_LAYOUT.toolsRow, /flex-wrap/)
    assert.match(LANDING_LAYOUT.section, /scroll-mt-24/)
  })

  test('shows the square mascot on a rounded card instead of covering it into a circle', () => {
    assert.doesNotMatch(LANDING_LAYOUT.heroArt, /rounded-full/)
    assert.match(LANDING_LAYOUT.heroArt, /rounded-\[1\.75rem\]/)
    assert.doesNotMatch(LANDING_LAYOUT.heroImage, /object-cover/)
    assert.match(LANDING_LAYOUT.heroImage, /object-contain/)
    assert.match(LANDING_LAYOUT.heroGlow, /rounded-full/)
  })

  test('renders the mascot, headline, and in-page sections without a local header', async () => {
    rendered = await renderLanding(false)
    const root = rendered.host

    assert.equal(root.querySelector('[data-landing="ni-tuan"]') !== null, true)
    assert.equal(root.querySelector('header'), null)
    const mascot = root.querySelector(`img[src="${LANDING_LOGO_SRC}"]`)
    assert.equal(mascot !== null, true)
    assert.match(mascot?.className ?? '', /object-contain/)
    assert.doesNotMatch(mascot?.className ?? '', /object-cover/)
    assert.doesNotMatch(mascot?.parentElement?.className ?? '', /rounded-full/)
    assert.match(root.textContent ?? '', /One key, every model you want/)
    assert.match(root.textContent ?? '', /Your tools work — just change the address/)

    for (const id of Object.values(LANDING_SECTION_IDS)) {
      const section = root.querySelector(`#${id}`)
      assert.equal(section !== null, true)
      assert.match(section?.className ?? '', /scroll-mt-24/)
    }

    const signUp = root.querySelector(`a[href="${LANDING_ROUTES.signUp}"]`)
    const howTo = root.querySelector(`a[href="#${LANDING_SECTION_IDS.howTo}"]`)
    const docs = root.querySelector('a[href*="docs.newapi.pro"]')

    assert.equal(signUp !== null, true)
    assert.equal(howTo !== null, true)
    assert.equal(docs, null)
  })

  test('points the hero CTA at the dashboard after sign-in', async () => {
    if (rendered) {
      rendered.root.unmount()
      rendered.host.remove()
    }
    rendered = await renderLanding(true)
    const root = rendered.host

    assert.equal(root.querySelector('header'), null)
    assert.equal(
      root.querySelector(`a[href="${LANDING_ROUTES.dashboard}"]`) !== null,
      true
    )
    assert.equal(root.querySelector(`a[href="${LANDING_ROUTES.signUp}"]`), null)
  })
})
