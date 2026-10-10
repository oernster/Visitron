// The close choice (FR-050, Amendment 4), ported from PigeonPost: the window's
// close button asks whether to keep Visitron running in the tray or to quit.
// Dismissing it (Escape or the cross, Amendment 10) cancels the close and
// leaves the window open, so an accidental click costs nothing.

import { useState } from 'react'
import { Modal } from './Modal'

interface Props {
  onMinimise: () => void
  onQuit: () => void
  onCancel: () => void
}

export function CloseChoiceDialog({ onMinimise, onQuit, onCancel }: Props) {
  // Sampled once as this dialog opens, before its own box exists: any dialog
  // already open is work a quit would take down with it.
  const [workOpen] = useState(() => document.querySelector('.dialog') !== null)
  // The safe answer is drawn first, so the dialog's first-stop rule opens on
  // it and a stray Enter never quits: Go back while work is open, else
  // Minimise to tray.
  return (
    <Modal labelId="close-title" role="alertdialog" onClose={onCancel}>
      <h2 id="close-title">Close Visitron</h2>
      <p>
        Keep Visitron running in the system tray or quit the application? While it runs in the
        tray, it keeps checking your websites on schedule.
      </p>
      {workOpen && (
        <p className="problem" role="alert">
          A dialog you were working in is still open and anything unsaved in it may be lost if you
          quit. Go back to finish or save it first.
        </p>
      )}
      <div className="actions">
        {workOpen && (
          <button type="button" onClick={onCancel}>
            Go back
          </button>
        )}
        <button type="button" onClick={onMinimise}>
          Minimise to tray
        </button>
        <button type="button" className="danger" onClick={onQuit}>
          Quit
        </button>
      </div>
    </Modal>
  )
}
