// The selected website's detail (FR-042): charts of page loads and downloads
// per day, then the downloads to date. The tables by platform, repository and
// release live in the Statistics dialog (Amendment 23).

import type { Detail } from './api'
import { Chart } from './Chart'

// Why a chart can be empty while the website is fine (Amendment 7).
const noPageLoads =
  'None in this period yet. Page loads come from GoatCounter, so they need its key in Settings.'
const noDownloads =
  'None in this period yet. A day\'s downloads are the rise between two checks on different days,'
  + ' so they start to show from the second day Visitron checks.'

export function DetailPane({ detail }: { detail: Detail }) {
  return (
    <section className="detail" aria-label={detail.url}>
      <h2>{detail.url.replace('https://', '')}</h2>
      <Chart title="Page loads per day" days={detail.dailyPageLoads} empty={noPageLoads} />
      <Chart title="Downloads per day" days={detail.dailyDownloads} empty={noDownloads} />
      <p>Downloads to date: {detail.total}</p>
    </section>
  )
}
