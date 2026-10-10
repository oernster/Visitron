// Package setup holds the per-user install policy behind the bespoke setup
// program: the paths and the payload extraction, which are portable and unit
// tested, plus the registry, shortcut and process work in the Windows files
// beside this one.
//
// Everything is per user. Nothing here needs administrator rights, so the whole
// flow runs without an elevation prompt (FR-080).
//
// Ported from ED Voyage Companion's internal/infrastructure/setup.
package setup

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/oernster/visitron/internal/infrastructure/store"
	"github.com/oernster/visitron/internal/product"
)

const (
	// AppName is the product name shown to the user, used for the Start Menu
	// entry and for the Apps list.
	AppName = product.Name
	// InstallFolder names the install directory and the registry key.
	InstallFolder = product.Name
	// ExeName is the installed application executable.
	ExeName = product.Name + ".exe"
	// Publisher is recorded in the uninstall registry entry.
	Publisher = product.Author

	installSubdir = "Programs"
	dirPerm       = 0o755
)

// UninstallFlag is the argument the Apps list passes back to setup, recorded in
// the registry as the UninstallString. It lives here rather than in the setup
// program because the registry writer and the program that reads it back are in
// different packages.
const UninstallFlag = "-uninstall"

// quotedPath wraps a path in plain double quotes, as every other entry in the
// Apps list does.
//
// It exists because the obvious %q ships a broken entry. %q is Go's quoting,
// which escapes the separators inside the string, so a Windows path reaches the
// registry with every backslash doubled and Windows then looks for a place that
// does not exist. Measured on the first real install: the UninstallString read
// C:\\Users\\Oliver\\... and neither Uninstall nor Modify in the Apps list
// could have worked.
func quotedPath(path string) string {
	return `"` + path + `"`
}

// InstallDir answers the per-user install directory,
// %LOCALAPPDATA%\Programs\Visitron. Installing there is what keeps the whole
// flow free of an administrator prompt.
func InstallDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", fmt.Errorf("LOCALAPPDATA is not set")
	}
	return filepath.Join(base, installSubdir, InstallFolder), nil
}

// RecordDir answers the folder holding Visitron's data: the websites, their
// history and the run log, %LOCALAPPDATA%\Visitron. It is the one thing an
// uninstall must not remove unless it is asked to in so many words (FR-080).
// The store says where that is, so the two can never name different folders.
func RecordDir() (string, error) {
	file, err := RecordFile()
	if err != nil {
		return "", err
	}
	return filepath.Dir(file), nil
}

// RecordFile answers the data file itself, so the uninstall screen can name
// the exact file it is offering to delete rather than a folder.
func RecordFile() (string, error) { return store.DefaultPath() }

// WebViewDir answers the folder WebView2 makes for the window. The application
// sets no user-data path, so WebView2 falls back to %APPDATA% joined with the
// executable's own file name, the .exe included. It holds no part of the
// record, so an uninstall clears it without asking.
func WebViewDir() (string, error) {
	base := os.Getenv("APPDATA")
	if base == "" {
		return "", fmt.Errorf("APPDATA is not set")
	}
	return filepath.Join(base, ExeName), nil
}

// Leftovers answers the folders an uninstall always clears: what the window
// kept for itself and the run log. Neither holds any part of the record.
//
// The record's own folder is deliberately NOT here, which is the whole of
// FR-080: removing the program is not a decision to throw away years of
// observations, so the record goes only when the user has ticked the box that
// names the file. A test asserts that no folder in this list is the record's
// folder or a parent of it, so a later change to any of these paths cannot
// take the record with it by accident.
//
// The install directory is not here either: setup is running from inside it,
// so it is scheduled for deletion after this process exits.
//
// A folder whose environment variable is missing is left out rather than
// guessed at. The machine cannot say where it would be; joining an empty
// base gives a relative path, which would delete a folder of that name beside
// whatever the working directory happens to be.
func Leftovers() []string {
	var folders []string
	for _, resolve := range []func() (string, error){WebViewDir} {
		if folder, err := resolve(); err == nil {
			folders = append(folders, folder)
		}
	}
	return folders
}

// ExtractZip extracts a zip archive into dest, creating directories as needed
// and refusing any entry whose path would escape dest.
func ExtractZip(data []byte, dest string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open payload: %w", err)
	}
	if err := os.MkdirAll(dest, dirPerm); err != nil {
		return fmt.Errorf("create install dir %q: %w", dest, err)
	}
	for _, file := range reader.File {
		if err := extractEntry(file, dest); err != nil {
			return err
		}
	}
	return nil
}

// extractEntry writes one archive entry, rejecting a name that climbs out of
// dest. Every entry is checked against the target before it is written, so a
// crafted archive cannot escape the install folder.
func extractEntry(file *zip.File, dest string) error {
	target := filepath.Join(dest, file.Name)
	fence := filepath.Clean(dest) + string(os.PathSeparator)
	if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), fence) {
		return fmt.Errorf("unsafe path in payload: %q", file.Name)
	}
	if file.FileInfo().IsDir() {
		return os.MkdirAll(target, dirPerm)
	}
	if err := os.MkdirAll(filepath.Dir(target), dirPerm); err != nil {
		return fmt.Errorf("create dir for %q: %w", target, err)
	}
	source, err := file.Open()
	if err != nil {
		return fmt.Errorf("open entry %q: %w", file.Name, err)
	}
	defer source.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, dirPerm)
	if err != nil {
		return fmt.Errorf("create %q: %w", target, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, source); err != nil {
		return fmt.Errorf("write %q: %w", target, err)
	}
	return nil
}

// DirSizeKB answers the total size of a directory tree in kilobytes, which is
// what the uninstall entry's EstimatedSize value wants.
func DirSizeKB(dir string) (uint32, error) {
	var total int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("size %q: %w", dir, err)
	}
	return uint32(total / 1024), nil
}

// RemoveTree deletes a directory tree.
func RemoveTree(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove %q: %w", dir, err)
	}
	return nil
}

// CopyFile copies one file, used to leave a copy of the setup program inside
// the install directory so the Apps list has an uninstaller to call.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %q: %w", src, err)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, dirPerm)
	if err != nil {
		return fmt.Errorf("create %q: %w", dst, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("write %q: %w", dst, err)
	}
	return nil
}
