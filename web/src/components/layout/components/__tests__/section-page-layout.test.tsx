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

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { SectionPageLayout } = await import('../section-page-layout')

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

type RenderedLayout = {
  container: HTMLDivElement
  root: ReturnType<typeof createRoot>
}

async function renderLayout(children: ReactNode): Promise<RenderedLayout> {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(<SectionPageLayout>{children}</SectionPageLayout>)
  })

  return { container, root }
}

async function unmountLayout(rendered: RenderedLayout) {
  await act(async () => rendered.root.unmount())
  rendered.container.remove()
}

describe('section page layout description slot', () => {
  after(() => {
    domWindow.close()
  })

  test('keeps the title in the heading and places description below it', async () => {
    const rendered = await renderLayout([
      <SectionPageLayout.Title key='title'>API Keys</SectionPageLayout.Title>,
      <SectionPageLayout.Actions key='actions'>
        <button type='button'>Add</button>
      </SectionPageLayout.Actions>,
      <SectionPageLayout.Description key='description'>
        <span>https://api.example.com</span>
      </SectionPageLayout.Description>,
      <SectionPageLayout.Content key='content'>
        <div>table</div>
      </SectionPageLayout.Content>,
    ])

    const heading = rendered.container.querySelector('h2')
    const description = rendered.container.querySelector(
      '[data-slot="section-description"]'
    )
    assert.ok(heading)
    assert.ok(description)
    assert.equal(heading.textContent, 'API Keys')
    assert.equal(heading.contains(description), false)
    assert.equal(description.textContent, 'https://api.example.com')
    assert.equal(
      Boolean(
        heading.compareDocumentPosition(description) &
          Node.DOCUMENT_POSITION_FOLLOWING
      ),
      true
    )
    assert.equal(rendered.container.textContent?.includes('table'), true)
    assert.equal(description.textContent?.includes('table'), false)

    await unmountLayout(rendered)
  })

  test('does not render a description slot when other pages omit it', async () => {
    const rendered = await renderLayout([
      <SectionPageLayout.Title key='title'>Users</SectionPageLayout.Title>,
      <SectionPageLayout.Content key='content'>
        <div>users table</div>
      </SectionPageLayout.Content>,
    ])

    assert.equal(
      rendered.container.querySelector('[data-slot="section-description"]'),
      null
    )
    assert.equal(rendered.container.querySelector('h2')?.textContent, 'Users')

    await unmountLayout(rendered)
  })
})
