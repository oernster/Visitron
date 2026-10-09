import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { WebsiteList } from './WebsiteList'
import { anOverview, periods } from './bridge-fake'

function list(selected: number | null, overview = anOverview) {
  const onSelect = vi.fn()
  const onPeriod = vi.fn()
  render(<WebsiteList overview={overview} periods={periods} selected={selected}
    onSelect={onSelect} onPeriod={onPeriod} />)
  return { onSelect, onPeriod, table: screen.getByRole('table', { name: 'Websites' }) }
}

describe('the website list', () => {
  it('shows each website without its scheme and its counts', () => {
    list(1)
    const rows = screen.getAllByRole('row')
    const cells = (i: number) => Array.from(rows[i].querySelectorAll('td')).map((c) => c.textContent)
    expect(cells(1)).toEqual(['symdiary.com', '120', '14', '80', '+3'])
    expect(rows[1]).toHaveAttribute('aria-selected', 'true')
    expect(cells(2)).toEqual(['ernster.dev/WhatDay', '9', '0', '0', '0'])
    expect(rows[2]).toHaveAttribute('aria-selected', 'false')
  })

  it('is one stop whose arrows walk the rows and wrap', () => {
    const { onSelect, table } = list(null)
    expect(table).toHaveAttribute('tabIndex', '0')
    fireEvent.keyDown(table, { key: 'ArrowDown' })
    expect(onSelect).toHaveBeenLastCalledWith(1)
  })

  it('starts at the last row when Up is pressed with nothing selected', () => {
    const { onSelect, table } = list(null)
    fireEvent.keyDown(table, { key: 'ArrowUp' })
    expect(onSelect).toHaveBeenLastCalledWith(2)
  })

  it('wraps from the first row to the last', () => {
    const { onSelect, table } = list(1)
    fireEvent.keyDown(table, { key: 'ArrowUp' })
    expect(onSelect).toHaveBeenLastCalledWith(2)
  })

  it('wraps from the last row to the first and ignores other keys', () => {
    const { onSelect, table } = list(2)
    fireEvent.keyDown(table, { key: 'ArrowDown' })
    expect(onSelect).toHaveBeenLastCalledWith(1)
    fireEvent.keyDown(table, { key: 'Enter' })
    expect(onSelect).toHaveBeenCalledTimes(1)
  })

  it('selects nothing when there are no rows', () => {
    const { onSelect, table } = list(null, { ...anOverview, rows: [] })
    fireEvent.keyDown(table, { key: 'ArrowDown' })
    expect(onSelect).not.toHaveBeenCalled()
  })

  it('selects a clicked row and offers every period', () => {
    const { onSelect, onPeriod } = list(null)
    fireEvent.click(screen.getByText('ernster.dev/WhatDay'))
    expect(onSelect).toHaveBeenCalledWith(2)

    const period = screen.getByRole('combobox', { name: 'Period' })
    expect(period).toHaveValue('30')
    expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual(
      ['7 days', '30 days', '90 days', '1 year'],
    )
    fireEvent.change(period, { target: { value: '90' } })
    expect(onPeriod).toHaveBeenCalledWith(90)
  })
})
