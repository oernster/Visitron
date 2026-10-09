// Typed access to the Go facade, ported from SymDiary.
//
// Wails injects window.go.main.App at load time. The interfaces below restate
// the Go DTOs in dto.go by hand; tests/structural/wire_test.go compares the
// two field for field.
//
// Every call takes a refusal handler as its last argument and never rejects: a
// refused call hands the reason to the handler and answers null. A call without
// a handler does not compile, so no refusal can go unseen (house robustness
// rule 11).

export interface State {
  name: string
  version: string
  /** Why the data could not be opened; empty when it opened. */
  problem: string
  periods: number[]
}

export interface WebsiteRow {
  id: number
  url: string
  repos: string[]
  pageLoads: number
  downloads: number
  totalDownloads: number
  sinceLastCheck: number
}

export interface Overview {
  rows: WebsiteRow[]
  period: number
  lastSuccess: string
  lastFailure: string
  failure: string
  noKey: boolean
  running: boolean
}

export interface NamedCount {
  name: string
  count: number
}

export interface DayCount {
  day: string
  count: number
}

export interface Detail {
  id: number
  url: string
  total: number
  byRepo: NamedCount[]
  byRelease: NamedCount[]
  byPlatform: NamedCount[]
  dailyPageLoads: DayCount[]
  dailyDownloads: DayCount[]
}

export interface Proposal {
  url: string
  found: string[]
  ticked: string[]
  problem: string
}

export interface Progress {
  done: number
  total: number
  site: string
}

export interface Settings {
  intervalHours: number
  minInterval: number
  maxInterval: number
  updateCheck: boolean
  startWithWindows: boolean
  goatCounterSet: boolean
  gitHubTokenSet: boolean
}

export interface Credit {
  work: string
  licence: string
  holder: string
}

export interface About {
  name: string
  author: string
  version: string
  licence: string
  credits: Credit[]
}

/** The two secrets, as the facade names them. */
export type SecretName = 'goatcounter' | 'github'

interface Bridge {
  State(): Promise<State>
  Overview(): Promise<Overview>
  Detail(id: number): Promise<Detail>
  Propose(entry: string, editID: number): Promise<Proposal>
  ConfirmRepo(text: string): Promise<string>
  SaveWebsite(id: number, url: string, repos: string[]): Promise<number>
  DeleteWebsite(id: number): Promise<void>
  Refresh(): Promise<void>
  Settings(): Promise<Settings>
  SaveInterval(hours: number): Promise<void>
  SavePeriod(days: number): Promise<void>
  SaveUpdateCheck(on: boolean): Promise<void>
  SaveStartWithWindows(on: boolean): Promise<void>
  SaveSecret(which: SecretName, value: string): Promise<string>
  RemoveSecret(which: SecretName): Promise<void>
  About(): Promise<About>
  Donate(): Promise<void>
  MinimiseToTray(): Promise<void>
  RequestQuit(): Promise<void>
}

interface WailsRuntime {
  EventsOn(name: string, callback: (data: never) => void): () => void
}

interface WailsWindow {
  go?: { main?: { App?: Bridge } }
  runtime?: WailsRuntime
}

/** Refused receives the reason a call did not happen, ready to show. */
export type Refused = (reason: string) => void

/** noWindow is the reason given when the page runs outside Visitron's window. */
export const noWindow = 'Visitron is not running: this page needs its window.'

/** progressEvent is the event Go sends as a check moves on. */
export const progressEvent = 'check-progress'

/** closeRequestEvent is Go asking for the close choice (FR-050, Amendment 4). */
export const closeRequestEvent = 'close-request'

const wails = (): WailsWindow => window as unknown as WailsWindow
const bridge = (): Bridge | null => wails().go?.main?.App ?? null

/** sentence turns a refusal from Go into something to show: a capital first letter. */
export function sentence(reason: unknown): string {
  const text = reason instanceof Error ? reason.message : String(reason)
  return text.charAt(0).toUpperCase() + text.slice(1)
}

async function ask<T>(invoke: (b: Bridge) => Promise<T>, refused: Refused): Promise<T | null> {
  const b = bridge()
  if (!b) {
    refused(noWindow)
    return null
  }
  try {
    return await invoke(b)
  } catch (reason) {
    refused(sentence(reason))
    return null
  }
}

async function act(invoke: (b: Bridge) => Promise<void>, refused: Refused): Promise<true | null> {
  return ask(async (b) => {
    await invoke(b)
    return true as const
  }, refused)
}

/** onProgress follows a running check; it answers the way to stop following. */
export function onProgress(callback: (progress: Progress) => void): () => void {
  return wails().runtime?.EventsOn(progressEvent, callback) ?? (() => undefined)
}

/** onCloseRequest follows the window's close button; it answers the way to stop. */
export function onCloseRequest(callback: () => void): () => void {
  return wails().runtime?.EventsOn(closeRequestEvent, callback) ?? (() => undefined)
}

export const api = {
  state: (refused: Refused) => ask((b) => b.State(), refused),
  overview: (refused: Refused) => ask((b) => b.Overview(), refused),
  detail: (id: number, refused: Refused) => ask((b) => b.Detail(id), refused),
  propose: (entry: string, editID: number, refused: Refused) =>
    ask((b) => b.Propose(entry, editID), refused),
  confirmRepo: (text: string, refused: Refused) => ask((b) => b.ConfirmRepo(text), refused),
  saveWebsite: (id: number, url: string, repos: string[], refused: Refused) =>
    ask((b) => b.SaveWebsite(id, url, repos), refused),
  deleteWebsite: (id: number, refused: Refused) => act((b) => b.DeleteWebsite(id), refused),
  refresh: (refused: Refused) => act((b) => b.Refresh(), refused),
  settings: (refused: Refused) => ask((b) => b.Settings(), refused),
  saveInterval: (hours: number, refused: Refused) => act((b) => b.SaveInterval(hours), refused),
  savePeriod: (days: number, refused: Refused) => act((b) => b.SavePeriod(days), refused),
  saveUpdateCheck: (on: boolean, refused: Refused) => act((b) => b.SaveUpdateCheck(on), refused),
  saveStartWithWindows: (on: boolean, refused: Refused) =>
    act((b) => b.SaveStartWithWindows(on), refused),
  saveSecret: (which: SecretName, value: string, refused: Refused) =>
    ask((b) => b.SaveSecret(which, value), refused),
  removeSecret: (which: SecretName, refused: Refused) => act((b) => b.RemoveSecret(which), refused),
  about: (refused: Refused) => ask((b) => b.About(), refused),
  // The page asks for the donation page; it never names an address. Its one
  // home is Go's product package.
  donate: (refused: Refused) => act((b) => b.Donate(), refused),
  minimiseToTray: (refused: Refused) => act((b) => b.MinimiseToTray(), refused),
  requestQuit: (refused: Refused) => act((b) => b.RequestQuit(), refused),
}
