package main

import (
	"fmt"
	"testing"
)

func TestProbe(t *testing.T) {
	fmt.Println("test starting")
	getVideoAspectRatio("samples/boots-video-horizontal.mp4")
}
