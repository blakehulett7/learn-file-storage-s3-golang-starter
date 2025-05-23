package main

import (
	"bytes"
	"os/exec"
)

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return outputPath, nil
}
