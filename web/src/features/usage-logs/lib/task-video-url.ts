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
import { TASK_ACTIONS, TASK_PLATFORMS, TASK_STATUS } from '../constants'

export function localVideoContentPath(taskId: string): string {
  return `/v1/videos/${taskId}/content`
}

export function isVideoTaskAction(action: string | undefined): boolean {
  return (
    action === TASK_ACTIONS.GENERATE ||
    action === TASK_ACTIONS.TEXT_GENERATE ||
    action === TASK_ACTIONS.FIRST_TAIL_GENERATE ||
    action === TASK_ACTIONS.REFERENCE_GENERATE ||
    action === TASK_ACTIONS.REMIX_GENERATE
  )
}

export function isExternalVideoUrl(value: string | undefined): boolean {
  const url = value?.trim() ?? ''
  if (!url.startsWith('http://') && !url.startsWith('https://')) {
    return false
  }
  return !isLocalVideoContentPath(url)
}

export function isLocalVideoContentPath(value: string | undefined): boolean {
  const url = value?.trim() ?? ''
  if (!url) {
    return false
  }
  try {
    const path =
      url.startsWith('http://') || url.startsWith('https://')
        ? new URL(url).pathname
        : url.split('?')[0]
    return /^\/v1\/videos\/[^/]+\/content$/.test(path)
  } catch {
    return false
  }
}

export function shouldShowTaskVideoLink(log: {
  status?: string
  action?: string
  platform?: string
  task_id?: string
  fail_reason?: string
  result_url?: string
}): boolean {
  if (
    log.status !== TASK_STATUS.SUCCESS ||
    log.platform === TASK_PLATFORMS.SUNO ||
    !log.task_id
  ) {
    return false
  }
  return (
    isVideoTaskAction(log.action) ||
    isLocalVideoContentPath(log.fail_reason) ||
    isLocalVideoContentPath(log.result_url)
  )
}

export function resolveTaskVideoHref(
  log: {
    status?: string
    action?: string
    platform?: string
    task_id?: string
    fail_reason?: string
    result_url?: string
  },
  origin?: string
): string | null {
  if (!shouldShowTaskVideoLink(log) || !log.task_id) {
    return null
  }
  const base =
    origin ??
    (typeof window !== 'undefined' ? window.location.origin : '')
  const raw = log.result_url?.trim()
  if (raw && isLocalVideoContentPath(raw)) {
    return rebaseVideoContentUrl(raw, base)
  }
  const fallback = localVideoContentPath(log.task_id)
  return base ? `${base}${fallback}` : fallback
}

function rebaseVideoContentUrl(raw: string, origin: string): string {
  if (raw.startsWith('http://') || raw.startsWith('https://')) {
    try {
      const parsed = new URL(raw)
      if (!origin) {
        return raw
      }
      return `${origin}${parsed.pathname}${parsed.search}`
    } catch {
      return raw
    }
  }
  const path = raw.startsWith('/') ? raw : `/${raw}`
  return origin ? `${origin}${path}` : path
}

export function displayTaskFailReason(failReason: string | undefined): string {
  if (
    !failReason ||
    isExternalVideoUrl(failReason) ||
    isLocalVideoContentPath(failReason)
  ) {
    return ''
  }
  return failReason
}
