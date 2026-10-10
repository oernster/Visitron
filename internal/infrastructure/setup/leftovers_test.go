package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// holds answers whether folder is parent or sits inside it.
func holds(parent, folder string) bool {
	parent = filepath.Clean(parent)
	folder = filepath.Clean(folder)
	if parent == folder {
		return true
	}
	return strings.HasPrefix(folder+string(os.PathSeparator), parent+string(os.PathSeparator))
}

// TestTheRecordIsNeverClearedWithTheLeftovers is FR-080 made checkable. An
// uninstall clears what the window kept without asking; the data, with the
// run log that lives beside it, goes only when the box naming it has been
// ticked. Comparing the
// folders rather than trusting the names is what makes a later change to any
// of these paths fail here instead of quietly deleting the owner's history.
func TestTheRecordIsNeverClearedWithTheLeftovers(t *testing.T) {
	base := t.TempDir()
	t.Setenv("APPDATA", filepath.Join(base, "Roaming"))
	t.Setenv("LOCALAPPDATA", filepath.Join(base, "Local"))

	record, err := RecordDir()
	if err != nil {
		t.Fatalf("RecordDir: %v", err)
	}
	folders := Leftovers()
	if len(folders) != 1 {
		t.Fatalf("Leftovers cleared %d folder(s), want the window's alone", len(folders))
	}
	for _, folder := range folders {
		if holds(folder, record) {
			t.Errorf("an uninstall clears %q, which holds the record at %q: "+
				"FR-080 keeps the record unless the user asks for it by name", folder, record)
		}
	}
}

// TestTheRecordGoesOnlyWhenItIsAskedFor proves the other half: the folder the
// uninstall screen offers to delete is the one holding the record file it
// names, so a tick removes what the words promised and nothing else.
func TestTheRecordGoesOnlyWhenItIsAskedFor(t *testing.T) {
	base := t.TempDir()
	t.Setenv("APPDATA", base)
	t.Setenv("LOCALAPPDATA", base)

	record, err := RecordDir()
	if err != nil {
		t.Fatalf("RecordDir: %v", err)
	}
	file, err := RecordFile()
	if err != nil {
		t.Fatalf("RecordFile: %v", err)
	}
	if !holds(record, file) {
		t.Fatalf("the screen names %q while the uninstall would remove %q", file, record)
	}
	if err := os.MkdirAll(record, dirPerm); err != nil {
		t.Fatalf("making the record folder: %v", err)
	}
	if err := os.WriteFile(file, []byte("a website or two"), dirPerm); err != nil {
		t.Fatalf("writing the record: %v", err)
	}
	if err := RemoveTree(record); err != nil {
		t.Fatalf("RemoveTree: %v", err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("the record is still at %q after a removal that was asked for", file)
	}
}

// TestAFolderWithNowhereToLiveIsLeftOut keeps a missing environment variable
// from turning into a relative path. Joining an empty base gives "Visitron.exe"
// rather than a folder under the user's profile; removing that would take
// a folder of that name beside whatever the working directory happens to be.
func TestAFolderWithNowhereToLiveIsLeftOut(t *testing.T) {
	t.Setenv("APPDATA", "")
	t.Setenv("LOCALAPPDATA", filepath.Join(t.TempDir(), "Local"))

	folders := Leftovers()
	if len(folders) != 0 {
		t.Fatalf("Leftovers cleared %d folder(s) with no APPDATA, want none", len(folders))
	}
	for _, folder := range folders {
		if !filepath.IsAbs(folder) {
			t.Errorf("an uninstall would clear %q, which is not a place on the machine", folder)
		}
	}
}
