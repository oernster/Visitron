// A stand-in for the window's Go facade, so components can be driven in a test.
// Ported from SymDiary.
//
// Nothing here fakes a rule: every answer is stated by the test. A method not
// installed rejects, which is how a test says "this call should not happen".

import { vi } from 'vitest'
import { closeRequestEvent, noWindow, progressEvent, sentence } from './api'
import type {
  About, Detail, Overview, Progress, Proposal, SecretName, Settings, State, Statistics, Update, WebsiteRow,
} from './api'

/** noWindowShown is that refusal as the status line shows it. */
export const noWindowShown = sentence(noWindow)

type Fn = ReturnType<typeof vi.fn>

export interface FakeBridge {
  State: Fn
  Overview: Fn
  Detail: Fn
  Statistics: Fn
  Propose: Fn
  ConfirmRepo: Fn
  SaveWebsite: Fn
  DeleteWebsite: Fn
  Refresh: Fn
  Settings: Fn
  SaveInterval: Fn
  SavePeriod: Fn
  SaveUpdateCheck: Fn
  SaveStartWithWindows: Fn
  SaveSelfDownloads: Fn
  SaveSecret: Fn
  Secret: Fn
  SaveGoatCounterSite: Fn
  RemoveSecret: Fn
  About: Fn
  Donate: Fn
  MinimiseToTray: Fn
  RequestQuit: Fn
  CheckForUpdates: Fn
  DownloadUpdate: Fn
  SkipUpdate: Fn
}

export const periods = [7, 30, 90, 365]

export const aState: State = { name: 'Visitron', version: '1.0.0', problem: '', periods }

export const widget: WebsiteRow = {
  id: 1,
  url: 'https://example.org',
  repos: ['someone/Widget'],
  pageLoads: 120,
  downloads: 14,
  totalDownloads: 80,
  sinceLastCheck: 3,
}

export const app: WebsiteRow = {
  id: 2,
  url: 'https://example.com/App',
  repos: [],
  pageLoads: 9,
  downloads: 0,
  totalDownloads: 0,
  sinceLastCheck: 0,
}

export const anOverview: Overview = {
  rows: [widget, app],
  period: 30,
  lastSuccess: '9 Oct 2026 20:00',
  lastFailure: '',
  failure: '',
  noKey: false,
  noToken: false,
  running: false,
  since: '',
}

export const aDetail: Detail = {
  id: 1,
  url: 'https://example.org',
  total: 80,
  dailyPageLoads: [{ day: '2026-10-08', count: 4 }, { day: '2026-10-09', count: 6 }],
  dailyDownloads: [{ day: '2026-10-08', count: 1 }, { day: '2026-10-09', count: 2 }],
}

export const someStatistics: Statistics = {
  id: 1,
  url: 'https://example.org',
  period: 30,
  byPlatform: [{ name: 'Windows', count: 70 }, { name: 'macOS', count: 10 }],
  byRepo: [{ name: 'someone/Widget', count: 80 }],
  byRelease: [{ name: 'v1.0.0', count: 80 }],
  countries: [{ code: 'GB', name: 'United Kingdom', count: 5 }, { code: 'FR', name: 'France', count: 2 }],
  noGoatCounter: false,
  countriesProblem: '',
}

export const aProposal: Proposal = {
  url: 'https://example.org',
  found: ['someone/Widget', 'someone/Widget-site'],
  ticked: ['someone/Widget'],
  problem: '',
}

export const someSettings: Settings = {
  intervalHours: 24,
  minInterval: 1,
  maxInterval: 168,
  updateCheck: true,
  startWithWindows: false,
  goatCounterSet: false,
  gitHubTokenSet: true,
  goatCounterSite: '',
  selfDownloads: 0,
  maxSelfDownloads: 10,
}

/** someSecrets are the stored values behind someSettings: the key not set, the token set. */
export const someSecrets: Record<SecretName, string> = { goatcounter: '', github: 'a-github-token' }

export const anAbout: About = {
  name: 'Visitron',
  author: 'Oliver Ernster',
  version: '1.0.0',
  licence: 'GNU GENERAL PUBLIC LICENSE',
  credits: [{ work: 'Go', licence: 'BSD 3-Clause', holder: 'The Go Authors' }],
}

/** current is what a check answers by default: nothing newer. */
export const current: Update = { outcome: 'current', running: '1.0.0', latest: '1.0.0' }

/** anOffer is a check that found a newer release. */
export const anOffer: Update = { outcome: 'available', running: '1.0.0', latest: '1.1.0' }

/** installBridge puts a fake facade on the window and answers it. */
export function installBridge(answers: Partial<FakeBridge> = {}): FakeBridge {
  const refuse = () => Promise.reject(new Error('this call was not expected'))
  const bridge = {
    State: vi.fn(() => Promise.resolve(aState)),
    Overview: vi.fn(() => Promise.resolve(anOverview)),
    Detail: vi.fn(() => Promise.resolve(aDetail)),
    Statistics: vi.fn(() => Promise.resolve(someStatistics)),
    Propose: vi.fn(refuse),
    ConfirmRepo: vi.fn(refuse),
    SaveWebsite: vi.fn(refuse),
    DeleteWebsite: vi.fn(refuse),
    Refresh: vi.fn(refuse),
    Settings: vi.fn(() => Promise.resolve(someSettings)),
    SaveInterval: vi.fn(refuse),
    SavePeriod: vi.fn(refuse),
    SaveUpdateCheck: vi.fn(refuse),
    SaveStartWithWindows: vi.fn(refuse),
    SaveSelfDownloads: vi.fn(refuse),
    SaveSecret: vi.fn(refuse),
    Secret: vi.fn((which: SecretName) => Promise.resolve(someSecrets[which])),
    SaveGoatCounterSite: vi.fn(refuse),
    RemoveSecret: vi.fn(refuse),
    About: vi.fn(() => Promise.resolve(anAbout)),
    Donate: vi.fn(refuse),
    MinimiseToTray: vi.fn(refuse),
    RequestQuit: vi.fn(refuse),
    CheckForUpdates: vi.fn(() => Promise.resolve(current)),
    DownloadUpdate: vi.fn(refuse),
    SkipUpdate: vi.fn(refuse),
    ...answers,
  } as FakeBridge
  ;(window as unknown as { go: unknown }).go = { main: { App: bridge } }
  return bridge
}

/** installEvents puts a fake event runtime on the window; send fires a
 * progress event and close presses the window's close button. */
export function installEvents(): { send: (p: Progress) => void; close: () => void } {
  const listeners: Record<string, ((data?: Progress) => void)[]> = {}
  ;(window as unknown as { runtime: unknown }).runtime = {
    EventsOn: (name: string, callback: (data?: Progress) => void) => {
      ;(listeners[name] ??= []).push(callback)
      return () => undefined
    },
  }
  return {
    send: (p) => listeners[progressEvent]?.forEach((l) => l(p)),
    close: () => listeners[closeRequestEvent]?.forEach((l) => l()),
  }
}
