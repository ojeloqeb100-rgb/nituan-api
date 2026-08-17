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

import {
  isLandingHashHref,
  isOfficialDocsHref,
  LANDING_ROUTES,
  LANDING_SECTION_IDS,
} from '../lib/landing'

describe('landing page canvas anchors', () => {
  test('keeps in-page section ids as hash targets, not a replacement header', () => {
    for (const id of Object.values(LANDING_SECTION_IDS)) {
      assert.equal(isLandingHashHref(`#${id}`), true)
      assert.equal(isOfficialDocsHref(`#${id}`), false)
    }
  })

  test('keeps page CTAs on existing app routes', () => {
    assert.equal(LANDING_ROUTES.signUp, '/sign-up')
    assert.equal(LANDING_ROUTES.pricing, '/pricing')
    assert.equal(LANDING_ROUTES.dashboard, '/dashboard')
    assert.equal(isOfficialDocsHref(LANDING_ROUTES.signUp), false)
    assert.equal(isOfficialDocsHref(LANDING_ROUTES.pricing), false)
  })
})
