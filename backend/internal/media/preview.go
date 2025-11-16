package media

import (
	"image"
	"image/jpeg"
	"os"

	_ "image/png"
)

// GenerateThumbnail attempts to create a JPEG thumbnail from an image file.
// For EXR/DPX this is a placeholder; integrate a proper reader in production.
func GenerateThumbnail(inputPath, outputPath string, maxW, maxH int) error {
	in, err := os.Open(inputPath)
	if err != nil { return err }
	defer in.Close()
	img, _, err := image.Decode(in)
	if err != nil { return err }
	// naive scale (no resample): keep as is for placeholder
	out, err := os.Create(outputPath)
	if err != nil { return err }
	defer out.Close()
	return jpeg.Encode(out, img, &jpeg.Options{Quality: 80})
}
