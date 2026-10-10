// The Guide: what each control is, how the keyboard moves and the rules the
// window cannot state for itself. Its words live in guideContent.ts; this file
// only draws them.

import { guideSections } from './guideContent'
import { Modal } from './Modal'
import { useAutoScroll } from './useAutoScroll'

interface Props {
  onClose: () => void
  /** Opens About, which the Guide leads to (FR-072). */
  onAbout: () => void
}

export function GuideDialog({ onClose, onAbout }: Props) {
  // The Guide is longer than anything else the window shows, so the body reads
  // itself down gently and steps aside the moment the reader takes over. The
  // scroller is the body rather than the dialog, so Close never drifts away
  // with the words.
  const autoScroll = useAutoScroll()
  return (
    <Modal labelId="guide-title" role="dialog" onClose={onClose} pinnedActions>
      <h2 id="guide-title">How Visitron works</h2>
      <div className="dialog-body" ref={autoScroll}>
        {guideSections.map((section) => (
          <section className="guide-section" key={section.heading}>
            <h3>{section.heading}</h3>
            {section.intro && <p className="guide-intro">{section.intro}</p>}
            {section.steps && (
              <ol className="guide-steps">
                {section.steps.map((step) => (
                  <li key={step.text}>
                    {step.text}
                    {step.code && <pre className="guide-code">{step.code.join('\n')}</pre>}
                  </li>
                ))}
              </ol>
            )}
            {section.paragraphs?.map((text) => (
              <p key={text}>{text}</p>
            ))}
            {section.entries?.map((entry) => (
              <p className="guide-entry" key={entry.name}>
                <img className="guide-icon" src={entry.icon} alt="" draggable={false} />
                <span>
                  <b>{entry.name}</b>: {entry.text}
                </span>
              </p>
            ))}
            {section.rules?.map((rule) => (
              <p className="guide-rule" key={rule.title}>
                <b>{rule.title}</b> {rule.text}
              </p>
            ))}
          </section>
        ))}
      </div>
      <div className="actions">
        <button type="button" onClick={onAbout}>
          About
        </button>
        <button type="button" className="close-guide" onClick={onClose}>
          Close
        </button>
      </div>
    </Modal>
  )
}
