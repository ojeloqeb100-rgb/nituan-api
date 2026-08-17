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

import { LANDING_LAYOUT } from '../../lib/landing'

const FEATURES = [
  {
    titleKey: 'Official-ratio billing',
    descKey: 'Charges follow the public model price list.',
  },
  {
    titleKey: 'Balance keeps working',
    descKey: 'Unused balance does not reset or expire.',
  },
  {
    titleKey: 'Request-level records',
    descKey: 'Every call is logged so you can audit usage.',
  },
  {
    titleKey: 'Works with your stack',
    descKey: 'OpenAI SDK, Claude Code, Cherry Studio, and more.',
  },
] as const

export function Features() {
  const { t } = useTranslation()

  return (
    <section className={LANDING_LAYOUT.section}>
      <div className={LANDING_LAYOUT.sectionInner}>
        <div className='mb-10 max-w-xl'>
          <h2 className='text-2xl font-bold tracking-tight md:text-3xl'>
            {t('Why people stay')}
          </h2>
          <p className='mt-3 text-sm leading-relaxed text-[color:var(--nt-ink-soft)] md:text-base'>
            {t('Clear pricing and a key that just works.')}
          </p>
        </div>
        <div className='grid gap-4 md:grid-cols-2'>
          {FEATURES.map((feature) => (
            <div key={feature.titleKey} className='nt-card p-6'>
              <h3 className='text-base font-semibold'>{t(feature.titleKey)}</h3>
              <p className='mt-2 text-sm leading-relaxed text-[color:var(--nt-ink-soft)]'>
                {t(feature.descKey)}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
