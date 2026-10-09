package windowfocus

import "testing"

func TestAskingForAWindowThatIsNotThereIsNotAFault(t *testing.T) {
	t.Parallel()
	// The suite owns no window, so this is the not-found path on Windows and
	// the no-op everywhere else. Either way it must return rather than panic:
	// the keyboard is worth a best effort and nothing more; a program that fell
	// over trying to focus itself would be worse than one that opened needing
	// a click.
	GiveTheKeyboardToThePage("a window no test has opened")
}

func TestAnEmptyTitleIsAnswerable(t *testing.T) {
	t.Parallel()
	// A title is only ever the product's own name, so this cannot happen in
	// production; it is here because a best-effort function that panics on an
	// odd argument is not best effort.
	GiveTheKeyboardToThePage("")
}
