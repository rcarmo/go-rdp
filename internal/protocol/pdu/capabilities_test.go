package pdu

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_GeneralCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeGeneral,
		GeneralCapabilitySet: &GeneralCapabilitySet{
			OSMajorType: 1,
			OSMinorType: 3,
			ExtraFlags:  0x041d,
		},
	}

	expected := []byte{
		0x01, 0x00, 0x18, 0x00, 0x01, 0x00, 0x03, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x1d, 0x04,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_GeneralCapabilitySet_Serialize2(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeGeneral,
		GeneralCapabilitySet: &GeneralCapabilitySet{
			OSMajorType: 1,
			OSMinorType: 3,
			ExtraFlags:  0x0415,
		},
	}

	expected, err := hex.DecodeString("010018000100030000020000000015040000000000000000")
	require.NoError(t, err)

	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_BitmapCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeBitmap,
		BitmapCapabilitySet: &BitmapCapabilitySet{
			PreferredBitsPerPixel: 0x18,
			Receive1BitPerPixel:   1,
			Receive4BitsPerPixel:  1,
			Receive8BitsPerPixel:  1,
			DesktopWidth:          1280,
			DesktopHeight:         1024,
			DesktopResizeFlag:     1,
		},
	}

	expected := []byte{
		0x02, 0x00, 0x1c, 0x00, 0x18, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x05, 0x00, 0x04,
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
	}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_BitmapCapabilitySet_Serialize2(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeBitmap,
		BitmapCapabilitySet: &BitmapCapabilitySet{
			PreferredBitsPerPixel: 0x18,
			Receive1BitPerPixel:   1,
			Receive4BitsPerPixel:  1,
			Receive8BitsPerPixel:  1,
			DesktopWidth:          1280,
			DesktopHeight:         800,
			DesktopResizeFlag:     0,
		},
	}

	expected, err := hex.DecodeString("02001c00180001000100010000052003000000000100000001000000")
	require.NoError(t, err)

	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_OrderCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeOrder,
		OrderCapabilitySet: &OrderCapabilitySet{
			OrderFlags: 0x002a,
			OrderSupport: [32]byte{
				0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			textFlags:        0x06a1,
			DesktopSaveSize:  0x38400,
			textANSICodePage: 0x04e4,
		},
	}

	expected := []byte{
		0x03, 0x00, 0x58, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x14, 0x00, 0x00, 0x00, 0x01, 0x00,
		0x00, 0x00, 0x2a, 0x00, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x01, 0x01, 0x01, 0x00,
		0x00, 0x00, 0x00, 0x00, 0xa1, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x84, 0x03, 0x00,
		0x00, 0x00, 0x00, 0x00, 0xe4, 0x04, 0x00, 0x00,
	}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_OrderCapabilitySet_Serialize2(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeOrder,
		OrderCapabilitySet: &OrderCapabilitySet{
			OrderFlags:       0xa,
			OrderSupport:     [32]byte{},
			textFlags:        0,
			DesktopSaveSize:  0x38400,
			textANSICodePage: 0,
		},
	}

	expected, err := hex.DecodeString("030058000000000000000000000000000000000000000000010014000000010000000a0000000000000000000000000000000000000000000000000000000000000000000000000000000000008403000000000000000000")
	require.NoError(t, err)

	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_BitmapCacheCapabilitySetRev1(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType:            CapabilitySetTypeBitmapCache,
		BitmapCacheCapabilitySetRev1: &BitmapCacheCapabilitySetRev1{},
	}

	expected, err := hex.DecodeString("04002800000000000000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)

	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_BitmapCacheCapabilitySetRev2(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeBitmapCacheRev2,
		BitmapCacheCapabilitySetRev2: &BitmapCacheCapabilitySetRev2{
			CacheFlags:           0x0003,
			NumCellCaches:        3,
			BitmapCache0CellInfo: 0x00000078,
			BitmapCache1CellInfo: 0x00000078,
			BitmapCache2CellInfo: 0x800009fb,
		},
	}

	expected := []byte{
		0x13, 0x00, 0x28, 0x00, 0x03, 0x00, 0x00, 0x03, 0x78, 0x00, 0x00, 0x00, 0x78, 0x00, 0x00, 0x00,
		0xfb, 0x09, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_ColorCacheCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeColorCache,
		ColorCacheCapabilitySet: &ColorCacheCapabilitySet{
			ColorTableCacheSize: 6,
		},
	}

	expected := []byte{
		0x0a, 0x00, 0x08, 0x00, 0x06, 0x00, 0x00, 0x00,
	}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_WindowActivationCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType:             CapabilitySetTypeActivation,
		WindowActivationCapabilitySet: &WindowActivationCapabilitySet{},
	}

	expected := []byte{
		0x07, 0x00, 0x0c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_ControlCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType:    CapabilitySetTypeControl,
		ControlCapabilitySet: &ControlCapabilitySet{},
	}

	expected := []byte{
		0x05, 0x00, 0x0c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x02, 0x00,
	}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_PointerCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypePointer,
		PointerCapabilitySet: &PointerCapabilitySet{
			ColorPointerFlag:      1,
			ColorPointerCacheSize: 20,
			PointerCacheSize:      21,
		},
	}

	expected := []byte{
		0x08, 0x00, 0x0a, 0x00, 0x01, 0x00, 0x14, 0x00, 0x15, 0x00,
	}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_ShareCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType:  CapabilitySetTypeShare,
		ShareCapabilitySet: &ShareCapabilitySet{},
	}

	expected := []byte{
		0x09, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_InputCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeInput,
		InputCapabilitySet: &InputCapabilitySet{
			InputFlags:          0x0015,
			KeyboardLayout:      0x00000409,
			KeyboardType:        4,
			KeyboardFunctionKey: 12,
		},
	}

	expected := []byte{
		0x0d, 0x00, 0x58, 0x00, 0x15, 0x00, 0x00, 0x00, 0x09, 0x04, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x0c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_InputCapabilitySet_Serialize2(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeInput,
		InputCapabilitySet: &InputCapabilitySet{
			InputFlags:          0x0015,
			KeyboardLayout:      0x00000409,
			KeyboardType:        4,
			KeyboardFunctionKey: 0,
		},
	}

	expected, err := hex.DecodeString("0d005800150000000904000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)

	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_SoundCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeSound,
		SoundCapabilitySet: &SoundCapabilitySet{
			SoundFlags: 0x0001,
		},
	}

	expected := []byte{0x0c, 0x00, 0x08, 0x00, 0x01, 0x00, 0x00, 0x00}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_SoundCapabilitySet_Serialize2(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeSound,
		SoundCapabilitySet: &SoundCapabilitySet{
			SoundFlags: 0,
		},
	}

	expected, err := hex.DecodeString("0c00080000000000")
	require.NoError(t, err)

	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_FontCapabilitySet(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeFont,
		FontCapabilitySet: &FontCapabilitySet{
			fontSupportFlags: 0x0001,
		},
	}

	expected := []byte{0x0e, 0x00, 0x08, 0x00, 0x01, 0x00, 0x00, 0x00}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_GlyphCacheCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeGlyphCache,
		GlyphCacheCapabilitySet: &GlyphCacheCapabilitySet{
			GlyphCache: [10]CacheDefinition{
				{
					CacheEntries:         254,
					CacheMaximumCellSize: 4,
				},
				{
					CacheEntries:         254,
					CacheMaximumCellSize: 4,
				},
				{
					CacheEntries:         254,
					CacheMaximumCellSize: 8,
				},
				{
					CacheEntries:         254,
					CacheMaximumCellSize: 8,
				},
				{
					CacheEntries:         254,
					CacheMaximumCellSize: 16,
				},
				{
					CacheEntries:         254,
					CacheMaximumCellSize: 32,
				},
				{
					CacheEntries:         254,
					CacheMaximumCellSize: 64,
				},
				{
					CacheEntries:         254,
					CacheMaximumCellSize: 128,
				},
				{
					CacheEntries:         254,
					CacheMaximumCellSize: 256,
				},
				{
					CacheEntries:         64,
					CacheMaximumCellSize: 256,
				},
			},
			FragCache:         0x1000100,
			GlyphSupportLevel: 3,
		},
	}

	expected := []byte{
		0x10, 0x00, 0x34, 0x00, 0xfe, 0x00, 0x04, 0x00, 0xfe, 0x00, 0x04, 0x00, 0xfe, 0x00, 0x08, 0x00,
		0xfe, 0x00, 0x08, 0x00, 0xfe, 0x00, 0x10, 0x00, 0xfe, 0x00, 0x20, 0x00, 0xfe, 0x00, 0x40, 0x00,
		0xfe, 0x00, 0x80, 0x00, 0xfe, 0x00, 0x00, 0x01, 0x40, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01,
		0x03, 0x00, 0x00, 0x00,
	}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_GlyphCacheCapabilitySet_Serialize2(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeGlyphCache,
		GlyphCacheCapabilitySet: &GlyphCacheCapabilitySet{
			FragCache:         0,
			GlyphSupportLevel: 0,
		},
	}

	expected, err := hex.DecodeString("10003400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)

	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_BrushCapabilitySet(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeBrush,
		BrushCapabilitySet: &BrushCapabilitySet{
			BrushSupportLevel: 1,
		},
	}

	expected := []byte{0x0f, 0x00, 0x08, 0x00, 0x01, 0x00, 0x00, 0x00}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_BrushCapabilitySet2(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeBrush,
		BrushCapabilitySet: &BrushCapabilitySet{
			BrushSupportLevel: 0,
		},
	}

	expected, err := hex.DecodeString("0f00080000000000")
	require.NoError(t, err)

	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_OffscreenBitmapCacheCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeOffscreenBitmapCache,
		OffscreenBitmapCacheCapabilitySet: &OffscreenBitmapCacheCapabilitySet{
			OffscreenSupportLevel: 1,
			OffscreenCacheSize:    7680,
			OffscreenCacheEntries: 100,
		},
	}

	expected := []byte{0x11, 0x00, 0x0c, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x1e, 0x64, 0x00}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_OffscreenBitmapCacheCapabilitySet_Serialize2(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeOffscreenBitmapCache,
		OffscreenBitmapCacheCapabilitySet: &OffscreenBitmapCacheCapabilitySet{
			OffscreenSupportLevel: 0,
			OffscreenCacheSize:    0,
			OffscreenCacheEntries: 0,
		},
	}

	expected, err := hex.DecodeString("11000c000000000000000000")
	require.NoError(t, err)

	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_VirtualChannelCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeVirtualChannel,
		VirtualChannelCapabilitySet: &VirtualChannelCapabilitySet{
			Flags: 0x00000001,
		},
	}

	expected := []byte{0x14, 0x00, 0x0c, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_VirtualChannelCapabilitySet_Serialize2(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeVirtualChannel,
		VirtualChannelCapabilitySet: &VirtualChannelCapabilitySet{
			Flags: 0,
		},
	}

	expected, err := hex.DecodeString("14000c000000000000000000")
	require.NoError(t, err)

	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_DrawNineGridCacheCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeDrawNineGridCache,
		DrawNineGridCacheCapabilitySet: &DrawNineGridCacheCapabilitySet{
			drawNineGridSupportLevel: 2,
			drawNineGridCacheSize:    2560,
			drawNineGridCacheEntries: 256,
		},
	}

	expected := []byte{0x15, 0x00, 0x0c, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x0a, 0x00, 0x01}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

func Test_DrawGDIPlusCapabilitySet_Serialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType:        CapabilitySetTypeDrawGDIPlus,
		DrawGDIPlusCapabilitySet: &DrawGDIPlusCapabilitySet{},
	}

	expected := []byte{
		0x16, 0x00, 0x28, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	actual := set.Serialize()

	require.Equal(t, expected, actual)
}

// Deserialize tests for all capability sets

func Test_GeneralCapabilitySet_Deserialize(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected GeneralCapabilitySet
	}{
		{
			name: "Standard",
			data: []byte{
				0x01, 0x00, 0x03, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x1d, 0x04,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01,
			},
			expected: GeneralCapabilitySet{
				OSMajorType:           1,
				OSMinorType:           3,
				ExtraFlags:            0x041d,
				RefreshRectSupport:    1,
				SuppressOutputSupport: 1,
			},
		},
		{
			name: "Windows10",
			data: []byte{
				0x0A, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x85, 0x05,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01,
			},
			expected: GeneralCapabilitySet{
				OSMajorType:           0x000A,
				OSMinorType:           0x0000,
				ExtraFlags:            0x0585,
				RefreshRectSupport:    1,
				SuppressOutputSupport: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var set GeneralCapabilitySet
			err := set.Deserialize(bytes.NewReader(tt.data))
			require.NoError(t, err)
			require.Equal(t, tt.expected, set)
		})
	}
}

func Test_BitmapCapabilitySet_Deserialize(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected BitmapCapabilitySet
	}{
		{
			name: "1280x1024",
			data: []byte{
				0x18, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x05, 0x00, 0x04,
				0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
			},
			expected: BitmapCapabilitySet{
				PreferredBitsPerPixel: 0x18,
				Receive1BitPerPixel:   1,
				Receive4BitsPerPixel:  1,
				Receive8BitsPerPixel:  1,
				DesktopWidth:          1280,
				DesktopHeight:         1024,
				DesktopResizeFlag:     1,
				DrawingFlags:          0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var set BitmapCapabilitySet
			err := set.Deserialize(bytes.NewReader(tt.data))
			require.NoError(t, err)
			require.Equal(t, tt.expected, set)
		})
	}
}

func Test_OrderCapabilitySet_Deserialize(t *testing.T) {
	set := CapabilitySet{
		CapabilitySetType: CapabilitySetTypeOrder,
		OrderCapabilitySet: &OrderCapabilitySet{
			OrderFlags: 0x002a,
			OrderSupport: [32]byte{
				0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			textFlags:        0x06a1,
			DesktopSaveSize:  0x38400,
			textANSICodePage: 0x04e4,
		},
	}

	serialized := set.Serialize()
	// Skip header (4 bytes)
	data := serialized[4:]

	var deserialized OrderCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, set.OrderCapabilitySet.OrderFlags, deserialized.OrderFlags)
	require.Equal(t, set.OrderCapabilitySet.OrderSupport, deserialized.OrderSupport)
	require.Equal(t, set.OrderCapabilitySet.DesktopSaveSize, deserialized.DesktopSaveSize)
}

func Test_BitmapCacheCapabilitySetRev1_Deserialize(t *testing.T) {
	set := BitmapCacheCapabilitySetRev1{
		Cache0Entries:         120,
		Cache0MaximumCellSize: 256,
		Cache1Entries:         120,
		Cache1MaximumCellSize: 1024,
		Cache2Entries:         240,
		Cache2MaximumCellSize: 4096,
	}
	serialized := set.Serialize()

	var deserialized BitmapCacheCapabilitySetRev1
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, set, deserialized)
}

func Test_BitmapCacheCapabilitySetRev2_Deserialize(t *testing.T) {
	set := BitmapCacheCapabilitySetRev2{
		CacheFlags:           0x0003,
		NumCellCaches:        3,
		BitmapCache0CellInfo: 0x00000078,
		BitmapCache1CellInfo: 0x00000078,
		BitmapCache2CellInfo: 0x800009fb,
	}
	serialized := set.Serialize()
	// Note: Deserialize reads BitmapCache4CellInfo twice (bug in source), so we need extra padding
	serialized = append(serialized, make([]byte, 4)...)

	var deserialized BitmapCacheCapabilitySetRev2
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, set.CacheFlags, deserialized.CacheFlags)
	require.Equal(t, set.NumCellCaches, deserialized.NumCellCaches)
}

func Test_ColorCacheCapabilitySet_Deserialize(t *testing.T) {
	set := ColorCacheCapabilitySet{ColorTableCacheSize: 6}
	serialized := set.Serialize()

	var deserialized ColorCacheCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, set.ColorTableCacheSize, deserialized.ColorTableCacheSize)
}

func Test_PointerCapabilitySet_Deserialize(t *testing.T) {
	tests := []struct {
		name             string
		lengthCapability uint16
		set              PointerCapabilitySet
	}{
		{
			name:             "Full",
			lengthCapability: 6,
			set: PointerCapabilitySet{
				ColorPointerFlag:      1,
				ColorPointerCacheSize: 20,
				PointerCacheSize:      21,
			},
		},
		{
			name:             "Short",
			lengthCapability: 4,
			set: PointerCapabilitySet{
				ColorPointerFlag:      1,
				ColorPointerCacheSize: 20,
				lengthCapability:      4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := tt.set
			serialized := set.Serialize()

			deserialized := PointerCapabilitySet{lengthCapability: tt.lengthCapability}
			var dataToRead []byte
			if tt.lengthCapability == 4 {
				dataToRead = serialized[:4]
			} else {
				dataToRead = serialized
			}
			err := deserialized.Deserialize(bytes.NewReader(dataToRead))
			require.NoError(t, err)
			require.Equal(t, set.ColorPointerFlag, deserialized.ColorPointerFlag)
			require.Equal(t, set.ColorPointerCacheSize, deserialized.ColorPointerCacheSize)
		})
	}
}

func Test_InputCapabilitySet_Deserialize(t *testing.T) {
	set := InputCapabilitySet{
		InputFlags:          0x0015,
		KeyboardLayout:      0x00000409,
		KeyboardType:        4,
		KeyboardFunctionKey: 12,
	}
	serialized := set.Serialize()

	var deserialized InputCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, set, deserialized)
}

func Test_BrushCapabilitySet_Deserialize(t *testing.T) {
	set := BrushCapabilitySet{BrushSupportLevel: 1}
	serialized := set.Serialize()

	var deserialized BrushCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, set, deserialized)
}

func Test_GlyphCacheCapabilitySet_Deserialize(t *testing.T) {
	set := GlyphCacheCapabilitySet{
		GlyphCache: [10]CacheDefinition{
			{CacheEntries: 254, CacheMaximumCellSize: 4},
			{CacheEntries: 254, CacheMaximumCellSize: 4},
			{CacheEntries: 254, CacheMaximumCellSize: 8},
			{CacheEntries: 254, CacheMaximumCellSize: 8},
			{CacheEntries: 254, CacheMaximumCellSize: 16},
			{CacheEntries: 254, CacheMaximumCellSize: 32},
			{CacheEntries: 254, CacheMaximumCellSize: 64},
			{CacheEntries: 254, CacheMaximumCellSize: 128},
			{CacheEntries: 254, CacheMaximumCellSize: 256},
			{CacheEntries: 64, CacheMaximumCellSize: 256},
		},
		FragCache:         0x1000100,
		GlyphSupportLevel: GlyphSupportLevelEncode,
	}
	serialized := set.Serialize()

	var deserialized GlyphCacheCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, set, deserialized)
}

func Test_OffscreenBitmapCacheCapabilitySet_Deserialize(t *testing.T) {
	set := OffscreenBitmapCacheCapabilitySet{
		OffscreenSupportLevel: 1,
		OffscreenCacheSize:    7680,
		OffscreenCacheEntries: 100,
	}
	serialized := set.Serialize()

	var deserialized OffscreenBitmapCacheCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, set, deserialized)
}

func Test_VirtualChannelCapabilitySet_Deserialize(t *testing.T) {
	set := VirtualChannelCapabilitySet{Flags: 0x00000001}
	serialized := set.Serialize()

	var deserialized VirtualChannelCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, set.Flags, deserialized.Flags)
}

func Test_DrawNineGridCacheCapabilitySet_Deserialize(t *testing.T) {
	set := DrawNineGridCacheCapabilitySet{
		drawNineGridSupportLevel: 2,
		drawNineGridCacheSize:    2560,
		drawNineGridCacheEntries: 256,
	}
	serialized := set.Serialize()

	var deserialized DrawNineGridCacheCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, set, deserialized)
}

func Test_DrawGDIPlusCapabilitySet_Deserialize(t *testing.T) {
	set := DrawGDIPlusCapabilitySet{}
	serialized := set.Serialize()

	var deserialized DrawGDIPlusCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
}

func Test_SoundCapabilitySet_Deserialize(t *testing.T) {
	set := SoundCapabilitySet{SoundFlags: 0x0001}
	serialized := set.Serialize()

	var deserialized SoundCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, set, deserialized)
}

func Test_FrameAcknowledgeCapabilitySet_SerializeDeserialize(t *testing.T) {
	set := FrameAcknowledgeCapabilitySet{MaxUnacknowledgedFrames: 5}
	serialized := set.Serialize()

	var deserialized FrameAcknowledgeCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, set, deserialized)
}

func Test_SurfaceCommandsCapabilitySet_SerializeDeserialize(t *testing.T) {
	set := SurfaceCommandsCapabilitySet{
		CmdFlags: SurfCmdSetSurfaceBits | SurfCmdFrameMarker | SurfCmdStreamSurfBits,
	}
	serialized := set.Serialize()

	var deserialized SurfaceCommandsCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, set.CmdFlags, deserialized.CmdFlags)
}

func Test_MultifragmentUpdateCapabilitySet_SerializeDeserialize(t *testing.T) {
	set := MultifragmentUpdateCapabilitySet{MaxRequestSize: 65536}
	serialized := set.Serialize()

	var deserialized MultifragmentUpdateCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, set, deserialized)
}

func Test_LargePointerCapabilitySet_Deserialize(t *testing.T) {
	data := []byte{0x01, 0x00}
	var set LargePointerCapabilitySet
	err := set.Deserialize(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, uint16(1), set.LargePointerSupportFlags)
}

func Test_DesktopCompositionCapabilitySet_Deserialize(t *testing.T) {
	data := []byte{0x01, 0x00}
	var set DesktopCompositionCapabilitySet
	err := set.Deserialize(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, uint16(1), set.CompDeskSupportLevel)
}

func Test_BitmapCodecsCapabilitySet_Deserialize(t *testing.T) {
	// Create a codec capability set
	set := NewBitmapCodecsCapabilitySet()
	serialized := set.Serialize()
	// Skip header (4 bytes)
	data := serialized[4:]

	var deserialized BitmapCodecsCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(data))
	require.NoError(t, err)
	require.Len(t, deserialized.BitmapCodecArray, 1)
	require.Equal(t, NSCodecGUID, deserialized.BitmapCodecArray[0].CodecGUID)
}

func Test_RailCapabilitySet_Serialize(t *testing.T) {
	set := NewRailCapabilitySet()
	serialized := set.Serialize()
	require.NotEmpty(t, serialized)
	// Type(2) + Length(2) + RailSupportLevel(4) = 8 bytes
	require.Len(t, serialized, 8)
}

func Test_WindowListCapabilitySet_Serialize(t *testing.T) {
	set := NewWindowListCapabilitySet()
	serialized := set.Serialize()
	require.NotEmpty(t, serialized)
	// Type(2) + Length(2) + WndSupportLevel(4) + NumIconCaches(1) + NumIconCacheEntries(2) = 11 bytes
	require.Len(t, serialized, 11)
}

func Test_ControlCapabilitySet_Deserialize(t *testing.T) {
	set := ControlCapabilitySet{}
	serialized := set.Serialize()

	var deserialized ControlCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
}

func Test_WindowActivationCapabilitySet_Deserialize(t *testing.T) {
	set := WindowActivationCapabilitySet{}
	serialized := set.Serialize()

	var deserialized WindowActivationCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
}

func Test_ShareCapabilitySet_Deserialize(t *testing.T) {
	set := ShareCapabilitySet{}
	serialized := set.Serialize()

	var deserialized ShareCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
}

func Test_FontCapabilitySet_Deserialize(t *testing.T) {
	set := FontCapabilitySet{fontSupportFlags: 0x0001}
	serialized := set.Serialize()

	var deserialized FontCapabilitySet
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, set.fontSupportFlags, deserialized.fontSupportFlags)
}

func Test_CapabilitySet_Deserialize_AllTypes(t *testing.T) {
	tests := []struct {
		name    string
		capType CapabilitySetType
		set     CapabilitySet
	}{
		{
			name:    "General",
			capType: CapabilitySetTypeGeneral,
			set:     NewGeneralCapabilitySet(),
		},
		{
			name:    "Bitmap",
			capType: CapabilitySetTypeBitmap,
			set:     NewBitmapCapabilitySet(1920, 1080),
		},
		{
			name:    "Order",
			capType: CapabilitySetTypeOrder,
			set:     NewOrderCapabilitySet(),
		},
		{
			name:    "BitmapCacheRev1",
			capType: CapabilitySetTypeBitmapCache,
			set:     NewBitmapCacheCapabilitySetRev1(),
		},
		{
			name:    "Pointer",
			capType: CapabilitySetTypePointer,
			set:     NewPointerCapabilitySet(),
		},
		{
			name:    "Input",
			capType: CapabilitySetTypeInput,
			set:     NewInputCapabilitySet(),
		},
		{
			name:    "Brush",
			capType: CapabilitySetTypeBrush,
			set:     NewBrushCapabilitySet(),
		},
		{
			name:    "GlyphCache",
			capType: CapabilitySetTypeGlyphCache,
			set:     NewGlyphCacheCapabilitySet(),
		},
		{
			name:    "OffscreenBitmapCache",
			capType: CapabilitySetTypeOffscreenBitmapCache,
			set:     NewOffscreenBitmapCacheCapabilitySet(),
		},
		{
			name:    "VirtualChannel",
			capType: CapabilitySetTypeVirtualChannel,
			set:     NewVirtualChannelCapabilitySet(),
		},
		{
			name:    "Sound",
			capType: CapabilitySetTypeSound,
			set:     NewSoundCapabilitySet(),
		},
		{
			name:    "MultifragmentUpdate",
			capType: CapabilitySetTypeMultifragmentUpdate,
			set:     NewMultifragmentUpdateCapabilitySet(),
		},
		{
			name:    "FrameAcknowledge",
			capType: CapabilitySetTypeFrameAcknowledge,
			set:     NewFrameAcknowledgeCapabilitySet(),
		},
		{
			name:    "SurfaceCommands",
			capType: CapabilitySetTypeSurfaceCommands,
			set:     NewSurfaceCommandsCapabilitySet(),
		},
		{
			name:    "BitmapCodecs",
			capType: CapabilitySetTypeBitmapCodecs,
			set:     NewBitmapCodecsCapabilitySet(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized := tt.set.Serialize()

			var deserialized CapabilitySet
			err := deserialized.Deserialize(bytes.NewReader(serialized))
			require.NoError(t, err)
			require.Equal(t, tt.capType, deserialized.CapabilitySetType)
		})
	}
}

func Test_CapabilitySet_DeserializeQuick(t *testing.T) {
	set := NewGeneralCapabilitySet()
	serialized := set.Serialize()

	var deserialized CapabilitySet
	err := deserialized.DeserializeQuick(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, CapabilitySetTypeGeneral, deserialized.CapabilitySetType)
}

func Test_CapabilitySet_DeserializeUnknownType(t *testing.T) {
	// Create data with unknown capability type
	data := []byte{
		0xFF, 0xFF, // Unknown type
		0x08, 0x00, // Length = 8
		0x00, 0x00, 0x00, 0x00, // Data
	}

	var set CapabilitySet
	err := set.Deserialize(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, CapabilitySetType(0xFFFF), set.CapabilitySetType)
}

func Test_NewCapabilitySets(t *testing.T) {
	// Test all New* constructor functions
	tests := []struct {
		name string
		set  CapabilitySet
	}{
		{"General", NewGeneralCapabilitySet()},
		{"Bitmap", NewBitmapCapabilitySet(1920, 1080)},
		{"Order", NewOrderCapabilitySet()},
		{"BitmapCacheRev1", NewBitmapCacheCapabilitySetRev1()},
		{"Pointer", NewPointerCapabilitySet()},
		{"Input", NewInputCapabilitySet()},
		{"Brush", NewBrushCapabilitySet()},
		{"GlyphCache", NewGlyphCacheCapabilitySet()},
		{"OffscreenBitmapCache", NewOffscreenBitmapCacheCapabilitySet()},
		{"VirtualChannel", NewVirtualChannelCapabilitySet()},
		{"Sound", NewSoundCapabilitySet()},
		{"MultifragmentUpdate", NewMultifragmentUpdateCapabilitySet()},
		{"FrameAcknowledge", NewFrameAcknowledgeCapabilitySet()},
		{"SurfaceCommands", NewSurfaceCommandsCapabilitySet()},
		{"BitmapCodecs", NewBitmapCodecsCapabilitySet()},
		{"Rail", NewRailCapabilitySet()},
		{"WindowList", NewWindowListCapabilitySet()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized := tt.set.Serialize()
			require.NotEmpty(t, serialized)
		})
	}
}

func Test_ClientConfirmActive_Deserialize(t *testing.T) {
	original := NewClientConfirmActive(66538, 1007, 1920, 1080, false)
	serialized := original.Serialize()

	var deserialized ClientConfirmActive
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, original.ShareID, deserialized.ShareID)
	require.Equal(t, len(original.CapabilitySets), len(deserialized.CapabilitySets))
}

func Test_ClientConfirmActive_WithRemoteApp(t *testing.T) {
	original := NewClientConfirmActive(66538, 1007, 1920, 1080, true)
	serialized := original.Serialize()
	require.NotEmpty(t, serialized)

	var deserialized ClientConfirmActive
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	// With RemoteApp, we have 2 additional capability sets (Rail + WindowList)
	require.Equal(t, len(original.CapabilitySets), len(deserialized.CapabilitySets))
}

func Test_NSCodecCapabilitySet_Serialize(t *testing.T) {
	set := NSCodecCapabilitySet{
		FAllowDynamicFidelity: 1,
		FAllowSubsampling:     1,
		ColorLossLevel:        3,
	}
	serialized := set.Serialize()
	require.Equal(t, []byte{1, 1, 3}, serialized)
}

func Test_BitmapCodec_Serialize(t *testing.T) {
	codec := BitmapCodec{
		CodecGUID:       NSCodecGUID,
		CodecID:         1,
		CodecProperties: []byte{1, 1, 3},
	}
	serialized := codec.Serialize()
	require.NotEmpty(t, serialized)
	// 16 (GUID) + 1 (CodecID) + 2 (length) + 3 (properties) = 22
	require.Len(t, serialized, 22)
}

func Test_CacheDefinition_SerializeDeserialize(t *testing.T) {
	def := CacheDefinition{
		CacheEntries:         254,
		CacheMaximumCellSize: 128,
	}
	serialized := def.Serialize()
	require.Len(t, serialized, 4)

	var deserialized CacheDefinition
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.Equal(t, def, deserialized)
}
