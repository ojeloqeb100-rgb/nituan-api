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
import { LANDING_LAYOUT } from '../lib/landing'
import { CTA } from './sections/cta'
import { Faq } from './sections/faq'
import { Features } from './sections/features'
import { Hero } from './sections/hero'
import { HowItWorks } from './sections/how-it-works'
import { Models } from './sections/models'
import { Samples } from './sections/samples'
import { ToolsBar } from './sections/tools-bar'
import { WhoUses } from './sections/who-uses'

interface LandingPageProps {
  isAuthenticated: boolean
}

export function LandingPage(props: LandingPageProps) {
  return (
    <div className={LANDING_LAYOUT.shell} data-landing='ni-tuan'>
      <main>
        <Hero isAuthenticated={props.isAuthenticated} />
        <ToolsBar />
        <Models />
        <HowItWorks />
        <Samples />
        <WhoUses />
        <Features />
        <Faq />
        <CTA isAuthenticated={props.isAuthenticated} />
      </main>
    </div>
  )
}
