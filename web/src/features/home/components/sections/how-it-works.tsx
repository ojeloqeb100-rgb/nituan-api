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

import { LANDING_LAYOUT, LANDING_SECTION_IDS } from '../../lib/landing'

const STEPS = [
  {
    num: '1',
    titleKey: 'Create an account',
    descKey: 'Sign up and get your API key in the console.',
  },
  {
    num: '2',
    titleKey: 'Change the address',
    descKey: 'Point your tool or SDK base_url to this site.',
  },
  {
    num: '3',
    titleKey: 'Call models as usual',
    descKey: 'Keep using OpenAI, Anthropic, or your current client.',
  },
] as const

export function HowItWorks() {
  const { t } = useTranslation()

  return (
    <section
      id={LANDING_SECTION_IDS.howTo}
      className={LANDING_LAYOUT.section}
    >
      <div className={LANDING_LAYOUT.sectionInner}>
        <div className='mb-10 max-w-xl'>
          <p className='mb-2 text-xs font-medium tracking-widest text-[color:var(--nt-ink-soft)] uppercase'>
            {t('How to use')}
          </p>
          <h2 className='text-2xl font-bold tracking-tight md:text-3xl'>
            {t('Three steps to connect')}
          </h2>
        </div>
        <div className='grid gap-4 md:grid-cols-3'>
          {STEPS.map((step) => (
            <div key={step.num} className='nt-card p-6'>
              <span className='nt-btn-primary size-8 text-sm font-bold'>
                {step.num}
              </span>
              <h3 className='mt-4 text-base font-semibold'>
                {t(step.titleKey)}
              </h3>
              <p className='mt-2 text-sm leading-relaxed text-[color:var(--nt-ink-soft)]'>
                {t(step.descKey)}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
