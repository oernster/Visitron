// A band dropdown: a band button that opens a list of entries (Amendment 18).
// Ported from PigeonPost's Menu.tsx, without the flyout submenus Visitron has
// no use for. The trigger keeps the band's picture and label, so Help reads
// exactly as it did before it became a menu.

import { useEffect, useRef, useState } from 'react'
import type { KeyboardEvent as ReactKeyboardEvent } from 'react'

/** One entry in a dropdown: an action, else a divider between groups of them. */
export type MenuItem = { label: string; onClick: () => void } | { separator: true }

/**
 * The grace the pointer gets after leaving a hover-opened menu before it
 * closes: long enough to travel from the trigger into the dropdown without the
 * menu slamming shut, short enough that it still feels dismissed on leave.
 */
export const hoverCloseDelayMs = 200

/** The enabled entries of an open dropdown, in the order they are drawn. */
function entries(panel: HTMLElement | null): HTMLButtonElement[] {
  if (!panel) return []
  return Array.from(panel.querySelectorAll<HTMLButtonElement>(':scope > .menu-item:not([disabled])'))
}

/** Answers whether key is one of those that open a closed menu. */
function opens(key: string): boolean {
  return key === 'ArrowDown' || key === 'Enter' || key === ' ' || key === 'Spacebar'
}

/** Answers whether key, pressed inside the dropdown, hands the reader back to the trigger. */
function leaves(key: string): boolean {
  return key === 'Escape' || key === 'Tab' || key === 'ArrowLeft' || key === 'ArrowRight'
}

interface Props {
  /**
   * The trigger's label, also the dropdown's accessible name. It is not drawn
   * as a heading: the trigger already says it, right above the dropdown.
   */
  label: string
  icon: string
  items: MenuItem[]
}

/**
 * Menu opens on a click, on hover and on Down, Enter or Space at the trigger.
 * A keyboard open moves focus to the first entry; a mouse open leaves it where
 * it was, so nothing is ringed unasked. Inside, Up and Down wrap and Home and
 * End jump; Escape, Tab, Left and Right close it back onto the trigger, so the
 * window's ring carries on from there. Up or Escape at the trigger retracts it.
 * It also closes on an outside click, when the pointer leaves and once an
 * entry is chosen. Every key it answers is consumed, so the window's ring does
 * not act on it a second time.
 */
export function Menu({ label, icon, items }: Props) {
  const [open, setOpen] = useState(false)
  const menu = useRef<HTMLDivElement>(null)
  const trigger = useRef<HTMLButtonElement>(null)
  const dropdown = useRef<HTMLDivElement>(null)
  const byKeyboard = useRef(false)
  const hoverClose = useRef<number | null>(null)

  const cancelHoverClose = () => {
    if (hoverClose.current !== null) {
      window.clearTimeout(hoverClose.current)
      hoverClose.current = null
    }
  }
  useEffect(() => cancelHoverClose, [])

  useEffect(() => {
    if (open && byKeyboard.current) entries(dropdown.current)[0]?.focus()
    if (!open) byKeyboard.current = false
  }, [open])

  useEffect(() => {
    if (!open) return
    const onDocDown = (e: MouseEvent) => {
      if (menu.current && !menu.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDocDown)
    return () => document.removeEventListener('mousedown', onDocDown)
  }, [open])

  const closeToTrigger = () => {
    setOpen(false)
    trigger.current?.focus()
  }

  const onTriggerKey = (e: ReactKeyboardEvent) => {
    if (opens(e.key)) {
      e.preventDefault()
      e.stopPropagation()
      if (open) {
        entries(dropdown.current)[0]?.focus()
      } else {
        byKeyboard.current = true
        setOpen(true)
      }
    } else if (open && (e.key === 'ArrowUp' || e.key === 'Escape')) {
      e.preventDefault()
      e.stopPropagation()
      setOpen(false)
    }
  }

  const onDropdownKey = (e: ReactKeyboardEvent) => {
    const list = entries(dropdown.current)
    if (list.length === 0) return
    const at = list.indexOf(document.activeElement as HTMLButtonElement)
    let next: number | null = null
    if (e.key === 'ArrowDown') next = (at + 1 + list.length) % list.length
    else if (e.key === 'ArrowUp') next = (at - 1 + list.length) % list.length
    else if (e.key === 'Home') next = 0
    else if (e.key === 'End') next = list.length - 1
    else if (!leaves(e.key)) return
    e.preventDefault()
    e.stopPropagation()
    if (next === null) closeToTrigger()
    else list[next].focus()
  }

  return (
    <div
      className="menu"
      ref={menu}
      onMouseEnter={() => {
        cancelHoverClose()
        setOpen(true)
      }}
      onMouseLeave={() => {
        cancelHoverClose()
        hoverClose.current = window.setTimeout(() => setOpen(false), hoverCloseDelayMs)
      }}
    >
      <button ref={trigger} type="button" className="band-button" aria-haspopup="menu" aria-expanded={open}
        onClick={() => setOpen((was) => !was)} onKeyDown={onTriggerKey}>
        <span className="band-picture">
          <img src={icon} alt="" />
        </span>
        <span>{label}</span>
      </button>
      {open && (
        <div className="menu-dropdown" role="menu" aria-label={label} ref={dropdown} onKeyDown={onDropdownKey}>
          {items.map((item, i) =>
            'separator' in item ? (
              <div key={i} className="menu-sep" role="separator" />
            ) : (
              <button key={item.label} type="button" className="menu-item" role="menuitem"
                onClick={() => {
                  setOpen(false)
                  item.onClick()
                }}>
                {item.label}
              </button>
            ),
          )}
        </div>
      )}
    </div>
  )
}
