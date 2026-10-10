// The window-style close cross in a dialog's top-right corner, ported from
// PigeonPost (Amendment 10). It is drawn there by position alone, so a dialog
// places it LAST in its markup: the dialog's first-stop rule then still opens
// on the dialog's own safe answer; Tab reaches the cross after the actions.

interface Props {
  onClose: () => void
}

export function ModalClose({ onClose }: Props) {
  // Close on mousedown so the first click always lands, even while another
  // control is committing its value. Keyboard activation fires a click with
  // detail 0, which still closes; a real mouse click (detail above 0) was
  // already handled by mousedown, so it does not close twice.
  return (
    <button type="button" className="modal-close" aria-label="Close" title="Close"
      onMouseDown={onClose}
      onClick={(e) => {
        if (e.detail === 0) onClose()
      }}>
      &times;
    </button>
  )
}
