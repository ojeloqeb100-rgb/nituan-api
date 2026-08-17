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

import { LandingLink } from '../landing-link'
import {
  LANDING_LAYOUT,
  LANDING_LOGO_SRC,
  LANDING_MODEL_FAMILIES,
  LANDING_ROUTES,
  LANDING_SECTION_IDS,
} from '../../lib/landing'

export function Models() {
  const { t } = useTranslation()

  return (
    <section
      id={LANDING_SECTION_IDS.models}
      className={LANDING_LAYOUT.section}
    >
      <div className={LANDING_LAYOUT.sectionInner}>
        <div className='flex flex-col justify-between gap-8 md:flex-row md:items-end'>
          <div className='max-w-xl'>
            <h2 className='text-2xl font-bold tracking-tight md:text-3xl'>
              {t('Most of what you want is here')}
            </h2>
            <p className='mt-3 text-sm leading-relaxed text-[color:var(--nt-ink-soft)] md:text-base'>
              {t('Billed at official price ratios, with a public price list.')}
            </p>
          </div>
          <LandingLink
            href={LANDING_ROUTES.pricing}
            className='nt-btn-ghost h-11 px-5 text-sm font-medium'
          >
            {t('Browse the model list')}
          </LandingLink>
        </div>
        <div className='mt-8 flex flex-wrap gap-2'>
          {LANDING_MODEL_FAMILIES.map((name) => (
            <span key={name} className='nt-pill px-4 py-2 text-sm font-medium'>
              {name}
            </span>
          ))}
        </div>
        <img
          src={LANDING_LOGO_SRC}
          alt=''
          className='mt-8 ml-auto hidden size-16 rounded-full object-cover opacity-90 md:block'
          aria-hidden='true'
        />
      </div>
    </section>
  )
}
