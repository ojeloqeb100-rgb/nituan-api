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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { LANDING_LAYOUT, LANDING_SECTION_IDS } from '../../lib/landing'

export function Samples() {
  const { t } = useTranslation()
  const origin =
    typeof window === 'undefined' ? 'https://your-host' : window.location.origin
  const samples = useMemo(
    () => [
      {
        id: 'chat',
        titleKey: 'Chat completion',
        code: `from openai import OpenAI

client = OpenAI(
  base_url="${origin}/v1",
  api_key="sk-...",
)
client.chat.completions.create(
  model="gpt-4o-mini",
  messages=[{"role": "user", "content": "Hi"}],
)`,
      },
      {
        id: 'claude',
        titleKey: 'Claude messages',
        code: `import Anthropic from "@anthropic-ai/sdk"

const client = new Anthropic({
  baseURL: "${origin}",
  apiKey: "sk-...",
})
await client.messages.create({
  model: "claude-sonnet-4-5",
  max_tokens: 256,
  messages: [{ role: "user", content: "Hi" }],
})`,
      },
      {
        id: 'image',
        titleKey: 'Image generation',
        code: `curl ${origin}/v1/images/generations \\
  -H "Authorization: Bearer sk-..." \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-image","prompt":"a clay mascot writing a letter"}'`,
      },
    ],
    [origin]
  )

  return (
    <section
      id={LANDING_SECTION_IDS.samples}
      className={LANDING_LAYOUT.section}
    >
      <div className={LANDING_LAYOUT.sectionInner}>
        <div className='mb-10 max-w-xl'>
          <h2 className='text-2xl font-bold tracking-tight md:text-3xl'>
            {t('Generation samples')}
          </h2>
          <p className='mt-3 text-sm leading-relaxed text-[color:var(--nt-ink-soft)] md:text-base'>
            {t('Example calls you can run after changing base_url.')}
          </p>
        </div>
        <div className='grid gap-4 lg:grid-cols-3'>
          {samples.map((sample) => (
            <article key={sample.id} className='nt-card overflow-hidden'>
              <div className='border-b border-[color:var(--nt-line)] px-5 py-3 text-sm font-semibold'>
                {t(sample.titleKey)}
              </div>
              <pre className='overflow-x-auto px-5 py-4 text-[11px] leading-relaxed text-[color:var(--nt-ink-soft)]'>
                <code>{sample.code}</code>
              </pre>
            </article>
          ))}
        </div>
      </div>
    </section>
  )
}
