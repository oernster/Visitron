// The window-style close cross in a dialog's top-right corner, ported from
// PigeonPost (Amendment 10). Modal draws it in every dialog. It is placed there
// by position alone and comes LAST in the markup: the first-stop rule then
// still opens on the dialog's own safe answer; Tab reaches the cross after the
// actions. Its accessible name differs from a dialog's own Close button, so a
// screen reader never hears two controls called Close.

interface Props {
  onClose: () => void
}

export function ModalClose({ onClose }: Props) {
  // Close on mousedown so the first click always lands, even while another
  // control is committing its value. Keyboard activation fires a click with
  // detail 0, which still closes; a real mouse click (detail above 0) was
  // already handled by mousedown, so it does not close twice.
  return (
    <button type="button" className="modal-close" aria-label="Close this dialog" title="Close"
      onMouseDown={onClose}
      onClick={(e) => {
        if (e.detail === 0) onClose()
      }}>
      &times;
    </button>
  )
}
