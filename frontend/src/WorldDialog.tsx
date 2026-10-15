// The World visitors dialog (FR-091 to FR-097): every website's visitors by
// country over the chosen period, as one GoatCounter read, drawn on a world map
// beside the table that carries the same figures. A read that fails or is not
// set up says why in their place.

import { Suspense, lazy, useEffect, useState } from 'react'
import { api } from './api'
import type { Refused, Visitors } from './api'
import { Modal } from './Modal'
import { CountryTable, visitorsGap } from './Visitors'
import { periodLabel } from './WebsiteList'
import { useAutoScroll } from './useAutoScroll'

// The map carries the world's shapes, most of the page's weight, so they load
// when the dialog first draws a map rather than with the window.
const WorldMap = lazy(() => import('./WorldMap').then((m) => ({ default: m.WorldMap })))
const drawing = 'Drawing the map.'

const title = 'World visitors'
const reading = 'Reading the countries.'
const howCounted =
  'As GoatCounter counts visitors by location across every website at once, so these need not add up to page loads or to each website’s own table.'

interface Props {
  refused: Refused
  onClose: () => void
}

export function WorldDialog({ refused, onClose }: Props) {
  const [visitors, setVisitors] = useState<Visitors | null>(null)
  // Every website's countries is the list that grows, so it reads itself.
  const autoScroll = useAutoScroll()
  useEffect(() => {
    void api.worldVisitors((reason) => {
      refused(reason)
      onClose()
    }).then((found) => found && setVisitors(found))
  }, [refused, onClose])

  const gap = visitors && visitorsGap(visitors)
  return (
    <Modal labelId="world-title" role="dialog" onClose={onClose} pinnedActions>
      <div className="dialog-body world-body">
        <h2 id="world-title">{title}</h2>
        {!visitors && <p>{reading}</p>}
        {visitors && (
          <section className="statistics-section">
            <h3>Visitors by country, {periodLabel(visitors.period)}</h3>
            {gap ?? (
              <div className="world-columns">
                <Suspense fallback={<p>{drawing}</p>}>
                  <WorldMap countries={visitors.countries} />
                </Suspense>
                <div className="world-table">
                  <CountryTable countries={visitors.countries} note={howCounted} autoScroll={autoScroll} />
                </div>
              </div>
            )}
          </section>
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
