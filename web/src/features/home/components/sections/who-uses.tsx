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

import {
  LANDING_LAYOUT,
  LANDING_SECTION_IDS,
  LANDING_TOOLS,
} from '../../lib/landing'

export function WhoUses() {
  const { t } = useTranslation()

  return (
    <section
      id={LANDING_SECTION_IDS.whoUses}
      className={LANDING_LAYOUT.section}
    >
      <div className={LANDING_LAYOUT.sectionInner}>
        <div className='mb-10 max-w-xl'>
          <h2 className='text-2xl font-bold tracking-tight md:text-3xl'>
            {t('Who uses it')}
          </h2>
          <p className='mt-3 text-sm leading-relaxed text-[color:var(--nt-ink-soft)] md:text-base'>
            {t('If the tool can set an API address, it can connect here.')}
          </p>
        </div>
        <div className='grid grid-cols-2 gap-3 md:grid-cols-4'>
          {LANDING_TOOLS.map((tool) => (
            <div
              key={tool}
              className='nt-card flex min-h-20 items-center justify-center px-4 text-center text-sm font-medium'
            >
              {tool}
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
