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
  LANDING_BRAND_KEY,
  LANDING_FLOAT_TAGS,
  LANDING_HIGHLIGHTS,
  LANDING_LAYOUT,
  LANDING_LOGO_SRC,
  LANDING_ROUTES,
  LANDING_SECTION_IDS,
} from '../../lib/landing'

interface HeroProps {
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const primaryHref = props.isAuthenticated
    ? LANDING_ROUTES.dashboard
    : LANDING_ROUTES.signUp
  const primaryLabel = props.isAuthenticated
    ? t('Go to Dashboard')
    : t('Get a free key →')

  return (
    <section className='relative z-10 px-0 pt-24 pb-10 md:pt-32 md:pb-14 lg:pt-36'>
      <div className={LANDING_LAYOUT.heroGrid}>
        <div className='flex flex-col items-start text-left lg:col-span-6'>
          <div className='nt-pill landing-animate-fade-up mb-5 px-3 py-1.5 text-[11px] font-medium text-[color:var(--nt-ink-soft)]'>
            <span className='nt-dot nt-dot-amber' />
            <span>{t('One gateway, many models')}</span>
          </div>

          <h1
            className='landing-animate-fade-up max-w-xl text-[clamp(2.1rem,4.6vw,3.35rem)] leading-[1.15] font-bold tracking-tight'
            style={{ animationDelay: '60ms' }}
          >
            {t('One key, every model you want')}
          </h1>
          <p
            className='landing-animate-fade-up mt-5 max-w-xl text-base leading-relaxed text-[color:var(--nt-ink-soft)] md:text-[15px]'
            style={{ animationDelay: '120ms' }}
          >
            {t(
              'Use Claude, OpenAI, and more with your existing SDK — just change the base_url.'
            )}
          </p>

          <div
            className='landing-animate-fade-up mt-8 flex flex-wrap items-center gap-3'
            style={{ animationDelay: '180ms' }}
          >
            <LandingLink
              href={primaryHref}
              className='nt-btn-primary h-12 px-6 text-sm font-semibold'
            >
              {primaryLabel}
            </LandingLink>
            <LandingLink
              href={`#${LANDING_SECTION_IDS.howTo}`}
              className='nt-btn-ghost h-12 px-5 text-sm font-medium'
            >
              {t('See how to connect')}
            </LandingLink>
          </div>

          <div
            className='landing-animate-fade-up mt-10 grid w-full max-w-xl grid-cols-1 gap-3 sm:grid-cols-3'
            style={{ animationDelay: '240ms' }}
          >
            {LANDING_HIGHLIGHTS.map((item) => (
              <div key={item.id} className='nt-pill px-4 py-3'>
                <div>
                  <p className='text-lg leading-none font-bold'>{item.value}</p>
                  <p className='mt-1 text-xs text-[color:var(--nt-ink-soft)]'>
                    {t(item.labelKey)}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div
          className='landing-animate-fade-up flex justify-center lg:col-span-6'
          style={{ animationDelay: '280ms' }}
        >
          <div className={LANDING_LAYOUT.heroVisual}>
            <div className={LANDING_LAYOUT.heroGlow} aria-hidden='true' />
            <div className={LANDING_LAYOUT.heroArt}>
              <img
                src={LANDING_LOGO_SRC}
                alt={t(LANDING_BRAND_KEY)}
                className={LANDING_LAYOUT.heroImage}
              />
            </div>
            <div className='flex flex-wrap justify-center gap-2 md:contents'>
              {LANDING_FLOAT_TAGS.map((tag) => (
                <span
                  key={tag.id}
                  className={`nt-pill nt-float-tag ${tag.posClass} px-3 py-1.5 text-xs font-medium`}
                >
                  <span className={`nt-dot ${tag.dotClass}`} />
                  {t(tag.labelKey)}
                </span>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
