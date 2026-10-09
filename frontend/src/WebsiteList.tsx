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

const periodLabel = (days: number) => (days === 365 ? '1 year' : `${days} days`)

export function WebsiteList({ overview, periods, selected, onSelect, onPeriod }: Props) {
  const ids = overview.rows.map((r) => r.id)
  const step = (event: KeyboardEvent) => {
    if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return
    event.preventDefault()
    if (ids.length === 0) return
    const at = selected === null ? -1 : ids.indexOf(selected)
    const delta = event.key === 'ArrowDown' ? 1 : -1
    onSelect(ids[(at + delta + ids.length) % ids.length])
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
            <th>Website</th>
            <th>Page loads</th>
            <th>Downloads</th>
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
