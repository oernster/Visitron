package setup

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// zipped answers an archive holding the named entries with the given contents.
func zipped(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, body := range entries {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatalf("creating %q: %v", name, err)
		}
		if _, err := file.Write([]byte(body)); err != nil {
			t.Fatalf("writing %q: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing the archive: %v", err)
	}
	return buffer.Bytes()
}

func TestExtractWritesEveryEntry(t *testing.T) {
	t.Parallel()
	dest := filepath.Join(t.TempDir(), "Programs", "Visitron")
	payload := zipped(t, map[string]string{
		"Visitron.exe":    "the application",
		"docs/README.txt": "a note",
	})
	if err := ExtractZip(payload, dest); err != nil {
		t.Fatalf("ExtractZip: %v", err)
	}
	for name, want := range map[string]string{
		"Visitron.exe":    "the application",
		"docs/README.txt": "a note",
	} {
		got, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(name)))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want %q", name, got, err, want)
		}
	}
}

// TestExtractRefusesAnEntryThatClimbsOut proves the fence: an archive entry is
// checked against the install folder before a single byte of it is written.
func TestExtractRefusesAnEntryThatClimbsOut(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	dest := filepath.Join(folder, "install")
	payload := zipped(t, map[string]string{"../escaped.txt": "out"})
	err := ExtractZip(payload, dest)
	if err == nil || !strings.Contains(err.Error(), "unsafe path") {
		t.Fatalf("ExtractZip = %v, want a refusal", err)
	}
	if _, err := os.Stat(filepath.Join(folder, "escaped.txt")); err == nil {
		t.Error("the entry was written outside the install folder")
	}
}

func TestExtractRefusesRubbish(t *testing.T) {
	t.Parallel()
	if err := ExtractZip([]byte("not a zip"), t.TempDir()); err == nil {
		t.Error("a payload that is not an archive was accepted")
	}
}

func TestSizeAndCopyAndRemove(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	source := filepath.Join(folder, "source.bin")
	if err := os.WriteFile(source, bytes.Repeat([]byte("x"), 4096), 0o600); err != nil {
		t.Fatal(err)
	}
	if size, err := DirSizeKB(folder); err != nil || size != 4 {
		t.Errorf("DirSizeKB = %d, %v; want 4", size, err)
	}
	if _, err := DirSizeKB(filepath.Join(folder, "absent")); err == nil {
		t.Error("sizing a missing folder was accepted")
	}

	target := filepath.Join(folder, "copy.bin")
	if err := CopyFile(source, target); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	if got, err := os.ReadFile(target); err != nil || len(got) != 4096 {
		t.Errorf("the copy holds %d bytes, %v", len(got), err)
	}
	if err := CopyFile(filepath.Join(folder, "absent"), target); err == nil {
		t.Error("copying a missing file was accepted")
	}
	if err := CopyFile(source, filepath.Join(folder, "absent", "copy.bin")); err == nil {
		t.Error("copying into a missing folder was accepted")
	}

	if err := RemoveTree(folder); err != nil {
		t.Fatalf("RemoveTree: %v", err)
	}
	if _, err := os.Stat(folder); err == nil {
		t.Error("the tree is still there")
	}
}

// TestQuotedWritesThePathAsWindowsDoes proves the registry value is a plainly
// quoted path.
//
// Go's %q would escape every separator, which is what the first real install
// wrote: the Apps list then held C:\\Users\\... for both Uninstall and Modify,
// naming a place that does not exist.
func TestQuotedWritesThePathAsWindowsDoes(t *testing.T) {
	t.Parallel()
	path := `C:\Users\Oliver\AppData\Local\Programs\Visitron\uninstall.exe`
	got := quotedPath(path)
	if want := `"` + path + `"`; got != want {
		t.Errorf("quotedPath = %s, want %s", got, want)
	}
	if strings.Contains(got, `\\`) {
		t.Errorf("the value carries doubled separators: %s", got)
	}
}

// TestThePathsAreWhereTheyAreSaid holds each path to what the documents and the
// uninstall screen promise.
func TestThePathsAreWhereTheyAreSaid(t *testing.T) {
	local := t.TempDir()
	roaming := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	t.Setenv("APPDATA", roaming)

	cases := map[string]struct {
		answer func() (string, error)
		want   string
	}{
		"install":  {InstallDir, filepath.Join(local, "Programs", "Visitron")},
		"record":   {RecordDir, filepath.Join(local, "Visitron")},
		"file":     {RecordFile, filepath.Join(local, "Visitron", "visitron.db")},
		"webview":  {WebViewDir, filepath.Join(roaming, "Visitron.exe")},
		"startdir": {StartMenuProgramsDir, filepath.Join(roaming, "Microsoft", "Windows", "Start Menu", "Programs")},
	}
	for name, c := range cases {
		got, err := c.answer()
		if err != nil || got != c.want {
			t.Errorf("%s = %q, %v; want %q", name, got, err, c.want)
		}
	}

	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("APPDATA", "")
	for name, answer := range map[string]func() (string, error){
		"install": InstallDir, "record": RecordDir,
		"file": RecordFile, "webview": WebViewDir,
	} {
		if _, err := answer(); err == nil {
			t.Errorf("%s answered a path with nothing set", name)
		}
	}
}
