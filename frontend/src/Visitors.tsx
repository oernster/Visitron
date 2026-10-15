// Visitors by country as both dialogs show them: one website's in Statistics
// (FR-047 to FR-049) and every website's in World visitors (FR-092 to FR-097).
// The words for why there is no table live here once, so the two dialogs
// cannot come to say it differently.

import type { ReactElement } from 'react'
import type { Country, Visitors } from './api'

const noGoatCounter = 'Countries come from GoatCounter, so they need its account name and key in Settings.'
const noCountries = 'No visitors in this period yet.'

/** What stands in the countries' place when there is no table to show; null when there is one. */
export function visitorsGap(visitors: Visitors): ReactElement | null {
  if (visitors.noGoatCounter) return <p>{noGoatCounter}</p>
  if (visitors.problem) return <p role="alert">The countries could not be read: {visitors.problem}</p>
  if (visitors.countries.length === 0) return <p>{noCountries}</p>
  return null
}

interface Props {
  countries: Country[]
  /** How GoatCounter counts, beneath the table. */
  note: string
  /** The self-reading cycle, where this is the list in its dialog that can run long. */
  autoScroll?: (node: HTMLElement | null) => void
}

// The countries largest first, as the read answered them, rows scrolling under
// the caption.
export function CountryTable({ countries, note, autoScroll }: Props) {
  return (
    <>
      <div className="totals-scroll countries-scroll" ref={autoScroll}>
        <table className="totals countries">
          <caption>By country</caption>
          <tbody>
            {countries.map((c) => (
              <tr key={c.code}>
                <td>{c.name}</td>
                <td>{c.count}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="secret-help">{note}</p>
    </>
  )
}
