// What each secret is for and how to get it: the one home for the words
// Settings shows under each field (Amendment 7) and the empty detail pane
// repeats while one is missing (Amendment 9).

import type { Overview, SecretName, Settings } from './api'

/** missingSecrets names the secrets the overview says are not set. */
export function missingSecrets(overview: Overview): SecretName[] {
  const missing: SecretName[] = []
  if (overview.noKey) missing.push('goatcounter')
  if (overview.noToken) missing.push('github')
  return missing
}

export interface SecretHelp {
  which: SecretName
  label: string
  set: keyof Settings
  why: string
  steps: string[]
}

// The steps were read from GoatCounter's API help and GitHub's token and
// rate-limit pages on 2026-10-10; re-read them there when either site changes
// its menus.
export const secretHelp: SecretHelp[] = [
  {
    which: 'goatcounter', label: 'GoatCounter API key', set: 'goatCounterSet',
    why: 'Needed for page loads, with your GoatCounter account name: without them every Page loads figure stays at 0.',
    steps: [
      'Your GoatCounter account name is the part before .goatcounter.com in the address you use: for yourname.goatcounter.com it is yourname. Type it into GoatCounter account name below and press Save name.',
      'Open that address in your browser and sign in.',
      'Choose your username in the top menu, then API.',
      'Create a new key with the Read statistics permission; it needs nothing else.',
      'Copy the key, paste it below and press Save. Visitron tries it at once and says whether it works.',
    ],
  },
  {
    which: 'github', label: 'GitHub token (optional)', set: 'gitHubTokenSet',
    why: 'Downloads are read without one; GitHub then allows 60 requests an hour, which a token raises to 5,000.',
    steps: [
      'On github.com, choose your profile picture, then Settings.',
      'Choose Developer settings at the foot of the left-hand list, then Personal access tokens, Fine-grained tokens, Generate new token.',
      'Name it Visitron and pick an expiry. Leave every permission unticked: a token can always read public repositories.',
      'Press Generate token, copy it (GitHub shows it only once), paste it below and press Save.',
    ],
  },
]
