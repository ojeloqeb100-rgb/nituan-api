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
export const LANDING_BRAND_KEY = 'Ni Tuan AI'
export const LANDING_LOGO_SRC = '/ni-tuan-logo.png'

export const LANDING_SECTION_IDS = {
  models: 'models',
  howTo: 'how-to',
  samples: 'samples',
  whoUses: 'who-uses',
  faq: 'faq',
} as const

export const LANDING_ROUTES = {
  home: '/',
  signIn: '/sign-in',
  signUp: '/sign-up',
  pricing: '/pricing',
  dashboard: '/dashboard',
} as const

export const LANDING_TOOLS = [
  'OpenAI SDK',
  'Anthropic SDK',
  'Claude Code',
  'Cherry Studio',
  'Cline',
  'Codex CLI',
  'Dify',
  'LobeChat',
] as const

export const LANDING_MODEL_FAMILIES = [
  'Claude',
  'GPT',
  'Gemini',
  'DeepSeek',
  'Qwen',
  'Llama',
  'Grok',
  'Mistral',
] as const

export const LANDING_FLOAT_TAGS = [
  {
    id: 'topup',
    posClass: 'nt-float-tag-a',
    dotClass: 'nt-dot-peach',
    labelKey: '1:1 top-up',
  },
  {
    id: 'balance',
    posClass: 'nt-float-tag-b',
    dotClass: 'nt-dot-mint',
    labelKey: 'Balance never resets',
  },
  {
    id: 'logs',
    posClass: 'nt-float-tag-c',
    dotClass: 'nt-dot-amber',
    labelKey: 'Per-request records',
  },
] as const

export const LANDING_HIGHLIGHTS = [
  { id: 'rate', value: '¥1 = $1', labelKey: '1:1 top-up' },
  {
    id: 'balance',
    value: '∞',
    labelKey: 'Balance never resets or expires',
  },
  {
    id: 'oneline',
    value: '1',
    labelKey: 'Change one line to connect',
  },
] as const

export const LANDING_FAQ_ITEMS = [
  {
    id: 'key',
    questionKey: 'How do I get a key?',
    answerKey:
      'Create a free account, open the console, and generate an API key.',
  },
  {
    id: 'switch',
    questionKey: 'How do I switch from OpenAI?',
    answerKey:
      'Keep your SDK or client. Change base_url to this site and use your new key.',
  },
  {
    id: 'expire',
    questionKey: 'Does unused balance expire?',
    answerKey: 'No. Your balance stays until you use it.',
  },
  {
    id: 'price',
    questionKey: 'How is usage priced?',
    answerKey:
      'Models are billed using official price ratios. See the public model list.',
  },
  {
    id: 'tools',
    questionKey: 'Which tools are supported?',
    answerKey:
      'Any client that lets you set an OpenAI- or Anthropic-compatible API address.',
  },
] as const

export const LANDING_LAYOUT = {
  shell: 'nt-landing relative min-h-svh overflow-x-clip scroll-smooth',
  heroGrid:
    'mx-auto grid max-w-6xl grid-cols-1 items-center gap-12 px-4 md:px-6 lg:grid-cols-12 lg:gap-8',
  heroVisual:
    'relative mx-auto flex w-full max-w-[22rem] flex-col items-center gap-3 lg:max-w-[26rem]',
  heroGlow:
    'nt-hero-glow pointer-events-none absolute inset-[-10%] rounded-full',
  heroArt:
    'nt-hero-art relative z-[1] aspect-square w-full rounded-[1.75rem]',
  heroImage: 'size-full object-contain object-center p-3',
  toolsRow: 'flex flex-wrap items-center justify-center gap-2',
  section: 'scroll-mt-24 px-4 py-16 md:px-6 md:py-20',
  sectionInner: 'mx-auto w-full max-w-6xl',
} as const

export function isLandingHashHref(href: string): boolean {
  return href.startsWith('#')
}

export function isOfficialDocsHref(href: string): boolean {
  return href.includes('docs.newapi.pro')
}
