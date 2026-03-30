package ghostty

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type FocusResult string

const (
	FocusResultFocused     FocusResult = "focused"
	FocusResultMissing     FocusResult = "missing"
	FocusResultAmbiguous   FocusResult = "ambiguous"
	FocusResultUnavailable FocusResult = "unavailable"
)

type RuntimeDiagnostics struct {
	PID         int    `json:"pid"`
	PPID        int    `json:"ppid"`
	TTY         string `json:"tty,omitempty"`
	StdinIsTTY  bool   `json:"stdin_is_tty"`
	StdoutIsTTY bool   `json:"stdout_is_tty"`
	StderrIsTTY bool   `json:"stderr_is_tty"`
	TERM        string `json:"term,omitempty"`
	TermProgram string `json:"term_program,omitempty"`
	Tmux        string `json:"tmux,omitempty"`
}

func DetectFocusedTerminalID() string {
	out, err := exec.Command("osascript", "-e", `
tell application "Ghostty"
	set w to front window
	set t to selected tab of w
	set term to focused terminal of t
	return id of term
end tell
`).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func FocusTerminalByID(termID string) (FocusResult, error) {
	if termID == "" {
		return FocusResultMissing, nil
	}

	out, err := exec.Command("osascript", "-e", fmt.Sprintf(`
tell application "Ghostty"
	set targetID to %q
	set matches to every terminal whose id is targetID
	set matchCount to count of matches
	if matchCount is 1 then
		focus item 1 of matches
		return "focused"
	else if matchCount is 0 then
		return "missing"
	else
		return "ambiguous"
	end if
end tell
`, termID)).Output()
	if err != nil {
		return FocusResultUnavailable, err
	}

	switch strings.TrimSpace(string(out)) {
	case string(FocusResultFocused):
		return FocusResultFocused, nil
	case string(FocusResultMissing):
		return FocusResultMissing, nil
	case string(FocusResultAmbiguous):
		return FocusResultAmbiguous, nil
	default:
		return FocusResultUnavailable, nil
	}
}

func CollectRuntimeDiagnostics() RuntimeDiagnostics {
	return RuntimeDiagnostics{
		PID:         os.Getpid(),
		PPID:        os.Getppid(),
		TTY:         detectTTY(),
		StdinIsTTY:  isatty(os.Stdin),
		StdoutIsTTY: isatty(os.Stdout),
		StderrIsTTY: isatty(os.Stderr),
		TERM:        os.Getenv("TERM"),
		TermProgram: os.Getenv("TERM_PROGRAM"),
		Tmux:        os.Getenv("TMUX"),
	}
}

func AppendBindingLog(path string, record any) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func Timestamp() string {
	return time.Now().Format(time.RFC3339Nano)
}

func detectTTY() string {
	out, err := exec.Command("tty").Output()
	if err != nil {
		return ""
	}
	tty := strings.TrimSpace(string(out))
	if tty == "not a tty" {
		return ""
	}
	return tty
}

func isatty(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
