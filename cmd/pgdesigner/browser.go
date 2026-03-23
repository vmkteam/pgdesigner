package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		fmt.Printf("Open %s in your browser\n", url)
		return
	}
	_ = cmd.Start()
}

func openAppMode(url string) {
	browsers := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
	}

	switch runtime.GOOS {
	case "darwin":
		paths := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				cmd := exec.Command(p, "--app="+url)
				if err := cmd.Start(); err == nil {
					return
				}
			}
		}
	case "linux":
		for _, browser := range browsers {
			if path, err := exec.LookPath(browser); err == nil {
				cmd := exec.Command(path, "--app="+url)
				if err := cmd.Start(); err == nil {
					return
				}
			}
		}
	case "windows":
		paths := []string{
			os.Getenv("LOCALAPPDATA") + `\Google\Chrome\Application\chrome.exe`,
			os.Getenv("PROGRAMFILES") + `\Google\Chrome\Application\chrome.exe`,
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				cmd := exec.Command(p, "--app="+url)
				if err := cmd.Start(); err == nil {
					return
				}
			}
		}
	}

	fmt.Println("Chrome not found, falling back to default browser")
	openBrowser(url)
}
