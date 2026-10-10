// The website list (FR-041) with the period chooser (FR-043). The table is
// ONE stop on the ring; Up and Down walk its rows and selecting a row drives
// the detail, which is what makes it a stop (keeb invariant 4). It paints no
// ring of its own; the selected row is the indicator (noborderfocus).

import type { KeyboardEvent } from 'react'
import type { Overview } from './api'

interface Props {
  overview: Overview
  periods: number[]
  selected: number | null
  onSelect: (id: number) => void
  onPeriod: (days: number) => void
}

// ONE_YEAR_DAYS is the period that reads as a year rather than as a count of
// days. The periods themselves come from Go (domain.Periods); this only names
// the one the label treats differently.
const ONE_YEAR_DAYS = 365

const periodLabel = (days: number) => (days === ONE_YEAR_DAYS ? '1 year' : `${days} days`)

// The download columns sit under one Downloads heading, so the first names
// the period it covers rather than repeating the word (Amendment 7).
const periodHeading = (days: number) => (days === ONE_YEAR_DAYS ? 'Last year' : `Last ${days} days`)

export function WebsiteList({ overview, periods, selected, onSelect, onPeriod }: Props) {
  const ids = overview.rows.map((r) => r.id)
  const step = (event: KeyboardEvent) => {
    if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return
    event.preventDefault()
    if (ids.length === 0) return
    const down = event.key === 'ArrowDown'
    const at = selected === null ? -1 : ids.indexOf(selected)
    // With nothing selected, Down starts at the first row and Up at the last
    // (Amendment 3); otherwise both step one row and wrap.
    if (at === -1) return onSelect(ids[down ? 0 : ids.length - 1])
    onSelect(ids[(at + (down ? 1 : -1) + ids.length) % ids.length])
  }
  return (
    <section className="websites">
      <label className="field period">
        Period
        <select value={overview.period} onChange={(e) => onPeriod(Number(e.target.value))}>
          {periods.map((d) => (
            <option key={d} value={d}>
              {periodLabel(d)}
            </option>
          ))}
        </select>
      </label>
      <table className="website-table" tabIndex={0} onKeyDown={step}
        aria-label="Websites">
        <thead>
          <tr>
            <th rowSpan={2}>Website</th>
            <th rowSpan={2}>Page loads</th>
            <th colSpan={3} className="group">Downloads</th>
          </tr>
          <tr>
            <th>{periodHeading(overview.period)}</th>
            <th>All time</th>
            <th>Since last check</th>
          </tr>
        </thead>
        <tbody>
          {overview.rows.map((r) => (
            <tr key={r.id} className={r.id === selected ? 'selected' : ''}
              aria-selected={r.id === selected} onClick={() => onSelect(r.id)}>
              <td title={r.repos.join(', ')}>{r.url.replace('https://', '')}</td>
              <td>{r.pageLoads}</td>
              <td>{r.downloads}</td>
              <td>{r.totalDownloads}</td>
              <td>{r.sinceLastCheck > 0 ? `+${r.sinceLastCheck}` : '0'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  )
}
