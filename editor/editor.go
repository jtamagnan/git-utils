package editor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/jtamagnan/git-utils/git"
)

func editor() string {
	coreEditor, _ := git.GetConfig("core.editor")
	if coreEditor != "" {
		return coreEditor
	} else if os.Getenv("GIT_EDITOR") != "" {
		return os.Getenv("GIT_EDITOR")
	} else if os.Getenv("EDITOR") != "" {
		return os.Getenv("EDITOR")
	} else if os.Getenv("VISUAL") != "" {
		return os.Getenv("VISUAL")
	}
	return "vim" // TODO(jat): Would be nice to have per operating system defaults
}


func openEditor(file string) (error) {
	cmd := exec.Command(editor(), file)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

func OpenEditor(initialContent string) (string, error) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "*.md") // TODO(jat): Would be nice to have a better default name
	if err != nil { return "", err }
	defer os.Remove(tmpFile.Name())  // Clean up the file after use
	defer tmpFile.Close()  // Close the file after use

	// Write the initial content to the file
	_, err = tmpFile.WriteString(initialContent)
	if err != nil { return "", err }

	// Open the file in the editor
	err = openEditor(tmpFile.Name())
	if err != nil { return "", err }

	// Read the content of the file after editing
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil { return "", err }

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
	if err != nil { return err }
	return nil
}
