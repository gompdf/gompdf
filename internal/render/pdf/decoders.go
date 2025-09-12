package pdf

// Register a broad set of image decoders so image.Decode can handle many formats.
// These are blank imports to hook into the init() of respective packages.
import (
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)
