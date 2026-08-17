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
  displayTaskFailReason,
  isExternalVideoUrl,
  localVideoContentPath,
  resolveTaskVideoHref,
  shouldShowTaskVideoLink,
} from '../task-video-url'

describe('task video log details', () => {
  test('successful text-to-video tasks expose a copyable local content URL', () => {
    const taskId = 'task_1JgtEpdmvQDPIbkBLlMx1il9MHj1PsPK'
    assert.equal(shouldShowTaskVideoLink({
      status: 'SUCCESS',
      action: 'textGenerate',
      platform: 'openai',
      task_id: taskId,
    }), true)
    assert.equal(
      resolveTaskVideoHref({
        status: 'SUCCESS',
        action: 'textGenerate',
        platform: 'openai',
        task_id: taskId,
        result_url: `https://localhost:3000/v1/videos/${taskId}/content?expires=4102444800&sign=abc`,
        fail_reason: 'https://cdn.example/signed/v.mp4',
      }, 'http://localhost:5173'),
      `http://localhost:5173/v1/videos/${taskId}/content?expires=4102444800&sign=abc`
    )
    assert.equal(
      localVideoContentPath(taskId),
      `/v1/videos/${taskId}/content`
    )
    assert.equal(
      isExternalVideoUrl('https://cdn.example/signed/v.mp4'),
      true
    )
    assert.equal(
      isExternalVideoUrl(`/v1/videos/${taskId}/content?expires=1&sign=abc`),
      false
    )
  })

  test('in-progress and failed tasks do not show a video link', () => {
    assert.equal(shouldShowTaskVideoLink({
      status: 'IN_PROGRESS',
      action: 'textGenerate',
      platform: 'openai',
      task_id: 'task_public',
    }), false)
    assert.equal(shouldShowTaskVideoLink({
      status: 'FAILURE',
      action: 'textGenerate',
      platform: 'openai',
      task_id: 'task_public',
    }), false)
    assert.equal(
      displayTaskFailReason('upstream rejected the prompt'),
      'upstream rejected the prompt'
    )
    assert.equal(
      displayTaskFailReason('https://cdn.example/signed/v.mp4'),
      ''
    )
  })

  test('historical successful video tasks still resolve a local link from task id', () => {
    assert.equal(shouldShowTaskVideoLink({
      status: 'SUCCESS',
      action: 'generate',
      task_id: 'task_old',
    }), true)
    assert.equal(
      resolveTaskVideoHref({
        status: 'SUCCESS',
        action: 'generate',
        task_id: 'task_old',
        fail_reason: '/v1/videos/task_old/content',
      }, 'http://localhost:5173'),
      'http://localhost:5173/v1/videos/task_old/content'
    )
    assert.equal(localVideoContentPath('task_old'), '/v1/videos/task_old/content')
    assert.equal(displayTaskFailReason('/v1/videos/task_old/content'), '')
  })
})
