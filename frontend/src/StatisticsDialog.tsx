// One website's Statistics dialog (FR-045 to FR-049): its downloads to date by
// platform, repository and release, then its visitors by country over the
// chosen period as GoatCounter counts them. A country read that fails or is
// not set up leaves the downloads standing and says why in the countries' place.

import { useEffect, useState } from 'react'
import { api } from './api'
import type { NamedCount, Refused, Statistics, Visitors } from './api'
import { Modal } from './Modal'
import { CountryTable, visitorsGap } from './Visitors'
import { useAutoScroll } from './useAutoScroll'
import { periodLabel } from './WebsiteList'

interface TotalsProps {
  title: string
  rows: NamedCount[]
  /** The self-reading cycle, given to the one table whose rows can run long. */
  autoScroll?: (node: HTMLElement | null) => void
}

// A table whose rows scroll under its caption, which stays in place.
function Totals({ title, rows, autoScroll }: TotalsProps) {
  if (rows.length === 0) return null
  return (
    <div className="totals-scroll" ref={autoScroll}>
      <table className="totals">
        <caption>{title}</caption>
        <tbody>
          {rows.map((r) => (
            <tr key={r.name}>
              <td>{r.name}</td>
              <td>{r.count}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

const reading = 'Reading the statistics.'
const noDownloads = 'No downloads yet.'
const howCounted = 'As GoatCounter counts visitors by location, so these need not add up to page loads.'

function Countries({ visitors }: { visitors: Visitors }) {
  return visitorsGap(visitors) ?? <CountryTable countries={visitors.countries} note={howCounted} />
}

interface Props {
  id: number
  refused: Refused
  onClose: () => void
}

export function StatisticsDialog({ id, refused, onClose }: Props) {
  const [stats, setStats] = useState<Statistics | null>(null)
  const autoScroll = useAutoScroll()
  useEffect(() => {
    void api.statistics(id, (reason) => {
      refused(reason)
      onClose()
    }).then((found) => found && setStats(found))
  }, [id, refused, onClose])

  const downloads = stats ? [...stats.byPlatform, ...stats.byRepo, ...stats.byRelease] : []
  return (
    <Modal labelId="statistics-title" role="dialog" onClose={onClose} pinnedActions>
      {/* The downloads and the countries stand side by side, so a long release
          list never takes the countries' height. The headings stay put; the
          tables' rows scroll inside each table. The release list is the one
          that can run long, so it reads itself. */}
      <div className="dialog-body statistics-body">
        <h2 id="statistics-title">{stats ? stats.url.replace('https://', '') : 'Statistics'}</h2>
        {!stats && <p>{reading}</p>}
        {stats && (
          <div className="statistics-columns">
            <section className="statistics-section">
              <h3>Downloads to date</h3>
              {downloads.length === 0 && <p>{noDownloads}</p>}
              <div className="totals-row">
                <Totals title="By platform" rows={stats.byPlatform} />
                <Totals title="By repository" rows={stats.byRepo} />
                <Totals title="By release" rows={stats.byRelease} autoScroll={autoScroll} />
              </div>
            </section>
            <section className="statistics-section">
              <h3>Visitors by country, {periodLabel(stats.visitors.period)}</h3>
              <Countries visitors={stats.visitors} />
            </section>
          </div>
        )}
      </div>
      <div className="actions">
        <button type="button" onClick={onClose}>
          Close
        </button>
      </div>
    </Modal>
  )
}
