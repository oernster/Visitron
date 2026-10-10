// Package runlog keeps the log a run leaves (NFR-OBS-001): a line naming when the
// run started, then what the run reports. Where that file sits is the
// platform's business and logPath below states each rule. Ported from Bridge
// Talk's runlog, trimmed to what SymDiary needs, then ported to Visitron.
//
// A windowed Windows program started from a shortcut or the Run key has no
// error output: Windows hands it a handle of 0, so the Go runtime's own crash
// report would reach nobody. Keep points the error output at the log before
// anything else runs, so a crash leaves a record. That is the one part of this
// package a platform does differently; it is the only thing behind a build
// tag: output_windows.go and output_other.go.
package runlog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"visitron/internal/product"
)

const (
	// FileName names the log inside the product's local data folder.
	FileName = product.LogFileName
	// MaxBytes is the size past which a run starts the log afresh, so the
	// file cannot grow without end: 1 MB, as Bridge Talk measured enough.
	MaxBytes = 1 << 20
)

const (
	stampLayout = "2006-01-02 15:04:05"
	folderPerm  = 0o755
	filePerm    = 0o644
	dataFolder  = "LOCALAPPDATA"
	stateFolder = "XDG_STATE_HOME"
)

// The platforms named by name, so the rule below reads as the rule rather than
// as three string comparisons.
const (
	windowsOS = "windows"
	macOS     = "darwin"
)

// The folders each platform keeps a program's own log in, below the home
// directory. Windows takes its base from the environment instead.
var (
	macLogFolder    = []string{"Library", "Logs"}
	xdgLogFolder    = []string{".local", "state"}
	errNoDataFolder = errors.New(dataFolder + " is not set")
)

// Path answers the log's path without touching the disk.
func Path() (string, error) {
	return logPath(runtime.GOOS, os.Getenv, os.UserHomeDir)
}

// logPath works the log's path out from the platform, the environment and the
// home directory. Each is a parameter rather than read here, so every
// platform's rule is exercised on every platform: a rule that only runs on the
// machine it describes is a rule nobody checks until a user reports it.
//
// The three answers differ because the platforms do, not by preference:
// Windows keeps a program's own data under LOCALAPPDATA; macOS keeps logs in
// Library/Logs, where Console shows them; everywhere else the XDG rule puts
// state that is not a cache and not configuration under XDG_STATE_HOME, else
// ~/.local/state. Inside a Flatpak that variable is already pointed at the
// sandbox, so the same line lands in the right place there too.
func logPath(goos string, getenv func(string) string,
	home func() (string, error)) (string, error) {
	if goos == windowsOS {
		base := getenv(dataFolder)
		if base == "" {
			return "", errNoDataFolder
		}
		return filepath.Join(base, product.Name, FileName), nil
	}
	if goos != macOS {
		if base := getenv(stateFolder); base != "" {
			return filepath.Join(base, product.Name, FileName), nil
		}
	}
	found, err := home()
	if err != nil {
		return "", fmt.Errorf("finding the home directory: %w", err)
	}
	folder := xdgLogFolder
	if goos == macOS {
		folder = macLogFolder
	}
	parts := append([]string{found}, folder...)
	return filepath.Join(append(parts, product.Name, FileName)...), nil
}

// Open opens the log at path for a run started at started, making its folder
// where there is none, then adds the run's start line. The log is kept unless
// it is over MaxBytes, when it is started afresh.
func Open(path string, started time.Time) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), folderPerm); err != nil {
		return nil, fmt.Errorf("making the folder for %s: %w", path, err)
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if info, err := os.Stat(path); err == nil && info.Size() > MaxBytes {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, filePerm)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	if _, err := fmt.Fprintf(file, "%s started %s\n", product.Name, started.Format(stampLayout)); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("writing to %s: %w", path, err)
	}
	return file, nil
}

// Keep sends what the run reports as it fails to log, which stays open for
// the rest of the run. Where the run has no error output, all of it goes to
// the log; otherwise it stays where it is and the crash report is copied to
// the log as well.
func Keep(log *os.File) error {
	if !hasErrorOutput() {
		return sendAll(log)
	}
	if err := debug.SetCrashOutput(log, debug.CrashOptions{}); err != nil {
		return fmt.Errorf("copying crash reports to %s: %w", log.Name(), err)
	}
	return nil
}

// hasErrorOutput and sendAll are the platform's own, in output_windows.go and
// output_other.go: they are the only part of the log that Windows does
// differently.

// Logger writes timestamped lines to the log.
type Logger struct {
	mu   sync.Mutex
	file *os.File
	now  func() time.Time
}

// NewLogger writes to file, stamping each line with now.
func NewLogger(file *os.File, now func() time.Time) *Logger {
	return &Logger{file: file, now: now}
}

// Printf writes one stamped line. A line that cannot be written is dropped:
// the log is where failures are reported, so there is nowhere else to say so.
func (l *Logger) Printf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = fmt.Fprintf(l.file, "%s  %s\n", l.now().Format(stampLayout), fmt.Sprintf(format, args...))
}
