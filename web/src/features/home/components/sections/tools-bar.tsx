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

import { LANDING_LAYOUT, LANDING_TOOLS } from '../../lib/landing'

export function ToolsBar() {
  const { t } = useTranslation()

  return (
    <section className='px-4 pb-6 md:px-6'>
      <div className={LANDING_LAYOUT.sectionInner}>
        <div className={LANDING_LAYOUT.toolsRow}>
          <span className='nt-btn-primary px-4 py-2 text-xs font-medium'>
            <span className='nt-dot bg-white/80' />
            {t('Your tools work — just change the address')}
          </span>
          {LANDING_TOOLS.map((tool) => (
            <span
              key={tool}
              className='nt-pill px-4 py-2 text-xs font-medium text-[color:var(--nt-ink-soft)]'
            >
              {tool}
            </span>
          ))}
        </div>
      </div>
    </section>
  )
}
