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

import { getBillingModeLabelKey } from '../lib/model-helpers'
import {
  formatVideoGroupPrice,
  formatVideoPrice,
  getConfiguredVideoResolutions,
} from '../lib/price'
import type { PricingModel } from '../types'

function videoModel(videoPrices: PricingModel['video_prices']): PricingModel {
  return {
    id: 1,
    model_name: 'doubao-seedance-2.0-mini',
    quota_type: 1,
    model_ratio: 37.5,
    completion_ratio: 1,
    billing_mode: 'video',
    enable_groups: ['default'],
    video_prices: videoPrices,
  }
}

describe('model square video display contract', () => {
  test('shows per-second billing instead of token-based when API returns video mode', () => {
    assert.equal(
      getBillingModeLabelKey(videoModel({ '480p': 0.5 })),
      'Per second'
    )
    assert.equal(
      getBillingModeLabelKey({
        quota_type: 0,
        billing_mode: undefined,
        billing_expr: undefined,
      }),
      'Token-based'
    )
  })

  test('lists only configured video tiers for public pricing details', () => {
    const model = videoModel({ '480p': 0.5, '720p': 0.8 })
    assert.deepEqual(getConfiguredVideoResolutions(model), ['480p', '720p'])
    assert.equal(formatVideoPrice(model, '1080p'), '-')
    assert.notEqual(formatVideoPrice(model, '480p'), '-')
  })

  test('applies group ratio to configured video prices only', () => {
    const model = videoModel({ '480p': 0.5, '720p': 0.8 })
    const defaultPrice = formatVideoGroupPrice(
      model,
      '720p',
      'default',
      false,
      1,
      1,
      { default: 1 }
    )
    const vipPrice = formatVideoGroupPrice(model, '720p', 'vip', false, 1, 1, {
      default: 1,
      vip: 2,
    })
    assert.notEqual(defaultPrice, '-')
    assert.notEqual(vipPrice, '-')
    assert.notEqual(defaultPrice, vipPrice)
    assert.equal(
      formatVideoGroupPrice(model, '1080p', 'vip', false, 1, 1, { vip: 2 }),
      '-'
    )
  })
})
