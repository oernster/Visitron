// The World visitors map (FR-094, FR-095): every country Natural Earth draws
// at 1:50m, each shaded by its visitors. GoatCounter names a country by its
// two-letter ISO code while the shapes are keyed by its ISO number, so the code
// is turned into the number to find the shape. The shapes ship in the page, so
// drawing the map contacts no one (NFR-PRIV-001).
//
// The map is a picture beside the table, not a control: it takes no focus and
// a screen reader hears it as one image, while the table carries the figures.

import { geoNaturalEarth1, geoPath } from 'd3-geo'
import type { Feature, FeatureCollection, Geometry } from 'geojson'
import isoCountries from 'i18n-iso-countries'
import { feature } from 'topojson-client'
import type { GeometryCollection, Topology } from 'topojson-specification'
import atlas from 'world-atlas/countries-50m.json'
import type { Country } from './api'
import { shadeKey, shadeStep } from './mapShade'

/** The map's drawn width; its height follows from the projection. */
const mapWidth = 960

/** Antarctica is left off: no visitor comes from it and it takes a third of the height. */
const leftOff = isoCountries.alpha2ToNumeric('AQ')

interface Shape {
  /** The country's ISO number; "" for the few shapes Natural Earth gives none. */
  id: string
  d: string
}

interface Drawn {
  height: number
  shapes: Shape[]
}

let drawn: Drawn | null = null

// The shapes are projected once, on the first map drawn, rather than on every
// opening: the projection does not change with the figures.
function shapes(): Drawn {
  if (drawn) return drawn
  const topology = atlas as unknown as Topology<{ countries: GeometryCollection }>
  const all = feature(topology, topology.objects.countries) as FeatureCollection<Geometry>
  const kept: FeatureCollection<Geometry> = {
    type: 'FeatureCollection',
    features: all.features.filter((f) => String(f.id ?? '') !== leftOff),
  }
  const projection = geoNaturalEarth1().fitWidth(mapWidth, kept)
  const path = geoPath(projection)
  const [, [, bottom]] = path.bounds(kept)
  drawn = {
    height: Math.ceil(bottom),
    shapes: kept.features.map((f: Feature<Geometry>) => ({ id: String(f.id ?? ''), d: path(f) ?? '' })),
  }
  return drawn
}

const mapLabel = 'World map of visitors by country'
const keyLabel = 'Visitors per shade'

function range(low: number, high: number): string {
  return low === high ? String(low) : `${low} to ${high}`
}

export function WorldMap({ countries }: { countries: Country[] }) {
  const { height, shapes: all } = shapes()
  const byNumber = new Map<string, Country>()
  for (const c of countries) {
    const number = isoCountries.alpha2ToNumeric(c.code)
    if (number) byNumber.set(number, c)
  }
  const largest = countries.reduce((most, c) => Math.max(most, c.count), 0)
  return (
    <figure className="world-map">
      <svg viewBox={`0 0 ${mapWidth} ${height}`} role="img" aria-label={mapLabel}>
        {all.map((s, i) => {
          const c = byNumber.get(s.id)
          const step = c ? shadeStep(c.count, largest) : 0
          return (
            // Keyed by place: a number can name several shapes (Australia and
            // its islands share one) and the list never changes order.
            <path key={i} d={s.d} className={`map-country map-step-${step}`}
              data-shape={s.id}>
              {c && <title>{`${c.name}: ${c.count}`}</title>}
            </path>
          )
        })}
      </svg>
      <figcaption className="map-key" aria-label={keyLabel}>
        {shadeKey(largest).map((k) => (
          <span key={k.step} className="map-key-step">
            <span className={`map-swatch map-step-${k.step}`} aria-hidden="true" />
            {range(k.low, k.high)}
          </span>
        ))}
      </figcaption>
    </figure>
  )
}
