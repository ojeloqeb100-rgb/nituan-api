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
import { after, afterEach, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLButtonElement',
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

const copiedTexts: string[] = []
Object.defineProperty(domWindow.navigator, 'clipboard', {
  configurable: true,
  value: {
    writeText: async (text: string) => {
      copiedTexts.push(text)
    },
  },
})

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { TooltipProvider } = await import('@/components/ui/tooltip')
const { ApiBaseUrlBar } = await import('../api-base-url-bar')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'API Addresses': 'API Addresses',
        'Copy URL': 'Copy URL',
        Copied: 'Copied',
        'Copy to clipboard': 'Copy to clipboard',
        'Copied!': 'Copied!',
      },
    },
  },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

type StatusFixture = {
  api_info_enabled?: boolean
  api_info?: Array<{
    url: string
    route: string
    description: string
    color: string
  }>
}

type RenderedBar = {
  container: HTMLDivElement
  root: ReturnType<typeof createRoot>
}

async function renderBar(status?: StatusFixture): Promise<RenderedBar> {
  domWindow.localStorage.clear()
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  if (status) {
    queryClient.setQueryData(['status'], status, {
      updatedAt: Date.now() + 60_000,
    })
  } else {
    queryClient.setQueryDefaults(['status'], { enabled: false })
  }

  await act(async () => {
    root.render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <TooltipProvider>
            <ApiBaseUrlBar />
          </TooltipProvider>
        </I18nextProvider>
      </QueryClientProvider>
    )
  })

  return { container, root }
}

async function unmountBar(rendered: RenderedBar) {
  await act(async () => rendered.root.unmount())
  rendered.container.remove()
}

function apiAddressRegion(container: HTMLElement) {
  return container.querySelector('[role="region"][aria-label="API Addresses"]')
}

function highlightBlock(container: HTMLElement) {
  return container.querySelector('[data-slot="api-base-url-highlight"]')
}

function hasHighlightSurface(element: Element) {
  const classNames = [...element.classList]
  return (
    classNames.includes('border') &&
    classNames.some((name) => name.startsWith('rounded')) &&
    classNames.some((name) => name.startsWith('bg-muted'))
  )
}

function copyButtons(container: HTMLElement) {
  return [...container.querySelectorAll('button')].filter((button) => {
    const label = button.getAttribute('aria-label')
    return label === 'Copy URL' || label === 'Copied'
  })
}

function rowForUrl(container: HTMLElement, url: string) {
  return [...container.querySelectorAll('li')].find((row) =>
    row.textContent?.includes(url)
  )
}

describe('API keys base URL bar', () => {
  afterEach(() => {
    copiedTexts.length = 0
    domWindow.localStorage.clear()
  })

  after(() => {
    domWindow.close()
  })

  test('renders configured addresses and copies the matching URL', async () => {
    const rendered = await renderBar({
      api_info_enabled: true,
      api_info: [
        {
          url: 'https://api.example.com',
          route: 'Official',
          description: 'Primary',
          color: 'blue',
        },
      ],
    })

    const region = apiAddressRegion(rendered.container)
    assert.ok(region)
    assert.equal(region.textContent?.includes('https://api.example.com'), true)
    assert.equal(region.textContent?.includes('Official'), true)
    assert.equal(rendered.container.querySelector('h2'), null)

    const buttons = copyButtons(rendered.container)
    assert.equal(buttons.length, 1)

    await act(async () => {
      buttons[0].click()
      await Promise.resolve()
    })
    assert.deepEqual(copiedTexts, ['https://api.example.com'])

    await unmountBar(rendered)
  })

  test('wraps configured addresses in a highlighted container', async () => {
    const rendered = await renderBar({
      api_info_enabled: true,
      api_info: [
        {
          url: 'https://api.example.com',
          route: 'Official',
          description: 'Primary',
          color: 'blue',
        },
      ],
    })

    const highlight = highlightBlock(rendered.container)
    assert.ok(highlight)
    assert.equal(hasHighlightSurface(highlight), true)
    assert.equal(highlight.textContent?.includes('https://api.example.com'), true)
    assert.equal(highlight.querySelectorAll('li').length, 1)

    await unmountBar(rendered)
  })

  test('hides the bar when API addresses are disabled even if items exist', async () => {
    const rendered = await renderBar({
      api_info_enabled: false,
      api_info: [
        {
          url: 'https://hidden.example.com',
          route: 'Hidden',
          description: '',
          color: 'blue',
        },
      ],
    })

    assert.equal(apiAddressRegion(rendered.container), null)
    assert.equal(highlightBlock(rendered.container), null)
    assert.equal(
      rendered.container.textContent?.includes('https://hidden.example.com'),
      false
    )
    assert.equal(copyButtons(rendered.container).length, 0)
    assert.equal(
      rendered.container.textContent?.includes(domWindow.location.origin),
      false
    )

    await unmountBar(rendered)
  })

  test('hides the bar when the enabled list is empty and does not fall back to the page origin', async () => {
    const rendered = await renderBar({
      api_info_enabled: true,
      api_info: [],
    })

    assert.equal(apiAddressRegion(rendered.container), null)
    assert.equal(highlightBlock(rendered.container), null)
    assert.equal(copyButtons(rendered.container).length, 0)
    assert.equal(
      rendered.container.textContent?.includes(domWindow.location.origin),
      false
    )

    await unmountBar(rendered)
  })

  test('lists every configured address with its own copy action', async () => {
    const firstUrl = 'https://primary.example.com'
    const secondUrl = 'https://backup.example.com'
    const rendered = await renderBar({
      api_info_enabled: true,
      api_info: [
        {
          url: firstUrl,
          route: 'Primary',
          description: '',
          color: 'blue',
        },
        {
          url: secondUrl,
          route: 'Backup',
          description: '',
          color: 'green',
        },
      ],
    })

    const region = apiAddressRegion(rendered.container)
    assert.ok(region)
    assert.equal(region.textContent?.includes(firstUrl), true)
    assert.equal(region.textContent?.includes(secondUrl), true)
    assert.equal(region.textContent?.includes('Primary'), true)
    assert.equal(region.textContent?.includes('Backup'), true)
    assert.equal(copyButtons(rendered.container).length, 2)

    const firstRow = rowForUrl(rendered.container, firstUrl)
    const secondRow = rowForUrl(rendered.container, secondUrl)
    assert.ok(firstRow)
    assert.ok(secondRow)

    const firstButton = firstRow.querySelector('button')
    const secondButton = secondRow.querySelector('button')
    assert.ok(firstButton)
    assert.ok(secondButton)

    await act(async () => {
      firstButton.click()
      await Promise.resolve()
    })
    await act(async () => {
      secondButton.click()
      await Promise.resolve()
    })
    assert.deepEqual(copiedTexts, [firstUrl, secondUrl])

    await unmountBar(rendered)
  })
})
