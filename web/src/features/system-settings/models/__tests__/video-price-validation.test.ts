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
  VIDEO_TIER_VALIDATION_MESSAGE,
  validateVideoTierPrices,
} from '../model-pricing-core'

describe('video tier price validation', () => {
  test('allows saving when only 480P and 720P are priced', () => {
    const result = validateVideoTierPrices({
      video480Price: '0.5',
      video720Price: '0.8',
      video1080Price: '',
    })
    assert.deepEqual(result, { ok: true })
  })

  test('rejects an empty form with no priced tier', () => {
    const result = validateVideoTierPrices({
      video480Price: '',
      video720Price: '   ',
      video1080Price: undefined,
    })
    assert.deepEqual(result, {
      ok: false,
      message: VIDEO_TIER_VALIDATION_MESSAGE,
    })
  })

  test('rejects zero, negative, NaN, and Infinity values', () => {
    for (const video1080Price of ['0', '-1', 'NaN', 'Infinity']) {
      const result = validateVideoTierPrices({
        video480Price: '0.5',
        video720Price: '0.8',
        video1080Price,
      })
      assert.deepEqual(result, {
        ok: false,
        message: VIDEO_TIER_VALIDATION_MESSAGE,
      })
    }
  })
})
