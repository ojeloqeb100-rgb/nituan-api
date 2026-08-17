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
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { useApiInfo } from '@/features/dashboard/hooks/use-status-data'
import { getBgColorClass } from '@/lib/colors'
import { cn } from '@/lib/utils'

export function ApiBaseUrlBar() {
  const { t } = useTranslation()
  const { items } = useApiInfo()
  const visibleItems = items.filter(
    (item) => typeof item.url === 'string' && item.url.trim() !== ''
  )

  if (visibleItems.length === 0) {
    return null
  }

  return (
    <div
      role='region'
      aria-label={t('API Addresses')}
      data-slot='api-base-url-highlight'
      className='bg-muted/50 overflow-hidden rounded-lg border'
    >
      <p className='text-muted-foreground border-b px-3 py-1.5 text-xs font-medium'>
        {t('API Addresses')}
      </p>
      <ul className='divide-border divide-y'>
        {visibleItems.map((item) => (
          <li
            key={`${item.url}::${item.route}`}
            className='flex min-w-0 items-center gap-2 px-3 py-2'
          >
            <span
              className={cn(
                'inline-block size-2 shrink-0 rounded-full',
                getBgColorClass(item.color)
              )}
            />
            {item.route ? (
              <span className='text-muted-foreground max-w-32 shrink-0 truncate text-xs'>
                {item.route}
              </span>
            ) : null}
            <span className='min-w-0 truncate font-mono text-xs font-medium'>
              {item.url}
            </span>
            <CopyButton
              value={item.url}
              variant='ghost'
              size='sm'
              className='size-6 p-0'
              iconClassName='size-3'
              tooltip={t('Copy URL')}
              aria-label={t('Copy URL')}
            />
          </li>
        ))}
      </ul>
    </div>
  )
}
