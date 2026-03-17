package images

import (
	"strings"
	"testing"
)

func TestParseOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    map[string]any
		want    TransformOptions
		wantErr string
	}{
		{
			name: "empty options",
			opts: map[string]any{},
			want: TransformOptions{},
		},
		{
			name: "width only",
			opts: map[string]any{"width": 300},
			want: TransformOptions{Width: 300},
		},
		{
			name: "height only",
			opts: map[string]any{"height": 200},
			want: TransformOptions{Height: 200},
		},
		{
			name: "width and height",
			opts: map[string]any{"width": 300, "height": 200},
			want: TransformOptions{Width: 300, Height: 200},
		},
		{
			name: "with crop",
			opts: map[string]any{"width": 300, "height": 200, "crop": "center"},
			want: TransformOptions{Width: 300, Height: 200, Crop: "center"},
		},
		{
			name: "with quality",
			opts: map[string]any{"quality": 85},
			want: TransformOptions{Quality: 85},
		},
		{
			name: "with format",
			opts: map[string]any{"format": "webp"},
			want: TransformOptions{Format: "webp"},
		},
		{
			name: "all options",
			opts: map[string]any{
				"width":   400,
				"height":  300,
				"crop":    "center",
				"quality": 90,
				"format":  "jpeg",
			},
			want: TransformOptions{
				Width:   400,
				Height:  300,
				Crop:    "center",
				Quality: 90,
				Format:  "jpeg",
			},
		},
		{
			name: "int64 values",
			opts: map[string]any{"width": int64(300), "height": int64(200)},
			want: TransformOptions{Width: 300, Height: 200},
		},
		{
			name: "float64 values",
			opts: map[string]any{"width": float64(300), "height": float64(200)},
			want: TransformOptions{Width: 300, Height: 200},
		},
		{
			name:    "invalid width type",
			opts:    map[string]any{"width": "300"},
			wantErr: "width: expected number",
		},
		{
			name:    "invalid height type",
			opts:    map[string]any{"height": "200"},
			wantErr: "height: expected number",
		},
		{
			name:    "invalid crop type",
			opts:    map[string]any{"crop": 123},
			wantErr: "crop: expected string",
		},
		{
			name:    "invalid quality type",
			opts:    map[string]any{"quality": "high"},
			wantErr: "quality: expected number",
		},
		{
			name:    "invalid format type",
			opts:    map[string]any{"format": 123},
			wantErr: "format: expected string",
		},
		{
			name:    "unknown option",
			opts:    map[string]any{"unknown": "value"},
			wantErr: "unknown option: unknown",
		},
		// Sharpen option tests
		{
			name: "sharpen false",
			opts: map[string]any{"sharpen": false},
			want: TransformOptions{SharpenDisabled: true},
		},
		{
			name: "sharpen true",
			opts: map[string]any{"sharpen": true},
			want: TransformOptions{}, // Both zero = use default
		},
		{
			name: "sharpen float",
			opts: map[string]any{"sharpen": 0.8},
			want: TransformOptions{Sharpen: 0.8},
		},
		{
			name: "sharpen int",
			opts: map[string]any{"sharpen": 1},
			want: TransformOptions{Sharpen: 1.0},
		},
		{
			name: "sharpen int64",
			opts: map[string]any{"sharpen": int64(2)},
			want: TransformOptions{Sharpen: 2.0},
		},
		{
			name:    "sharpen invalid type",
			opts:    map[string]any{"sharpen": "high"},
			wantErr: "sharpen: expected boolean or number",
		},
		// Scale option tests
		{
			name: "scale smart",
			opts: map[string]any{"scale": "smart", "width": 800},
			want: TransformOptions{Scale: "smart", Width: 800},
		},
		{
			name:    "scale invalid type",
			opts:    map[string]any{"scale": 123},
			wantErr: "scale: expected string",
		},
		// Focal option tests
		{
			name: "focal point",
			opts: map[string]any{"crop": "smart", "width": 400, "height": 300, "focal": map[string]any{"x": 0.5, "y": 0.5}},
			want: TransformOptions{Crop: "smart", Width: 400, Height: 300, Focal: &FocalRegion{X: 0.5, Y: 0.5}},
		},
		{
			name: "focal region",
			opts: map[string]any{"crop": "smart", "width": 400, "height": 300, "focal": map[string]any{"x": 0.2, "y": 0.3, "w": 0.4, "h": 0.5}},
			want: TransformOptions{Crop: "smart", Width: 400, Height: 300, Focal: &FocalRegion{X: 0.2, Y: 0.3, W: 0.4, H: 0.5}},
		},
		{
			name:    "focal missing x",
			opts:    map[string]any{"focal": map[string]any{"y": 0.5}},
			wantErr: "focal: missing required key 'x'",
		},
		{
			name:    "focal missing y",
			opts:    map[string]any{"focal": map[string]any{"x": 0.5}},
			wantErr: "focal: missing required key 'y'",
		},
		{
			name:    "focal invalid type",
			opts:    map[string]any{"focal": "center"},
			wantErr: "focal: expected dict",
		},
		// Smart crop option
		{
			name: "crop smart",
			opts: map[string]any{"crop": "smart", "width": 400, "height": 300},
			want: TransformOptions{Crop: "smart", Width: 400, Height: 300},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOptions(tt.opts)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("ParseOptions() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ParseOptions() error = %v, wantErr containing %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseOptions() unexpected error = %v", err)
				return
			}
			// Compare Focal separately since it's a pointer
			if tt.want.Focal != nil {
				if got.Focal == nil {
					t.Errorf("ParseOptions() Focal = nil, want %+v", tt.want.Focal)
					return
				}
				if *got.Focal != *tt.want.Focal {
					t.Errorf("ParseOptions() Focal = %+v, want %+v", *got.Focal, *tt.want.Focal)
					return
				}
				// Clear Focal for the main comparison
				got.Focal = nil
				tt.want.Focal = nil
			}
			if got != tt.want {
				t.Errorf("ParseOptions() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTransformOptions_Validate(t *testing.T) {
	tests := []struct {
		name      string
		opts      TransformOptions
		maxWidth  int
		maxHeight int
		wantErr   string
	}{
		{
			name:      "valid empty options",
			opts:      TransformOptions{},
			maxWidth:  4096,
			maxHeight: 4096,
		},
		{
			name:      "valid dimensions",
			opts:      TransformOptions{Width: 300, Height: 200},
			maxWidth:  4096,
			maxHeight: 4096,
		},
		{
			name:      "negative width",
			opts:      TransformOptions{Width: -1},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "width must be non-negative",
		},
		{
			name:      "negative height",
			opts:      TransformOptions{Height: -1},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "height must be non-negative",
		},
		{
			name:      "width exceeds max",
			opts:      TransformOptions{Width: 5000},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "width 5000 exceeds maximum 4096",
		},
		{
			name:      "height exceeds max",
			opts:      TransformOptions{Height: 5000},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "height 5000 exceeds maximum 4096",
		},
		{
			name:      "quality too low",
			opts:      TransformOptions{Quality: -1},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "quality must be 0-100",
		},
		{
			name:      "quality too high",
			opts:      TransformOptions{Quality: 101},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "quality must be 0-100",
		},
		{
			name:      "valid quality 0 (default)",
			opts:      TransformOptions{Quality: 0},
			maxWidth:  4096,
			maxHeight: 4096,
		},
		{
			name:      "valid quality 100",
			opts:      TransformOptions{Quality: 100},
			maxWidth:  4096,
			maxHeight: 4096,
		},
		{
			name:      "invalid crop value",
			opts:      TransformOptions{Crop: "left"},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "crop must be \"center\", \"smart\", or empty",
		},
		{
			name:      "valid crop center",
			opts:      TransformOptions{Crop: "center"},
			maxWidth:  4096,
			maxHeight: 4096,
		},
		{
			name:      "unsupported format",
			opts:      TransformOptions{Format: "bmp"},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "unsupported format",
		},
		{
			name:      "valid format jpeg",
			opts:      TransformOptions{Format: "jpeg"},
			maxWidth:  4096,
			maxHeight: 4096,
		},
		{
			name:      "valid format png",
			opts:      TransformOptions{Format: "png"},
			maxWidth:  4096,
			maxHeight: 4096,
		},
		{
			name:      "valid format webp",
			opts:      TransformOptions{Format: "webp"},
			maxWidth:  4096,
			maxHeight: 4096,
		},
		{
			name:      "valid format gif",
			opts:      TransformOptions{Format: "gif"},
			maxWidth:  4096,
			maxHeight: 4096,
		},
		{
			name:      "no max limits",
			opts:      TransformOptions{Width: 10000, Height: 10000},
			maxWidth:  0,
			maxHeight: 0,
		},
		// Smart crop validation tests
		{
			name:      "valid crop smart",
			opts:      TransformOptions{Crop: "smart", Width: 400, Height: 300},
			maxWidth:  4096,
			maxHeight: 4096,
		},
		{
			name:      "smart crop requires width",
			opts:      TransformOptions{Crop: "smart", Height: 300},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "crop: \"smart\" requires both width and height",
		},
		{
			name:      "smart crop requires height",
			opts:      TransformOptions{Crop: "smart", Width: 400},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "crop: \"smart\" requires both width and height",
		},
		// Scale validation tests
		{
			name:      "valid scale smart",
			opts:      TransformOptions{Scale: "smart", Width: 800},
			maxWidth:  4096,
			maxHeight: 4096,
		},
		{
			name:      "invalid scale value",
			opts:      TransformOptions{Scale: "seam"},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "scale must be \"smart\" or empty",
		},
		{
			name:      "scale smart requires dimension",
			opts:      TransformOptions{Scale: "smart"},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "scale: \"smart\" requires at least width or height",
		},
		{
			name:      "crop and scale mutually exclusive",
			opts:      TransformOptions{Crop: "smart", Scale: "smart", Width: 400, Height: 300},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "crop and scale cannot be used together",
		},
		// Focal validation tests
		{
			name:      "focal requires smart crop",
			opts:      TransformOptions{Focal: &FocalRegion{X: 0.5, Y: 0.5}},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "focal requires crop: \"smart\"",
		},
		{
			name:      "valid focal with smart crop",
			opts:      TransformOptions{Crop: "smart", Width: 400, Height: 300, Focal: &FocalRegion{X: 0.5, Y: 0.5}},
			maxWidth:  4096,
			maxHeight: 4096,
		},
		{
			name:      "focal x out of range",
			opts:      TransformOptions{Crop: "smart", Width: 400, Height: 300, Focal: &FocalRegion{X: 1.5, Y: 0.5}},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "focal x must be 0-1",
		},
		{
			name:      "focal y out of range",
			opts:      TransformOptions{Crop: "smart", Width: 400, Height: 300, Focal: &FocalRegion{X: 0.5, Y: -0.1}},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "focal y must be 0-1",
		},
		// Sharpen validation tests
		{
			name:      "valid sharpen sigma",
			opts:      TransformOptions{Sharpen: 0.5},
			maxWidth:  4096,
			maxHeight: 4096,
		},
		{
			name:      "negative sharpen",
			opts:      TransformOptions{Sharpen: -0.5},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "sharpen must be non-negative",
		},
		{
			name:      "sharpen disabled",
			opts:      TransformOptions{SharpenDisabled: true},
			maxWidth:  4096,
			maxHeight: 4096,
		},
		{
			name:      "sharpen set and disabled",
			opts:      TransformOptions{Sharpen: 0.5, SharpenDisabled: true},
			maxWidth:  4096,
			maxHeight: 4096,
			wantErr:   "sharpen cannot be both set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate(tt.maxWidth, tt.maxHeight)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("Validate() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Validate() error = %v, wantErr containing %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestTransformOptions_Canonical(t *testing.T) {
	tests := []struct {
		name string
		opts TransformOptions
		want string
	}{
		{
			name: "empty options",
			opts: TransformOptions{},
			want: "w=0|h=0|c=|q=0|f=|s=0|sc=",
		},
		{
			name: "with dimensions",
			opts: TransformOptions{Width: 300, Height: 200},
			want: "w=300|h=200|c=|q=0|f=|s=0|sc=",
		},
		{
			name: "with all options",
			opts: TransformOptions{
				Width:   400,
				Height:  300,
				Crop:    "center",
				Quality: 85,
				Format:  "JPEG",
			},
			want: "w=400|h=300|c=center|q=85|f=jpeg|s=0|sc=",
		},
		{
			name: "with sharpen disabled",
			opts: TransformOptions{Width: 300, SharpenDisabled: true},
			want: "w=300|h=0|c=|q=0|f=|s=off|sc=",
		},
		{
			name: "with explicit sharpen",
			opts: TransformOptions{Width: 300, Sharpen: 0.8},
			want: "w=300|h=0|c=|q=0|f=|s=0.80|sc=",
		},
		{
			name: "with smart crop",
			opts: TransformOptions{Width: 400, Height: 300, Crop: "smart"},
			want: "w=400|h=300|c=smart|q=0|f=|s=0|sc=",
		},
		{
			name: "with smart scale",
			opts: TransformOptions{Width: 800, Scale: "smart"},
			want: "w=800|h=0|c=|q=0|f=|s=0|sc=smart",
		},
		{
			name: "with focal point",
			opts: TransformOptions{Width: 400, Height: 300, Crop: "smart", Focal: &FocalRegion{X: 0.5, Y: 0.5}},
			want: "w=400|h=300|c=smart|q=0|f=|s=0|sc=|f=0.500,0.500",
		},
		{
			name: "with focal region",
			opts: TransformOptions{Width: 400, Height: 300, Crop: "smart", Focal: &FocalRegion{X: 0.2, Y: 0.3, W: 0.4, H: 0.5}},
			want: "w=400|h=300|c=smart|q=0|f=|s=0|sc=|f=0.200,0.300,0.400,0.500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opts.Canonical()
			if got != tt.want {
				t.Errorf("Canonical() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCacheKey(t *testing.T) {
	// Cache key should be deterministic
	opts := TransformOptions{Width: 300, Height: 200}
	key1 := CacheKey("abc123", opts)
	key2 := CacheKey("abc123", opts)

	if key1 != key2 {
		t.Errorf("CacheKey not deterministic: %q != %q", key1, key2)
	}

	// Cache key should be 16 chars
	if len(key1) != 16 {
		t.Errorf("CacheKey length = %d, want 16", len(key1))
	}

	// Different options should produce different keys
	opts2 := TransformOptions{Width: 400, Height: 200}
	key3 := CacheKey("abc123", opts2)
	if key1 == key3 {
		t.Errorf("Different options should produce different keys")
	}

	// Different source hashes should produce different keys
	key4 := CacheKey("def456", opts)
	if key1 == key4 {
		t.Errorf("Different source hashes should produce different keys")
	}

	// Sharpen state should produce different keys
	optsSharpened := TransformOptions{Width: 300, Height: 200, Sharpen: 0.5}
	keySharpened := CacheKey("abc123", optsSharpened)
	if key1 == keySharpened {
		t.Errorf("Sharpened options should produce different cache key")
	}

	optsNoSharpen := TransformOptions{Width: 300, Height: 200, SharpenDisabled: true}
	keyNoSharpen := CacheKey("abc123", optsNoSharpen)
	if key1 == keyNoSharpen {
		t.Errorf("SharpenDisabled options should produce different cache key")
	}

	if keySharpened == keyNoSharpen {
		t.Errorf("Sharpened vs disabled should produce different cache keys")
	}

	// Focal should produce different keys
	optsFocal := TransformOptions{Width: 300, Height: 200, Crop: "smart", Focal: &FocalRegion{X: 0.5, Y: 0.5}}
	keyFocal := CacheKey("abc123", optsFocal)

	optsFocal2 := TransformOptions{Width: 300, Height: 200, Crop: "smart", Focal: &FocalRegion{X: 0.3, Y: 0.7}}
	keyFocal2 := CacheKey("abc123", optsFocal2)

	if keyFocal == keyFocal2 {
		t.Errorf("Different focal points should produce different cache keys")
	}

	// Scale should produce different keys
	optsScale := TransformOptions{Width: 300, Scale: "smart"}
	keyScale := CacheKey("abc123", optsScale)

	optsNoScale := TransformOptions{Width: 300}
	keyNoScale := CacheKey("abc123", optsNoScale)

	if keyScale == keyNoScale {
		t.Errorf("Scale smart vs no scale should produce different cache keys")
	}
}

func TestNormalizeFormat(t *testing.T) {
	tests := []struct {
		input   string
		wantFmt string
		wantExt string
	}{
		{"jpeg", "jpeg", ".jpg"},
		{"JPEG", "jpeg", ".jpg"},
		{"jpg", "jpeg", ".jpg"},
		{"JPG", "jpeg", ".jpg"},
		{"png", "png", ".png"},
		{"PNG", "png", ".png"},
		{"webp", "webp", ".webp"},
		{"WEBP", "webp", ".webp"},
		{"gif", "gif", ".gif"},
		{"GIF", "gif", ".gif"},
		{"other", "other", ".other"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotFmt, gotExt := NormalizeFormat(tt.input)
			if gotFmt != tt.wantFmt || gotExt != tt.wantExt {
				t.Errorf("NormalizeFormat(%q) = (%q, %q), want (%q, %q)",
					tt.input, gotFmt, gotExt, tt.wantFmt, tt.wantExt)
			}
		})
	}
}

func TestExtensionToFormat(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{".jpg", "jpeg"},
		{".jpeg", "jpeg"},
		{".JPG", "jpeg"},
		{".JPEG", "jpeg"},
		{"jpg", "jpeg"},
		{"jpeg", "jpeg"},
		{".png", "png"},
		{".PNG", "png"},
		{".webp", "webp"},
		{".gif", "gif"},
		{".bmp", "bmp"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := ExtensionToFormat(tt.ext)
			if got != tt.want {
				t.Errorf("ExtensionToFormat(%q) = %q, want %q", tt.ext, got, tt.want)
			}
		})
	}
}

func TestDefaultQuality(t *testing.T) {
	tests := []struct {
		format string
		want   int
	}{
		{"jpeg", 85},
		{"jpg", 85},
		{"JPEG", 85},
		{"webp", 80},
		{"WEBP", 80},
		{"png", 0},
		{"gif", 0},
		{"unknown", 85},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			got := DefaultQuality(tt.format)
			if got != tt.want {
				t.Errorf("DefaultQuality(%q) = %d, want %d", tt.format, got, tt.want)
			}
		})
	}
}
