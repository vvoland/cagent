package chat

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log/slog"
	"net/http"

	"golang.org/x/image/draw"
	// Register WebP decoder for image.Decode.
	_ "golang.org/x/image/webp"
)

const (
	// MaxImageDimension is the maximum width or height for images sent to LLM providers.
	MaxImageDimension = 2000
	// MaxImageBytes is the maximum file size for images sent to LLM providers (4.5MB,
	// below Anthropic's 5MB limit).
	MaxImageBytes = 4_500_000
	// jpegQuality is the default JPEG encoding quality.
	jpegQuality = 80
)

// DetectMimeTypeByContent detects the MIME type of data by inspecting its content
// (magic bytes). Returns the MIME type or "application/octet-stream" if unknown.
// This supplements extension-based detection for files with missing or wrong extensions.
func DetectMimeTypeByContent(data []byte) string {
	if len(data) == 0 {
		return "application/octet-stream"
	}

	// Check WebP first — http.DetectContentType doesn't recognise it.
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}

	ct := http.DetectContentType(data)

	// http.DetectContentType returns types like "image/jpeg", "image/png",
	// "image/gif", "application/pdf", etc. For our purposes we accept it as-is.
	return ct
}

// ImageResizeResult holds the output of an image resize operation.
type ImageResizeResult struct {
	// Data is the (possibly re-encoded) image bytes.
	Data []byte
	// MimeType is the MIME type of the output image.
	MimeType string
	// OriginalWidth and OriginalHeight are the dimensions of the input image.
	OriginalWidth, OriginalHeight int
	// Width and Height are the dimensions of the output image.
	Width, Height int
	// Resized indicates whether the image was actually modified.
	Resized bool
}

// ResizeImage takes raw image bytes and ensures they fit within provider limits
// (max 2000×2000 pixels, max 4.5 MB). If the image already fits, it is returned
// unchanged. Otherwise it is scaled down (preserving aspect ratio) and re-encoded.
//
// The function tries to produce the smallest output by comparing PNG and JPEG
// encoding, and progressively reducing JPEG quality and dimensions if needed.
func ResizeImage(data []byte, mimeType string) (*ImageResizeResult, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	origW, origH := bounds.Dx(), bounds.Dy()

	// If the image already fits within all limits, return unchanged.
	if origW <= MaxImageDimension && origH <= MaxImageDimension && len(data) <= MaxImageBytes {
		return &ImageResizeResult{
			Data:           data,
			MimeType:       mimeType,
			OriginalWidth:  origW,
			OriginalHeight: origH,
			Width:          origW,
			Height:         origH,
			Resized:        false,
		}, nil
	}

	// Scale down to fit within MaxImageDimension, preserving aspect ratio.
	newW, newH := fitDimensions(origW, origH, MaxImageDimension, MaxImageDimension)
	resized := scaleImage(img, newW, newH)

	// Try both PNG and JPEG at default quality, pick the smaller one.
	best, bestMime, err := pickSmallestEncoding(resized)
	if err != nil {
		return nil, fmt.Errorf("picking smallest encoding: %w", err)
	}

	// If still over the byte limit, try JPEG with decreasing quality.
	if len(best) > MaxImageBytes {
		for _, q := range []int{70, 55, 40} {
			encoded, err := encodeJPEG(resized, q)
			if err != nil {
				continue
			}

			if len(encoded) < len(best) {
				best = encoded
				bestMime = "image/jpeg"
			}
			if len(best) <= MaxImageBytes {
				break
			}
		}
	}

	// If still over, progressively reduce dimensions.
	if len(best) > MaxImageBytes {
		for _, scale := range []float64{0.75, 0.50, 0.35, 0.25} {
			scaledW := int(float64(newW) * scale)
			scaledH := int(float64(newH) * scale)
			if scaledW < 1 {
				scaledW = 1
			}
			if scaledH < 1 {
				scaledH = 1
			}
			smaller := scaleImage(img, scaledW, scaledH)
			for _, q := range []int{80, 55, 40} {
				encoded, err := encodeJPEG(smaller, q)
				if err != nil {
					continue
				}

				if len(encoded) < len(best) {
					best = encoded
					bestMime = "image/jpeg"
					newW, newH = scaledW, scaledH
				}
				if len(best) <= MaxImageBytes {
					break
				}
			}
			if len(best) <= MaxImageBytes {
				break
			}
		}
	}

	if len(best) > MaxImageBytes {
		slog.Warn("Image still exceeds size limit after all resize attempts",
			"original_size", len(data), "final_size", len(best), "limit", MaxImageBytes)
	}

	return &ImageResizeResult{
		Data:           best,
		MimeType:       bestMime,
		OriginalWidth:  origW,
		OriginalHeight: origH,
		Width:          newW,
		Height:         newH,
		Resized:        true,
	}, nil
}

// ResizeImageBase64 is a convenience wrapper around ResizeImage that accepts
// and returns base64-encoded image data.
func ResizeImageBase64(b64Data, mimeType string) (*ImageResizeResult, error) {
	raw, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}
	result, err := ResizeImage(raw, mimeType)
	if err != nil {
		return nil, err
	}
	result.Data = []byte(base64.StdEncoding.EncodeToString(result.Data))
	return result, nil
}

// FormatDimensionNote produces a human-readable note describing the resize mapping.
// This helps the model translate coordinates from the resized image back to the original.
func FormatDimensionNote(r *ImageResizeResult) string {
	if !r.Resized {
		return ""
	}
	scaleX := float64(r.OriginalWidth) / float64(r.Width)
	scaleY := float64(r.OriginalHeight) / float64(r.Height)
	scale := scaleX
	if scaleY > scaleX {
		scale = scaleY
	}
	return fmt.Sprintf("[Image: original %dx%d, displayed at %dx%d. Multiply coordinates by %.2f to map to original image.]",
		r.OriginalWidth, r.OriginalHeight, r.Width, r.Height, scale)
}

// fitDimensions returns new dimensions that fit within maxW×maxH while
// preserving the aspect ratio of w×h.
func fitDimensions(w, h, maxW, maxH int) (int, int) {
	if w <= maxW && h <= maxH {
		return w, h
	}
	ratioW := float64(maxW) / float64(w)
	ratioH := float64(maxH) / float64(h)
	ratio := ratioW
	if ratioH < ratioW {
		ratio = ratioH
	}
	newW := int(float64(w) * ratio)
	newH := int(float64(h) * ratio)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	return newW, newH
}

// scaleImage resizes img to the given dimensions using CatmullRom (bicubic) interpolation.
func scaleImage(img image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}

// pickSmallestEncoding encodes the image as both PNG and JPEG and returns whichever is smaller.
func pickSmallestEncoding(img image.Image) ([]byte, string, error) {
	pngData, errPNG := encodePNG(img)
	jpegData, errJPEG := encodeJPEG(img, jpegQuality)
	if errPNG != nil && errJPEG != nil {
		return nil, "", errors.Join(errPNG, errJPEG)
	}
	if errPNG != nil {
		return jpegData, "image/jpeg", nil
	}
	if errJPEG != nil {
		return pngData, "image/png", nil
	}

	if len(pngData) <= len(jpegData) {
		return pngData, "image/png", nil
	}
	return jpegData, "image/jpeg", nil
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		// Fallback: should not happen for RGBA images.
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
