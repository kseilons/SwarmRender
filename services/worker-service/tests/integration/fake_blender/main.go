package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Simple mock of blender output
	args := os.Args[1:]
	
	// Check if it's a metadata request
	isMetadata := false
	for i, arg := range args {
		if arg == "--python-expr" && strings.Contains(args[i+1], "bpy.context.scene.frame_start") {
			isMetadata = true
			break
		}
	}

	if isMetadata {
		metadata := map[string]int{
			"frame_start": 1,
			"frame_end":   100,
		}
		jsonStr, _ := json.Marshal(metadata)
		fmt.Printf("FRAME_INFO:%s\n", string(jsonStr))
		return
	}

	// If it's a render request
	startFrame := 1
	endFrame := 1
	outputPath := ""
	for i, arg := range args {
		if arg == "-s" {
			fmt.Sscanf(args[i+1], "%d", &startFrame)
		}
		if arg == "-e" {
			fmt.Sscanf(args[i+1], "%d", &endFrame)
		}
		if arg == "-f" {
			fmt.Sscanf(args[i+1], "%d", &startFrame)
			endFrame = startFrame
		}
		if arg == "-o" {
			outputPath = args[i+1]
		}
	}

	for f := startFrame; f <= endFrame; f++ {
		fileName := fmt.Sprintf("frame_%04d.png", f)
		if outputPath != "" {
			// outputPath might be a directory or a file prefix
			// Use simple logic: if it ends with path separator, it's a dir.
			// Actually Blender -o /path/to/frame_ means files will be /path/to/frame_0001.png
			
			fullPath := outputPath
			if strings.HasSuffix(outputPath, string(os.PathSeparator)) || strings.HasSuffix(outputPath, "/") {
				fullPath = filepath.Join(outputPath, fileName)
			} else {
				fullPath = outputPath + fmt.Sprintf("%04d.png", f)
			}
			
			os.WriteFile(fullPath, []byte("fake frame content"), 0644)
		}
		fmt.Printf("Fra:%d Mem:10.0M (0.00M, Peak 10.0M) | Time:00:01.00 | Remaining:00:00.00 | Saved: '%s'\n", f, fileName)
	}
}
