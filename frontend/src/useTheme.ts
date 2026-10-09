// The DOM half of the theme: what the page is wearing, plus remembering it.
//
// The tokens live in theme.css, which dresses the document dark unless the root
// element says otherwise, so this hook only ever sets one attribute.

import { useCallback, useEffect, useState } from 'react'
import { keepStored, readStored } from './storage'
import { nextTheme, storedTheme, themeKey, type Theme } from './theme'

/**
 * The theme remembered in this window, else the default. Storage that refuses
 * to be read holds nothing, which is the default too.
 */
export function rememberedTheme(): Theme {
  return storedTheme(readStored(themeKey))
}

/**
 * Dress the document, then answer the theme with the way to change it.
 *
 * The attribute is set on the root element rather than a class on the body,
 * because the tokens hang off `:root` and a dialog drawn outside the body's
 * tree would miss a class set there.
 */
export function useTheme(): [Theme, () => void] {
  const [theme, setTheme] = useState<Theme>(rememberedTheme)

  useEffect(() => {
    document.documentElement.dataset.theme = theme
  }, [theme])

  const toggle = useCallback(() => {
    setTheme((current) => {
      const wanted = nextTheme(current)
      keepStored(themeKey, wanted)
      return wanted
    })
  }, [])

  return [theme, toggle]
}
