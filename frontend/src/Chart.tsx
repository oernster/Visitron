// A bar chart of one figure per day, drawn as SVG so the page carries no
// charting library (FR-042). It is a picture, not a control: it takes no focus
// and rings in no state; the table beside it carries the numbers for anyone
// who cannot see it. Its axes are HTML beside the drawing, so their words stay
// sharp at any width and wear the text tokens rather than the bar colour
// (Amendment 17).

import type { DayCount } from './api'

const width = 600
const height = 160
const gap = 1

// The steps a top value is rounded up to within each power of ten, so the
// axis reads 0 / 5 / 10 rather than 0 / 3.5 / 7.
const niceSteps = [1, 2, 5, 10]
const decimal = 10

// How the axes and tooltips name a day: "9 Oct", in the window's own English.
const dayLabel = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short' })

/** niceTop rounds a largest count up to a clean axis top: 1, 2, 5, 10, 20, 50... */
export function niceTop(most: number): number {
  if (most <= 1) return 1
  const power = decimal ** Math.floor(Math.log10(most))
  return (niceSteps.find((step) => step * power >= most) ?? decimal) * power
}

/** showDay turns an ISO day, 2026-10-09, into "9 Oct" without a time zone shift. */
export function showDay(day: string): string {
  const [year, month, date] = day.split('-').map(Number)
  return dayLabel.format(new Date(year, month - 1, date))
}

interface Props {
  title: string
  days: DayCount[]
  /** What the chart is waiting for, said in place of bars while it has none. */
  empty: string
}

export function Chart({ title, days, empty }: Props) {
  const top = niceTop(Math.max(0, ...days.map((d) => d.count)))
  const half = top / 2
  const ticks = Number.isInteger(half) ? [top, half, 0] : [top, 0]
  const bar = days.length > 0 ? width / days.length : width
  const total = days.reduce((sum, d) => sum + d.count, 0)
  const middle = Math.floor(days.length / 2)
  const marks = days.length > 2 ? [0, middle, days.length - 1] : days.map((_, i) => i)
  return (
    <figure className="chart">
      <figcaption>
        {title}, {total} in all
      </figcaption>
      {total === 0 && <p className="chart-empty">{empty}</p>}
      <div className="chart-plot">
        <div className="chart-y" aria-hidden="true">
          {ticks.map((t) => (
            <span key={t}>{t}</span>
          ))}
        </div>
        <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" role="img"
          aria-label={`${title}, ${total} in all`}>
          {ticks.map((t) => (
            <line key={t} className="chart-grid" x1={0} x2={width} y1={height - (t / top) * height}
              y2={height - (t / top) * height} vectorEffect="non-scaling-stroke" />
          ))}
          {days.map((d, i) => {
            const h = (d.count / top) * height
            return (
              <rect key={d.day} x={i * bar} y={height - h} width={Math.max(bar - gap, gap)} height={h}>
                <title>
                  {showDay(d.day)}: {d.count}
                </title>
              </rect>
            )
          })}
        </svg>
        <div className="chart-x" aria-hidden="true">
          {marks.map((i) => {
            const first = i === 0
            const last = i === days.length - 1 && !first
            const place = first ? { left: 0 } : last ? { right: 0 } : { left: `${((i + 0.5) / days.length) * 100}%` }
            return (
              <span key={days[i].day} className={first || last ? undefined : 'between'} style={place}>
                {showDay(days[i].day)}
              </span>
            )
          })}
        </div>
      </div>
    </figure>
  )
}
