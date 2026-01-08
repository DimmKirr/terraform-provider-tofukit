package test

import (
	"testing"

	"github.com/tofukit/opentofu-provider-tofukit/internal/resources"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// =============================================================================
// TFK-13: SVG file routing tests
// =============================================================================

func TestSplitSVGFromRasterFiles(t *testing.T) {
	tests := []struct {
		name                string
		files               []schemas.FileModelWithPath
		expectedSVGCount    int
		expectedRasterCount int
		expectedSVGPaths    []string
		expectedRasterPaths []string
	}{
		{
			name: "only SVG files",
			files: []schemas.FileModelWithPath{
				{Path: "icon.svg"},
				{Path: "logo.SVG"},
				{Path: "path/to/graphic.svg"},
			},
			expectedSVGCount:    3,
			expectedRasterCount: 0,
			expectedSVGPaths:    []string{"icon.svg", "logo.SVG", "path/to/graphic.svg"},
			expectedRasterPaths: []string{},
		},
		{
			name: "only raster files",
			files: []schemas.FileModelWithPath{
				{Path: "photo.png"},
				{Path: "image.jpg"},
				{Path: "pic.jpeg"},
			},
			expectedSVGCount:    0,
			expectedRasterCount: 3,
			expectedSVGPaths:    []string{},
			expectedRasterPaths: []string{"photo.png", "image.jpg", "pic.jpeg"},
		},
		{
			name: "mixed SVG and raster files",
			files: []schemas.FileModelWithPath{
				{Path: "icon.svg"},
				{Path: "photo.png"},
				{Path: "logo.svg"},
				{Path: "banner.jpg"},
			},
			expectedSVGCount:    2,
			expectedRasterCount: 2,
			expectedSVGPaths:    []string{"icon.svg", "logo.svg"},
			expectedRasterPaths: []string{"photo.png", "banner.jpg"},
		},
		{
			name:                "empty list",
			files:               []schemas.FileModelWithPath{},
			expectedSVGCount:    0,
			expectedRasterCount: 0,
			expectedSVGPaths:    []string{},
			expectedRasterPaths: []string{},
		},
		{
			name: "uppercase extensions",
			files: []schemas.FileModelWithPath{
				{Path: "icon.SVG"},
				{Path: "photo.PNG"},
				{Path: "image.JPG"},
			},
			expectedSVGCount:    1,
			expectedRasterCount: 2,
			expectedSVGPaths:    []string{"icon.SVG"},
			expectedRasterPaths: []string{"photo.PNG", "image.JPG"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svgFiles, rasterFiles := resources.SplitSVGFromRasterFiles(tt.files)

			if len(svgFiles) != tt.expectedSVGCount {
				t.Errorf("Expected %d SVG files, got %d", tt.expectedSVGCount, len(svgFiles))
			}

			if len(rasterFiles) != tt.expectedRasterCount {
				t.Errorf("Expected %d raster files, got %d", tt.expectedRasterCount, len(rasterFiles))
			}

			// Verify SVG paths
			for i, expected := range tt.expectedSVGPaths {
				if i < len(svgFiles) && svgFiles[i].Path != expected {
					t.Errorf("Expected SVG path %s, got %s", expected, svgFiles[i].Path)
				}
			}

			// Verify raster paths
			for i, expected := range tt.expectedRasterPaths {
				if i < len(rasterFiles) && rasterFiles[i].Path != expected {
					t.Errorf("Expected raster path %s, got %s", expected, rasterFiles[i].Path)
				}
			}
		})
	}
}
