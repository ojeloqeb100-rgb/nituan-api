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

import { ENDPOINT_TYPES } from '../../pricing/constants'
import { filterByEndpointType } from '../../pricing/lib/filters'
import type { PricingModel } from '../../pricing/types'
import { ENDPOINT_TEMPLATES } from '../constants'

function loadEndpointTemplate(templateKey: string): Record<string, unknown> {
  const template = ENDPOINT_TEMPLATES[templateKey]
  assert.ok(template, `missing endpoint template: ${templateKey}`)
  // Mirror the model drawer: selecting a template writes this JSON into the form.
  const templateJson = JSON.stringify({ [templateKey]: template }, null, 2)
  return JSON.parse(templateJson) as Record<string, unknown>
}

function plazaModelFromTemplate(
  templateKey: string
): Pick<PricingModel, 'supported_endpoint_types'> {
  return {
    supported_endpoint_types: Object.keys(loadEndpointTemplate(templateKey)),
  }
}

describe('video endpoint template plaza contract', () => {
  test('loading the openai-video template is recognized as Video by plaza filters', () => {
    const loaded = loadEndpointTemplate('openai-video')

    assert.deepEqual(loaded['openai-video'], {
      path: '/v1/videos',
      method: 'POST',
    })
    assert.equal(ENDPOINT_TYPES.OPENAI_VIDEO, 'openai-video')

    const matched = filterByEndpointType(
      [plazaModelFromTemplate('openai-video') as PricingModel],
      ENDPOINT_TYPES.OPENAI_VIDEO
    )
    assert.equal(matched.length, 1)
  })

  test('plaza Video filter ignores video or videos keys that are not openai-video', () => {
    const decoys = [
      { supported_endpoint_types: ['video'] },
      { supported_endpoint_types: ['videos'] },
      { supported_endpoint_types: ['image-generation'] },
    ] as PricingModel[]

    assert.equal(
      filterByEndpointType(decoys, ENDPOINT_TYPES.OPENAI_VIDEO).length,
      0
    )
  })
})
