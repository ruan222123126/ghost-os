package guiagent

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/png"
)

func VerifyAction(before Observation, after Observation, action Action) (VerificationResult, error) {
	if action.Type == ActionWait {
		return VerificationResult{
			VisibleEffect: true,
			Message:       "wait action does not require visible change",
		}, nil
	}
	screenshotChanged := hashBytes(before.ImageBytes) != hashBytes(after.ImageBytes)
	windowChanged := before.ActiveWindowTitle != after.ActiveWindowTitle ||
		before.ActiveWindowClass != after.ActiveWindowClass
	targetChanged, err := verifyTargetChange(before, after, action)
	if err != nil {
		return VerificationResult{}, &RunError{Code: ErrorVerificationFailed, Message: err.Error()}
	}
	visible := screenshotChanged || windowChanged || targetChanged
	message := "visible effect detected"
	if !visible {
		message = "previous step produced no visible effect"
	}
	return VerificationResult{
		VisibleEffect:     visible,
		ScreenshotChanged: screenshotChanged,
		WindowChanged:     windowChanged,
		TargetChanged:     targetChanged,
		Message:           message,
	}, nil
}

func verifyTargetChange(before Observation, after Observation, action Action) (bool, error) {
	target := action.Target
	if action.Type == ActionDrag && action.Destination != nil {
		target = action.Destination
	}
	if target == nil || target.Box == nil {
		return false, nil
	}
	beforeHash, err := cropHash(before.ImageBytes, before.ImageWidth, before.ImageHeight, *target.Box)
	if err != nil {
		return false, err
	}
	afterHash, err := cropHash(after.ImageBytes, after.ImageWidth, after.ImageHeight, *target.Box)
	if err != nil {
		return false, err
	}
	return beforeHash != afterHash, nil
}

func cropHash(imageBytes []byte, width int, height int, box NormalizedBox) (string, error) {
	img, err := png.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return "", fmt.Errorf("decode png: %w", err)
	}
	rect := normalizedBoxRect(width, height, box)
	cropped := cropImage(img, rect)
	if cropped == nil {
		return "", fmt.Errorf("crop image: invalid rectangle")
	}
	buf := bytes.Buffer{}
	if err := png.Encode(&buf, cropped); err != nil {
		return "", fmt.Errorf("encode cropped png: %w", err)
	}
	return hashBytes(buf.Bytes()), nil
}

func normalizedBoxRect(width int, height int, box NormalizedBox) image.Rectangle {
	x1 := clamp(int(box[0]*float64(width)), 0, width-1)
	y1 := clamp(int(box[1]*float64(height)), 0, height-1)
	x2 := clamp(int(box[2]*float64(width)), x1+1, width)
	y2 := clamp(int(box[3]*float64(height)), y1+1, height)
	return image.Rect(x1, y1, x2, y2)
}

func cropImage(src image.Image, rect image.Rectangle) image.Image {
	if rect.Empty() {
		return nil
	}
	out := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			out.Set(x-rect.Min.X, y-rect.Min.Y, src.At(x, y))
		}
	}
	return out
}

func hashBytes(buf []byte) string {
	sum := sha256.Sum256(buf)
	return fmt.Sprintf("%x", sum[:])
}

func clamp(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
