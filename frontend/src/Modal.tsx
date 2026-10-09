// The shell every dialog is drawn in: a backdrop, a labelled box and Escape to
// close. The dialogs themselves say what goes inside.

import { useEffect, useRef, type ReactNode } from 'react'

// The controls a dialog may open on, in document order. The scrolling body is
// not among them; nor is anything disabled, so a dialog whose leading control
// is inert opens on the next live one rather than on a dead stop.
const firstStopSelector = [
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
].join(',')

interface Props {
  labelId: string
  role: 'dialog' | 'alertdialog'
  onClose: () => void
  /**
   * Whether the dialog holds a scrolling body. Its action row is then pinned
   * beneath that body, so Close stays where the reader expects it however tall
   * the content grows and never drifts as the body reads itself.
   */
  pinnedActions?: boolean
  children: ReactNode
}

export function Modal({ labelId, role, onClose, pinnedActions, children }: Props) {
  const box = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])
  // A dialog opens already on its first stop, unlike the window behind it,
  // which opens on nothing. The reader opened this deliberately, to do the one
  // thing it is for, so asking for a Tab press first costs a keystroke and
  // tells them nothing.
  //
  // The scrolling body is passed over even though it is reachable by Tab: it
  // is a page of words rather than a control, so a dialog that opened on it
  // would ring nothing and offer nothing.
  useEffect(() => {
    const stops = box.current?.querySelectorAll<HTMLElement>(firstStopSelector)
    stops?.[0]?.focus()
  }, [])
  return (
    <div className="backdrop">
      <div
        className={pinnedActions ? 'dialog pinned-actions' : 'dialog'}
        role={role}
        aria-modal="true"
        aria-labelledby={labelId}
        ref={box}
      >
        {children}
      </div>
    </div>
  )
}
