// A bar chart of one figure per day, drawn as SVG so the page carries no
// charting library (FR-042). It is a picture, not a control: it takes no focus
// and rings in no state; the table beside it carries the numbers for anyone
// who cannot see it.

import type { DayCount } from './api'

const width = 600
const height = 160
const gap = 1

interface Props {
  title: string
  days: DayCount[]
}

export function Chart({ title, days }: Props) {
  const most = Math.max(1, ...days.map((d) => d.count))
  const bar = days.length > 0 ? width / days.length : width
  const total = days.reduce((sum, d) => sum + d.count, 0)
  return (
    <figure className="chart">
      <figcaption>
        {title}: {total}
      </figcaption>
      <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label={`${title}, ${total} in all`}>
        {days.map((d, i) => {
          const h = (d.count / most) * height
          return (
            <rect key={d.day} x={i * bar} y={height - h} width={Math.max(bar - gap, gap)} height={h}>
              <title>
                {d.day}: {d.count}
              </title>
            </rect>
          )
        })}
      </svg>
    </figure>
  )
}
