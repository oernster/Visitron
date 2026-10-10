// One website's Statistics dialog (FR-045 to FR-049): its downloads to date by
// platform, repository and release, then its visitors by country over the
// chosen period as GoatCounter counts them. A country read that fails or is
// not set up leaves the downloads standing and says why in the countries' place.

import { useEffect, useState } from 'react'
import { api } from './api'
import type { NamedCount, Refused, Statistics } from './api'
import { Modal } from './Modal'
import { useAutoScroll } from './useAutoScroll'
import { periodLabel } from './WebsiteList'

function Totals({ title, rows }: { title: string; rows: NamedCount[] }) {
  if (rows.length === 0) return null
  return (
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
  )
}

// What stands in the countries' place when there is no table to show.
const reading = 'Reading the statistics.'
const noGoatCounter = 'Countries come from GoatCounter, so they need its account name and key in Settings.'
const noCountries = 'No visitors in this period yet.'
const noDownloads = 'No downloads yet.'
const howCounted = 'As GoatCounter counts visitors by location, so these need not add up to page loads.'

function Countries({ stats }: { stats: Statistics }) {
  if (stats.noGoatCounter) return <p>{noGoatCounter}</p>
  if (stats.countriesProblem) return <p role="alert">The countries could not be read: {stats.countriesProblem}</p>
  if (stats.countries.length === 0) return <p>{noCountries}</p>
  return (
    <>
      <table className="totals countries">
        <caption>By country</caption>
        <tbody>
          {stats.countries.map((c) => (
            <tr key={c.code}>
              <td>{c.name}</td>
              <td>{c.count}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p className="secret-help">{howCounted}</p>
    </>
  )
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
    <Modal labelId="statistics-title" role="dialog" onClose={onClose} pinnedActions wide>
      <div className="dialog-body" ref={autoScroll}>
        <h2 id="statistics-title">{stats ? stats.url.replace('https://', '') : 'Statistics'}</h2>
        {!stats && <p>{reading}</p>}
        {stats && (
          <>
            <h3>Downloads to date</h3>
            {downloads.length === 0 && <p>{noDownloads}</p>}
            <div className="totals-row">
              <Totals title="By platform" rows={stats.byPlatform} />
              <Totals title="By repository" rows={stats.byRepo} />
              <Totals title="By release" rows={stats.byRelease} />
            </div>
            <h3>Visitors by country, {periodLabel(stats.period)}</h3>
            <Countries stats={stats} />
          </>
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
