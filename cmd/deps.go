package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
)

var (
	osLookPath  = exec.LookPath
	osRunCmd    = runCmdReal
	currentGOOS = runtime.GOOS
)

func toolExists(name string) bool {
	_, err := osLookPath(name)
	return err == nil
}

func runCmdReal(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func fallbackInstructions() string {
	return `Please install ffmpeg manually:

  macOS:   brew install ffmpeg
  Linux:   sudo apt-get install ffmpeg
  Windows: winget install --id Gyan.FFmpeg -e`
}

func ensureFFmpeg() error {
	if toolExists("ffmpeg") {
		return nil
	}

	switch currentGOOS {
	case "darwin":
		return installFFmpegDarwin()
	case "linux":
		return installFFmpegLinux()
	case "windows":
		return installFFmpegWindows()
	default:
		return fmt.Errorf("auto-install not supported on %s\n%s", currentGOOS, fallbackInstructions())
	}
}

func installFFmpegDarwin() error {
	if !toolExists("brew") {
		log.Printf("Homebrew not found — installing Homebrew")
		const brewScript = `curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh | NONINTERACTIVE=1 /bin/bash`
		if err := osRunCmd("/bin/sh", "-c", brewScript); err != nil {
			return fmt.Errorf("could not install Homebrew: %w\nInstall manually: https://brew.sh\n%s", err, fallbackInstructions())
		}
	}
	log.Printf("ffmpeg not found — installing via Homebrew")
	if err := osRunCmd("brew", "install", "ffmpeg"); err != nil {
		return fmt.Errorf("could not install ffmpeg: %w\n%s", err, fallbackInstructions())
	}
	return nil
}

func installFFmpegLinux() error {
	aptCmd := "apt-get"
	if !toolExists("apt-get") {
		aptCmd = "apt"
	}
	if !toolExists(aptCmd) {
		return fmt.Errorf("could not find apt-get or apt\n%s", fallbackInstructions())
	}
	log.Printf("ffmpeg not found — installing via %s", aptCmd)
	if err := osRunCmd(aptCmd, "install", "-y", "ffmpeg"); err != nil {
		return fmt.Errorf("could not install ffmpeg: %w\n%s", err, fallbackInstructions())
	}
	return nil
}

func installFFmpegWindows() error {
	if !toolExists("winget") {
		return fmt.Errorf("winget not found\n%s", fallbackInstructions())
	}
	log.Printf("ffmpeg not found — installing via winget")
	if err := osRunCmd("winget", "install", "--id", "Gyan.FFmpeg", "-e"); err != nil {
		return fmt.Errorf("could not install ffmpeg: %w\n%s", err, fallbackInstructions())
	}
	return nil
}
