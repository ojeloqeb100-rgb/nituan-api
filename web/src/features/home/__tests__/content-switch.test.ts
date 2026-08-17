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
import { describe, test } from 'node:test'

import { resolveHomeSurface } from '../lib/home-surface'

describe('home page content switch', () => {
  test('shows the default landing when HomePageContent is empty', () => {
    assert.equal(
      resolveHomeSurface({ isLoaded: true, content: '', isUrl: false }),
      'default'
    )
  })

  test('keeps custom URL, HTML, and markdown surfaces', () => {
    assert.equal(
      resolveHomeSurface({
        isLoaded: true,
        content: 'https://example.com/home',
        isUrl: true,
      }),
      'custom-url'
    )
    assert.equal(
      resolveHomeSurface({
        isLoaded: true,
        content: '<div>Custom</div>',
        isUrl: false,
      }),
      'custom-html'
    )
    assert.equal(
      resolveHomeSurface({
        isLoaded: true,
        content: '# Custom markdown',
        isUrl: false,
      }),
      'custom-markdown'
    )
  })

  test('stays on the loading surface until content is fetched', () => {
    assert.equal(
      resolveHomeSurface({ isLoaded: false, content: '', isUrl: false }),
      'loading'
    )
  })
})
