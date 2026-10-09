// The selected website's detail (FR-042): totals by repository, release and
// platform, then charts of page loads and downloads per day.

import type { Detail, NamedCount } from './api'
import { Chart } from './Chart'

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

export function DetailPane({ detail }: { detail: Detail }) {
  return (
    <section className="detail" aria-label={detail.url}>
      <h2>{detail.url.replace('https://', '')}</h2>
      <Chart title="Page loads per day" days={detail.dailyPageLoads} />
      <Chart title="Downloads per day" days={detail.dailyDownloads} />
      <p>Downloads to date: {detail.total}</p>
      <div className="totals-row">
        <Totals title="By platform" rows={detail.byPlatform} />
        <Totals title="By repository" rows={detail.byRepo} />
        <Totals title="By release" rows={detail.byRelease} />
      </div>
    </section>
  )
}
