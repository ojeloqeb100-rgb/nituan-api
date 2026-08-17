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
import { LANDING_LAYOUT, LANDING_ROUTES } from '../../lib/landing'

interface CTAProps {
  isAuthenticated?: boolean
}

export function CTA(props: CTAProps) {
  const { t } = useTranslation()

  if (props.isAuthenticated) {
    return null
  }

  return (
    <section className={LANDING_LAYOUT.section}>
      <div className={`${LANDING_LAYOUT.sectionInner} nt-card px-6 py-12 text-center md:px-10`}>
        <h2 className='text-2xl font-bold tracking-tight md:text-3xl'>
          {t('Ready to connect your models?')}
        </h2>
        <p className='mx-auto mt-3 max-w-md text-sm text-[color:var(--nt-ink-soft)] md:text-base'>
          {t('Create an account, copy your key, and change one line.')}
        </p>
        <div className='mt-8 flex flex-wrap items-center justify-center gap-3'>
          <LandingLink
            href={LANDING_ROUTES.signUp}
            className='nt-btn-primary h-12 px-6 text-sm font-semibold'
          >
            {t('Get a free key →')}
          </LandingLink>
          <LandingLink
            href={LANDING_ROUTES.pricing}
            className='nt-btn-ghost h-12 px-5 text-sm font-medium'
          >
            {t('View Pricing')}
          </LandingLink>
        </div>
      </div>
    </section>
  )
}
