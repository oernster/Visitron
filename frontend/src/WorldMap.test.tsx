import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { WorldMap } from './WorldMap'
import type { Country } from './api'

// ISO 3166-1 numbers, the key the shapes carry.
const unitedKingdom = '826'
const france = '250'
const japan = '392'
const brazil = '076'
const antarctica = '010'

const some: Country[] = [
  { code: 'GB', name: 'United Kingdom', count: 100 },
  { code: 'FR', name: 'France', count: 10 },
  { code: 'JP', name: 'Japan', count: 1 },
]

function shape(container: HTMLElement, number: string): Element {
  const found = container.querySelector(`[data-shape="${number}"]`)
  if (!found) throw new Error(`no shape for ${number}`)
  return found
}

describe('the world map', () => {
  it('shades each country by its visitors and leaves the rest as land', () => {
    const { container } = render(<WorldMap countries={some} />)
    expect(shape(container, unitedKingdom)).toHaveClass('map-step-5')
    expect(shape(container, japan)).toHaveClass('map-step-1')
    expect(shape(container, france)).toHaveClass('map-step-3')
    expect(shape(container, brazil)).toHaveClass('map-step-0')
    expect(container.querySelector(`[data-shape="${antarctica}"]`)).toBeNull()
  })

  it('names each shaded country with its visitors', () => {
    const { container } = render(<WorldMap countries={some} />)
    expect(shape(container, unitedKingdom).querySelector('title')).toHaveTextContent('United Kingdom: 100')
    expect(shape(container, brazil).querySelector('title')).toBeNull()
  })

  it('is one image to a screen reader, keyed by the counts each shade covers', () => {
    render(<WorldMap countries={some} />)
    expect(screen.getByRole('img', { name: 'World map of visitors by country' })).toBeInTheDocument()
    const key = screen.getByLabelText('Visitors per shade')
    expect(key).toHaveTextContent(/^1.*100$/)
  })

  it('keeps a country it has no shape for out of the map without failing', () => {
    const { container } = render(<WorldMap countries={[{ code: 'XK', name: 'Kosovo', count: 3 }]} />)
    expect(container.querySelectorAll('.map-step-5')).toHaveLength(1)
  })
})
