package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jtamagnan/git-utils/git"
)

// validateEditor checks if the editor command is safe to execute
func validateEditor(editorCmd string) string {
	// Split the command to get just the executable name
	parts := strings.Fields(editorCmd)
	if len(parts) == 0 {
		return ""
	}

	executable := parts[0]

	// Check if it's an absolute path
	if filepath.IsAbs(executable) {
		// For absolute paths, ensure the file exists and is executable
		if info, err := os.Stat(executable); err == nil && !info.IsDir() {
			return editorCmd
		}
		return ""
	}

	// For relative paths, check if the command exists in PATH
	if _, err := exec.LookPath(executable); err == nil {
		return editorCmd
	}

	return ""
}

func editor() string {
	coreEditor, _ := git.GetConfig("core.editor")
	if coreEditor != "" {
		if validated := validateEditor(coreEditor); validated != "" {
			return validated
		}
	}

	if gitEditor := os.Getenv("GIT_EDITOR"); gitEditor != "" {
		if validated := validateEditor(gitEditor); validated != "" {
			return validated
		}
	}

	if envEditor := os.Getenv("EDITOR"); envEditor != "" {
		if validated := validateEditor(envEditor); validated != "" {
			return validated
		}
	}

	if visual := os.Getenv("VISUAL"); visual != "" {
		if validated := validateEditor(visual); validated != "" {
			return validated
		}
	}

	return "vim" // TODO(jat): Would be nice to have per operating system defaults
}

func openEditor(file string) error {
	editorCmd := editor()
	if editorCmd == "" {
		return fmt.Errorf("no valid editor found")
	}

	// Split the command to handle editors with arguments (e.g., "code --wait")
	parts := strings.Fields(editorCmd)
	if len(parts) == 0 {
		return fmt.Errorf("invalid editor command")
	}

	// Build command with file as the last argument
	executable := parts[0]
	args := parts[1:]
	args = append(args, file)
	// #nosec G204 - executable is validated by validateEditor function
	cmd := exec.Command(executable, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start editor: %v", err)
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("editor exited with error: %v", err)
	}

	return nil
}

func OpenEditor(initialContent string) (string, error) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "*.md") // TODO(jat): Would be nice to have a better default name
	if err != nil {
		return "", err
	}
	// defer os.Remove(tmpFile.Name())  // Don't clean up the file after use in case we want it
	defer func() {
		if closeErr := tmpFile.Close(); closeErr != nil {
			// Log the error, but don't override the main function's error
			fmt.Fprintf(os.Stderr, "Warning: failed to close temporary file: %v\n", closeErr)
		}
	}()

	// Write the initial content to the file
	_, err = tmpFile.WriteString(initialContent)
	if err != nil {
		return "", err
	}

	// Open the file in the editor
	err = openEditor(tmpFile.Name())
	if err != nil {
		return "", err
	}

	// Read the content of the file after editing
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}

	// Return the content as a string
	return string(content), nil
}

func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
