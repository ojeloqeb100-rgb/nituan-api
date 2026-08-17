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
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'

import {
  LANDING_FAQ_ITEMS,
  LANDING_LAYOUT,
  LANDING_SECTION_IDS,
} from '../../lib/landing'

export function Faq() {
  const { t } = useTranslation()

  return (
    <section id={LANDING_SECTION_IDS.faq} className={LANDING_LAYOUT.section}>
      <div className={LANDING_LAYOUT.sectionInner}>
        <div className='mb-10 max-w-xl'>
          <h2 className='text-2xl font-bold tracking-tight md:text-3xl'>
            {t('Common questions')}
          </h2>
        </div>
        <Accordion className='nt-card px-5'>
          {LANDING_FAQ_ITEMS.map((item) => (
            <AccordionItem
              key={item.id}
              value={item.id}
              className='border-[color:var(--nt-line)]'
            >
              <AccordionTrigger className='text-start text-[color:var(--nt-ink)] hover:no-underline'>
                {t(item.questionKey)}
              </AccordionTrigger>
              <AccordionContent className='text-[color:var(--nt-ink-soft)]'>
                {t(item.answerKey)}
              </AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      </div>
    </section>
  )
}
