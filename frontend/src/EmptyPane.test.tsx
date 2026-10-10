import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { EmptyPane } from './EmptyPane'
import { anOverview } from './bridge-fake'
import { missingSecrets } from './secretHelp'

const pane = () => screen.getByRole('region', { name: 'No website selected' })

describe('the empty detail pane', () => {
  it('says how to fill it and what each missing secret would add', () => {
    render(<EmptyPane hasWebsites missing={['goatcounter', 'github']} />)
    expect(pane()).toHaveTextContent('Select a website to see its statistics.')
    expect(pane().querySelector('img.empty-mark')).toHaveAttribute('alt', '')
    const items = screen.getAllByRole('listitem').map((li) => li.textContent)
    expect(items).toEqual([
      'GoatCounter API key: Needed for page loads, with your GoatCounter account name: without them every Page loads figure stays at 0.',
      'GitHub token (optional): Downloads are read without one; GitHub then allows 60 requests an hour, which a token raises to 5,000.',
    ])
  })

  it('says nothing of Settings once both secrets are set; asks for a website when there is none', () => {
    render(<EmptyPane hasWebsites={false} missing={[]} />)
    expect(pane()).toHaveTextContent('Add a website to start.')
    expect(screen.queryByText(/under Settings/)).toBeNull()
  })

  it('reads which secrets are missing from the overview', () => {
    expect(missingSecrets(anOverview)).toEqual([])
    expect(missingSecrets({ ...anOverview, noKey: true, noToken: true })).toEqual(['goatcounter', 'github'])
    expect(missingSecrets({ ...anOverview, noToken: true })).toEqual(['github'])
  })
})
