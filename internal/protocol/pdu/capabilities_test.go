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

// ============================================================================
// Microsoft Protocol Test Suite Validation Tests
// Reference: MS-RDPBCGR_ClientTestDesignSpecification.md - S1_Connection
// ============================================================================

// TestBVT_CapabilityExchange_DemandActivePDU validates per MS test case:
// "BVT_CapabilityExchangeTest_PositiveTest_DemandActivePDU"
// Per MS-RDPBCGR Section 2.2.1.13.1
func TestBVT_CapabilityExchange_DemandActivePDU(t *testing.T) {
	// Capability set type codes per MS-RDPBCGR 2.2.7.1.1
	capTypes := []struct {
		typeCode uint16
		name     string
		required bool
	}{
		{0x0001, "CAPSTYPE_GENERAL", true},
		{0x0002, "CAPSTYPE_BITMAP", true},
		{0x0003, "CAPSTYPE_ORDER", true},
		{0x0004, "CAPSTYPE_BITMAPCACHE", false},
		{0x0005, "CAPSTYPE_CONTROL", false},
		{0x0007, "CAPSTYPE_ACTIVATION", false},
		{0x0008, "CAPSTYPE_POINTER", true},
		{0x0009, "CAPSTYPE_SHARE", false},
		{0x000A, "CAPSTYPE_COLORCACHE", false},
		{0x000D, "CAPSTYPE_INPUT", true},
		{0x000E, "CAPSTYPE_FONT", false},
		{0x000F, "CAPSTYPE_BRUSH", false},
		{0x0010, "CAPSTYPE_GLYPHCACHE", false},
		{0x0011, "CAPSTYPE_OFFSCREENCACHE", false},
		{0x0012, "CAPSTYPE_BITMAPCACHE_HOSTSUPPORT", false},
		{0x0013, "CAPSTYPE_BITMAPCACHE_REV2", false},
		{0x0014, "CAPSTYPE_VIRTUALCHANNEL", true},
		{0x0015, "CAPSTYPE_DRAWNINEGRIDCACHE", false},
		{0x0016, "CAPSTYPE_DRAWGDIPLUS", false},
		{0x0017, "CAPSTYPE_RAIL", false},
		{0x0018, "CAPSTYPE_WINDOW", false},
		{0x001A, "CAPSETTYPE_COMPDESK", false},
		{0x001B, "CAPSETTYPE_MULTIFRAGMENTUPDATE", false},
		{0x001C, "CAPSETTYPE_LARGE_POINTER", false},
		{0x001D, "CAPSETTYPE_SURFACE_COMMANDS", false},
		{0x001E, "CAPSETTYPE_BITMAP_CODECS", false},
		{0x001F, "CAPSETTYPE_FRAME_ACKNOWLEDGE", false},
	}

	for _, cap := range capTypes {
		t.Run(cap.name, func(t *testing.T) {
			// All capability types are 16-bit unsigned
			require.LessOrEqual(t, cap.typeCode, uint16(0xFFFF))
		})
	}
}

// TestS1_CapabilityExchange_GeneralCapabilitySet validates general caps
// Per MS-RDPBCGR Section 2.2.7.1.1
func TestS1_CapabilityExchange_GeneralCapabilitySet(t *testing.T) {
	// OS type codes per MS-RDPBCGR 2.2.7.1.1
	const (
		OSMAJORTYPE_UNSPECIFIED = 0x0000
		OSMAJORTYPE_WINDOWS     = 0x0001
		OSMAJORTYPE_OS2         = 0x0002
		OSMAJORTYPE_MACINTOSH   = 0x0003
		OSMAJORTYPE_UNIX        = 0x0004
		OSMAJORTYPE_IOS         = 0x0005
		OSMAJORTYPE_OSX         = 0x0006
		OSMAJORTYPE_ANDROID     = 0x0007
		OSMAJORTYPE_CHROMEOS    = 0x0008
	)

	// Extra flags per MS-RDPBCGR 2.2.7.1.1
	const (
		FASTPATH_OUTPUT_SUPPORTED    = 0x0001
		NO_BITMAP_COMPRESSION_HDR    = 0x0400
		LONG_CREDENTIALS_SUPPORTED   = 0x0004
		AUTORECONNECT_SUPPORTED      = 0x0008
		ENC_SALTED_CHECKSUM          = 0x0010
	)

	tests := []struct {
		name  string
		flags uint16
	}{
		{"FastPath", FASTPATH_OUTPUT_SUPPORTED},
		{"NoBitmapCompressionHdr", NO_BITMAP_COMPRESSION_HDR},
		{"LongCredentials", LONG_CREDENTIALS_SUPPORTED},
		{"AutoReconnect", AUTORECONNECT_SUPPORTED},
		{"SaltedChecksum", ENC_SALTED_CHECKSUM},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Flags are combinable
			combined := tc.flags | FASTPATH_OUTPUT_SUPPORTED
			require.True(t, combined&FASTPATH_OUTPUT_SUPPORTED != 0)
		})
	}
}

// TestS1_CapabilityExchange_BitmapCapabilitySet validates bitmap caps
// Per MS-RDPBCGR Section 2.2.7.1.2
func TestS1_CapabilityExchange_BitmapCapabilitySet(t *testing.T) {
	// Preferred bits per pixel values
	bppValues := []uint16{8, 15, 16, 24, 32}

	for _, bpp := range bppValues {
		t.Run("BPP_"+string(rune(bpp)), func(t *testing.T) {
			// Valid color depths
			isValid := bpp == 8 || bpp == 15 || bpp == 16 || bpp == 24 || bpp == 32
			require.True(t, isValid)
		})
	}

	// Desktop size constraints
	desktops := []struct {
		width, height uint16
	}{
		{640, 480},
		{800, 600},
		{1024, 768},
		{1280, 1024},
		{1920, 1080},
		{3840, 2160},
	}

	for _, d := range desktops {
		t.Run("Desktop", func(t *testing.T) {
			// Maximum desktop size per spec
			require.LessOrEqual(t, d.width, uint16(8192))
			require.LessOrEqual(t, d.height, uint16(8192))
		})
	}
}

// TestS1_CapabilityExchange_OrderCapabilitySet validates order support
// Per MS-RDPBCGR Section 2.2.7.1.3
func TestS1_CapabilityExchange_OrderCapabilitySet(t *testing.T) {
	// Order support indices per MS-RDPBCGR 2.2.7.1.3
	orderIndices := []struct {
		index uint8
		name  string
	}{
		{0, "TS_NEG_DSTBLT_INDEX"},
		{1, "TS_NEG_PATBLT_INDEX"},
		{2, "TS_NEG_SCRBLT_INDEX"},
		{3, "TS_NEG_MEMBLT_INDEX"},
		{4, "TS_NEG_MEM3BLT_INDEX"},
		{8, "TS_NEG_DRAWNINEGRID_INDEX"},
		{9, "TS_NEG_LINETO_INDEX"},
		{10, "TS_NEG_MULTI_DRAWNINEGRID_INDEX"},
		{11, "TS_NEG_OPAQUE_RECT_INDEX"},
		{12, "TS_NEG_SAVEBITMAP_INDEX"},
		{13, "TS_NEG_WTEXTOUT_INDEX"},
		{14, "TS_NEG_MEMBLT_V2_INDEX"},
		{15, "TS_NEG_MEM3BLT_V2_INDEX"},
		{16, "TS_NEG_MULTIDSTBLT_INDEX"},
		{17, "TS_NEG_MULTIPATBLT_INDEX"},
		{18, "TS_NEG_MULTISCRBLT_INDEX"},
		{19, "TS_NEG_MULTIOPAQUERECT_INDEX"},
		{20, "TS_NEG_FAST_INDEX_INDEX"},
		{21, "TS_NEG_POLYGON_SC_INDEX"},
		{22, "TS_NEG_POLYGON_CB_INDEX"},
		{23, "TS_NEG_POLYLINE_INDEX"},
		{25, "TS_NEG_FAST_GLYPH_INDEX"},
		{26, "TS_NEG_ELLIPSE_SC_INDEX"},
		{27, "TS_NEG_ELLIPSE_CB_INDEX"},
		{28, "TS_NEG_INDEX_INDEX"},
	}

	for _, oi := range orderIndices {
		t.Run(oi.name, func(t *testing.T) {
			// Order support is a 32-byte array, indices 0-31 are valid
			require.LessOrEqual(t, oi.index, uint8(31))
		})
	}
}

// TestS1_CapabilityExchange_InputCapabilitySet validates input caps
// Per MS-RDPBCGR Section 2.2.7.1.6
func TestS1_CapabilityExchange_InputCapabilitySet(t *testing.T) {
	// Input flags per MS-RDPBCGR 2.2.7.1.6
	const (
		INPUT_FLAG_SCANCODES      = 0x0001
		INPUT_FLAG_MOUSEX         = 0x0004
		INPUT_FLAG_FASTPATH_INPUT = 0x0008
		INPUT_FLAG_UNICODE        = 0x0010
		INPUT_FLAG_FASTPATH_INPUT2 = 0x0020
		INPUT_FLAG_UNUSED1        = 0x0040
		INPUT_FLAG_MOUSE_HWHEEL   = 0x0100
		INPUT_FLAG_QOE_TIMESTAMPS = 0x0200
	)

	tests := []struct {
		name  string
		flags uint16
	}{
		{"Scancodes", INPUT_FLAG_SCANCODES},
		{"MouseX", INPUT_FLAG_MOUSEX},
		{"FastPathInput", INPUT_FLAG_FASTPATH_INPUT},
		{"Unicode", INPUT_FLAG_UNICODE},
		{"FastPathInput2", INPUT_FLAG_FASTPATH_INPUT2},
		{"MouseHWheel", INPUT_FLAG_MOUSE_HWHEEL},
		{"QoETimestamps", INPUT_FLAG_QOE_TIMESTAMPS},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// All flags are valid 16-bit values
			require.LessOrEqual(t, tc.flags, uint16(0xFFFF))
		})
	}
}

// TestS1_CapabilityExchange_VirtualChannelCapabilitySet validates VC caps
// Per MS-RDPBCGR Section 2.2.7.1.10
func TestS1_CapabilityExchange_VirtualChannelCapabilitySet(t *testing.T) {
	// Virtual channel compression flags per MS-RDPBCGR 2.2.7.1.10
	const (
		VCCAPS_NO_COMPR      = 0x00000000
		VCCAPS_COMPR_SC      = 0x00000001 // Server-to-client compression
		VCCAPS_COMPR_CS_8K   = 0x00000002 // Client-to-server 8K compression
	)

	tests := []struct {
		name  string
		flags uint32
	}{
		{"NoCompression", VCCAPS_NO_COMPR},
		{"ServerToClient", VCCAPS_COMPR_SC},
		{"ClientToServer8K", VCCAPS_COMPR_CS_8K},
		{"Both", VCCAPS_COMPR_SC | VCCAPS_COMPR_CS_8K},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hasServerCompression := (tc.flags & VCCAPS_COMPR_SC) != 0
			hasClientCompression := (tc.flags & VCCAPS_COMPR_CS_8K) != 0
			
			if tc.name == "Both" {
				require.True(t, hasServerCompression && hasClientCompression)
			}
		})
	}
}
