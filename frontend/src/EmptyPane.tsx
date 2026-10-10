// What the detail side shows while no website is selected (Amendment 9): the
// application's mark, faded, over a line saying how to fill the side, then
// what each missing secret would add, in the words Settings uses.

import type { SecretName } from './api'
import appIcon from './assets/icons/application-icon.png'
import { secretHelp } from './secretHelp'

interface Props {
  /** Whether any website is recorded: with none, the way in is Add. */
  hasWebsites: boolean
  missing: SecretName[]
}

export function EmptyPane({ hasWebsites, missing }: Props) {
  const wanted = secretHelp.filter(({ which }) => missing.includes(which))
  return (
    <section className="empty-pane" aria-label="No website selected">
      <img className="empty-mark" src={appIcon} alt="" draggable={false} />
      <p className="empty-lead">
        {hasWebsites ? 'Select a website to see its statistics.' : 'Add a website to start.'}
      </p>
      {wanted.length > 0 && (
        <div className="empty-settings">
          <p>Fill these in under Settings for fuller figures:</p>
          <ul>
            {wanted.map(({ which, label, why }) => (
              <li key={which}>
                <b>{label}</b>: {why}
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  )
}
