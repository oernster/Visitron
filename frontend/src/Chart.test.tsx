import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Chart } from './Chart'

describe('the chart', () => {
  it('draws one bar per day, the tallest reaching the top', () => {
    const { container } = render(
      <Chart title="Page loads per day" days={[{ day: '2026-10-08', count: 2 }, { day: '2026-10-09', count: 4 }]} empty="nothing yet" />,
    )
    expect(screen.getByRole('img', { name: 'Page loads per day, 6 in all' })).toBeInTheDocument()
    expect(screen.getByText('Page loads per day: 6')).toBeInTheDocument()
    const bars = container.querySelectorAll('rect')
    expect(bars).toHaveLength(2)
    expect(bars[1].getAttribute('y')).toBe('0')
    expect(Number(bars[0].getAttribute('height'))).toBe(Number(bars[1].getAttribute('height')) / 2)
    expect(bars[0]).toHaveTextContent('2026-10-08: 2')
  })

  it('draws nothing for no days and a flat line for days of nothing', () => {
    const { container, rerender } = render(<Chart title="Downloads per day" days={[]} empty="nothing yet" />)
    expect(screen.getByText('nothing yet')).toBeInTheDocument()
    expect(container.querySelectorAll('rect')).toHaveLength(0)
    expect(screen.getByText('Downloads per day: 0')).toBeInTheDocument()

    rerender(<Chart title="Downloads per day" days={[{ day: '2026-10-09', count: 0 }]} empty="nothing yet" />)
    expect(container.querySelector('rect')?.getAttribute('height')).toBe('0')
  })
})
