import { act, render } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { themeKey } from './theme'
import { useTheme } from './useTheme'

/** Probe wears the hook so a test can press its toggle. */
function Probe() {
  const [theme, toggle] = useTheme()
  return (
    <button type="button" onClick={toggle}>
      {theme}
    </button>
  )
}

afterEach(() => {
  window.localStorage.clear()
  delete document.documentElement.dataset.theme
  vi.restoreAllMocks()
})

describe('wearing the theme', () => {
  it('dresses the document dark on a first opening', () => {
    render(<Probe />)
    expect(document.documentElement.dataset.theme).toBe('dark')
  })

  it('opens in what was remembered', () => {
    window.localStorage.setItem(themeKey, 'light')
    render(<Probe />)
    expect(document.documentElement.dataset.theme).toBe('light')
  })

  it('changes the document and remembers the choice', () => {
    const { getByRole } = render(<Probe />)
    act(() => getByRole('button').click())

    expect(document.documentElement.dataset.theme).toBe('light')
    expect(window.localStorage.getItem(themeKey)).toBe('light')

    act(() => getByRole('button').click())
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(window.localStorage.getItem(themeKey)).toBe('dark')
  })

  it('still opens when storage refuses to be read', () => {
    // A window with site data blocked throws rather than answering nothing. A
    // theme is not worth a dead page.
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('storage is not available')
    })
    render(<Probe />)
    expect(document.documentElement.dataset.theme).toBe('dark')
  })

  it('still changes when storage refuses to be written', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('storage is full')
    })
    const { getByRole } = render(<Probe />)
    act(() => getByRole('button').click())

    // The window is wearing it; only the remembering was lost.
    expect(document.documentElement.dataset.theme).toBe('light')
  })
})
