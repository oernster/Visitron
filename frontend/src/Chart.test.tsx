import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Chart, niceTop, showDay } from './Chart'

const axis = (container: HTMLElement, which: string) =>
  Array.from(container.querySelectorAll(`.chart-${which} span`)).map((s) => s.textContent)

describe('the chart', () => {
  it('draws one bar per day against a clean axis, with the days named', () => {
    const { container } = render(
      <Chart title="Page loads per day" days={[{ day: '2026-10-08', count: 2 }, { day: '2026-10-09', count: 4 }]} empty="nothing yet" />,
    )
    expect(screen.getByRole('img', { name: 'Page loads per day, 6 in all' })).toBeInTheDocument()
    expect(screen.getByText('Page loads per day, 6 in all')).toBeInTheDocument()
    const bars = container.querySelectorAll('rect')
    expect(bars).toHaveLength(2)
    // The axis tops out at 5, the clean value above the largest count of 4.
    expect(axis(container, 'y')).toEqual(['5', '0'])
    expect(Number(bars[1].getAttribute('height'))).toBe((4 / 5) * 160)
    expect(Number(bars[0].getAttribute('height'))).toBe(Number(bars[1].getAttribute('height')) / 2)
    expect(bars[0]).toHaveTextContent('8 Oct: 2')
    expect(axis(container, 'x')).toEqual(['8 Oct', '9 Oct'])
  })

  it('names the first, middle and last day of a long period, oldest on the left', () => {
    const days = Array.from({ length: 30 }, (_, i) => ({ day: `2026-09-${String(i + 1).padStart(2, '0')}`, count: i === 29 ? 2 : 0 }))
    const { container } = render(<Chart title="Page loads per day" days={days} empty="nothing yet" />)
    expect(axis(container, 'x')).toEqual(['1 Sept', '16 Sept', '30 Sept'])
    expect(axis(container, 'y')).toEqual(['2', '1', '0'])
    const labels = container.querySelectorAll<HTMLElement>('.chart-x span')
    expect(labels[0].style.left).toBe('0px')
    expect(labels[2].style.right).toBe('0px')
    expect(labels[1]).toHaveClass('between')
  })

  it('draws nothing for no days and a flat line for days of nothing', () => {
    const { container, rerender } = render(<Chart title="Downloads per day" days={[]} empty="nothing yet" />)
    expect(screen.getByText('nothing yet')).toBeInTheDocument()
    expect(container.querySelectorAll('rect')).toHaveLength(0)
    expect(screen.getByText('Downloads per day, 0 in all')).toBeInTheDocument()

    rerender(<Chart title="Downloads per day" days={[{ day: '2026-10-09', count: 0 }]} empty="nothing yet" />)
    expect(container.querySelector('rect')?.getAttribute('height')).toBe('0')
    expect(axis(container, 'y')).toEqual(['1', '0'])
  })
})

describe('the axis', () => {
  it('rounds the largest count up to 1, 2 or 5 of a power of ten', () => {
    const cases: [number, number][] = [[0, 1], [1, 1], [2, 2], [3, 5], [7, 10], [10, 10], [11, 20], [23, 50], [120, 200], [5001, 10000]]
    for (const [most, top] of cases) expect(niceTop(most)).toBe(top)
  })

  it('names a day without shifting it across a time zone', () => {
    expect(showDay('2026-10-09')).toBe('9 Oct')
    expect(showDay('2026-01-01')).toBe('1 Jan')
  })
})
