package main

import (
	"fmt"
	"testing"
)

func TestProbe(t *testing.T) {
	fmt.Println("test starting")
	fmt.Println(getVideoAspectRatio("samples/boots-video-vertical.mp4"))
}
