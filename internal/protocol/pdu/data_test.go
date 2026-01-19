package pdu

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestType_IsMethods(t *testing.T) {
	tests := []struct {
		name           string
		pduType        Type
		isDemandActive bool
		isConfirmActive bool
		isDeactivateAll bool
		isData          bool
	}{
		{
			name:           "DemandActive",
			pduType:        TypeDemandActive,
			isDemandActive: true,
		},
		{
			name:            "ConfirmActive",
			pduType:         TypeConfirmActive,
			isConfirmActive: true,
		},
		{
			name:            "DeactivateAll",
			pduType:         TypeDeactivateAll,
			isDeactivateAll: true,
		},
		{
			name:   "Data",
			pduType: TypeData,
			isData: true,
		},
		{
			name:    "Unknown",
			pduType: Type(0xFF),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isDemandActive, tt.pduType.IsDemandActive())
			require.Equal(t, tt.isConfirmActive, tt.pduType.IsConfirmActive())
			require.Equal(t, tt.isDeactivateAll, tt.pduType.IsDeactivateAll())
			require.Equal(t, tt.isData, tt.pduType.IsData())
		})
	}
}

func TestType2_IsMethods(t *testing.T) {
	tests := []struct {
		name            string
		pduType2        Type2
		isUpdate        bool
		isControl       bool
		isPointer       bool
		isInput         bool
		isSynchronize   bool
		isFontlist      bool
		isFontmap       bool
		isErrorInfo     bool
		isSaveSession   bool
	}{
		{
			name:     "Update",
			pduType2: Type2Update,
			isUpdate: true,
		},
		{
			name:      "Control",
			pduType2:  Type2Control,
			isControl: true,
		},
		{
			name:      "Pointer",
			pduType2:  Type2Pointer,
			isPointer: true,
		},
		{
			name:    "Input",
			pduType2: Type2Input,
			isInput: true,
		},
		{
			name:          "Synchronize",
			pduType2:      Type2Synchronize,
			isSynchronize: true,
		},
		{
			name:       "Fontlist",
			pduType2:   Type2Fontlist,
			isFontlist: true,
		},
		{
			name:      "Fontmap",
			pduType2:  Type2Fontmap,
			isFontmap: true,
		},
		{
			name:        "ErrorInfo",
			pduType2:    Type2ErrorInfo,
			isErrorInfo: true,
		},
		{
			name:          "SaveSessionInfo",
			pduType2:      Type2SaveSessionInfo,
			isSaveSession: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isUpdate, tt.pduType2.IsUpdate())
			require.Equal(t, tt.isControl, tt.pduType2.IsControl())
			require.Equal(t, tt.isPointer, tt.pduType2.IsPointer())
			require.Equal(t, tt.isInput, tt.pduType2.IsInput())
			require.Equal(t, tt.isSynchronize, tt.pduType2.IsSynchronize())
			require.Equal(t, tt.isFontlist, tt.pduType2.IsFontlist())
			require.Equal(t, tt.isFontmap, tt.pduType2.IsFontmap())
			require.Equal(t, tt.isErrorInfo, tt.pduType2.IsErrorInfo())
			require.Equal(t, tt.isSaveSession, tt.pduType2.IsSaveSessionInfo())
		})
	}
}

func TestShareControlHeader_SerializeDeserialize(t *testing.T) {
	tests := []struct {
		name        string
		header      ShareControlHeader
		expected    []byte
	}{
		{
			name: "DemandActive",
			header: ShareControlHeader{
				TotalLength: 100,
				PDUType:     TypeDemandActive,
				PDUSource:   1007,
			},
			expected: []byte{0x64, 0x00, 0x11, 0x00, 0xef, 0x03},
		},
		{
			name: "ConfirmActive",
			header: ShareControlHeader{
				TotalLength: 200,
				PDUType:     TypeConfirmActive,
				PDUSource:   1004,
			},
			expected: []byte{0xc8, 0x00, 0x13, 0x00, 0xec, 0x03},
		},
		{
			name: "Data",
			header: ShareControlHeader{
				TotalLength: 50,
				PDUType:     TypeData,
				PDUSource:   1002,
			},
			expected: []byte{0x32, 0x00, 0x17, 0x00, 0xea, 0x03},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Serialize
			actual := tt.header.Serialize()
			require.Equal(t, tt.expected, actual)

			// Test Deserialize
			var deserialized ShareControlHeader
			err := deserialized.Deserialize(bytes.NewReader(tt.expected))
			require.NoError(t, err)
			require.Equal(t, tt.header, deserialized)
		})
	}
}

func TestShareDataHeader_SerializeDeserialize(t *testing.T) {
	tests := []struct {
		name     string
		header   ShareDataHeader
		wantErr  bool
	}{
		{
			name: "Synchronize",
			header: ShareDataHeader{
				ShareControlHeader: ShareControlHeader{
					TotalLength: 22,
					PDUType:     TypeData,
					PDUSource:   1007,
				},
				ShareID:            66538,
				StreamID:           0x01,
				UncompressedLength: 8,
				PDUType2:           Type2Synchronize,
				CompressedType:     0,
				CompressedLength:   0,
			},
		},
		{
			name: "Control",
			header: ShareDataHeader{
				ShareControlHeader: ShareControlHeader{
					TotalLength: 26,
					PDUType:     TypeData,
					PDUSource:   1007,
				},
				ShareID:            66538,
				StreamID:           0x01,
				UncompressedLength: 12,
				PDUType2:           Type2Control,
				CompressedType:     0,
				CompressedLength:   0,
			},
		},
		{
			name: "Fontlist",
			header: ShareDataHeader{
				ShareControlHeader: ShareControlHeader{
					TotalLength: 26,
					PDUType:     TypeData,
					PDUSource:   1004,
				},
				ShareID:            66538,
				StreamID:           0x01,
				UncompressedLength: 12,
				PDUType2:           Type2Fontlist,
				CompressedType:     0,
				CompressedLength:   0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Serialize
			serialized := tt.header.Serialize()
			require.NotEmpty(t, serialized)

			// Test Deserialize
			var deserialized ShareDataHeader
			err := deserialized.Deserialize(bytes.NewReader(serialized))
			require.NoError(t, err)
			require.Equal(t, tt.header, deserialized)
		})
	}
}

func TestShareDataHeader_DeserializeDeactivateAll(t *testing.T) {
	// Create a ShareControlHeader with DeactivateAll type
	header := ShareControlHeader{
		TotalLength: 6,
		PDUType:     TypeDeactivateAll,
		PDUSource:   1007,
	}
	data := header.Serialize()

	var shareDataHeader ShareDataHeader
	err := shareDataHeader.Deserialize(bytes.NewReader(data))
	require.ErrorIs(t, err, ErrDeactiateAll)
}

func TestShareDataHeader_DeserializeErrors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Empty",
			data: []byte{},
		},
		{
			name: "TooShort",
			data: []byte{0x00, 0x00, 0x17, 0x00},
		},
		{
			name: "PartialShareID",
			data: []byte{0x16, 0x00, 0x17, 0x00, 0xef, 0x03, 0xea, 0x03},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var header ShareDataHeader
			err := header.Deserialize(bytes.NewReader(tt.data))
			require.Error(t, err)
		})
	}
}

func TestData_Serialize(t *testing.T) {
	tests := []struct {
		name string
		data *Data
	}{
		{
			name: "Synchronize",
			data: NewSynchronize(66538, 1007),
		},
		{
			name: "ControlCooperate",
			data: NewControl(66538, 1007, ControlActionCooperate),
		},
		{
			name: "ControlRequestControl",
			data: NewControl(66538, 1007, ControlActionRequestControl),
		},
		{
			name: "FontList",
			data: NewFontList(66538, 1007),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized := tt.data.Serialize()
			require.NotEmpty(t, serialized)
			require.GreaterOrEqual(t, len(serialized), 18) // minimum ShareDataHeader size
		})
	}
}

func TestData_DeserializeSynchronize(t *testing.T) {
	original := NewSynchronize(66538, 1007)
	serialized := original.Serialize()

	var deserialized Data
	err := deserialized.Deserialize(bytes.NewReader(serialized))
	require.NoError(t, err)
	require.NotNil(t, deserialized.SynchronizePDUData)
	require.Equal(t, MessageTypeSync, deserialized.SynchronizePDUData.MessageType)
}

func TestData_DeserializeControl(t *testing.T) {
	tests := []struct {
		name   string
		action ControlAction
	}{
		{"Cooperate", ControlActionCooperate},
		{"RequestControl", ControlActionRequestControl},
		{"GrantedControl", ControlActionGrantedControl},
		{"Detach", ControlActionDetach},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := NewControl(66538, 1007, tt.action)
			serialized := original.Serialize()

			var deserialized Data
			err := deserialized.Deserialize(bytes.NewReader(serialized))
			require.NoError(t, err)
			require.NotNil(t, deserialized.ControlPDUData)
			require.Equal(t, tt.action, deserialized.ControlPDUData.Action)
		})
	}
}

func TestData_DeserializeFontMap(t *testing.T) {
	// Create FontMap PDU data directly
	header := newShareDataHeader(66538, 1007, TypeData, Type2Fontmap)
	header.ShareControlHeader.TotalLength = 26
	header.UncompressedLength = 12

	buf := bytes.Buffer{}
	buf.Write(header.Serialize())
	// FontMap data: numberEntries, totalNumEntries, mapFlags, entrySize
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x04, 0x00})

	var data Data
	err := data.Deserialize(&buf)
	require.NoError(t, err)
	require.NotNil(t, data.FontMapPDUData)
}

func TestData_DeserializeErrorInfo(t *testing.T) {
	header := newShareDataHeader(66538, 1007, TypeData, Type2ErrorInfo)
	header.ShareControlHeader.TotalLength = 22
	header.UncompressedLength = 8

	buf := bytes.Buffer{}
	buf.Write(header.Serialize())
	// Error info: ERRINFO_RPC_INITIATED_DISCONNECT (0x00000001)
	buf.Write([]byte{0x01, 0x00, 0x00, 0x00})

	var data Data
	err := data.Deserialize(&buf)
	require.NoError(t, err)
	require.NotNil(t, data.ErrorInfoPDUData)
	require.Equal(t, uint32(0x00000001), data.ErrorInfoPDUData.ErrorInfo)
}

func TestData_DeserializeSaveSessionInfo(t *testing.T) {
	header := newShareDataHeader(66538, 1007, TypeData, Type2SaveSessionInfo)
	header.ShareControlHeader.TotalLength = 18
	header.UncompressedLength = 4

	buf := bytes.Buffer{}
	buf.Write(header.Serialize())

	var data Data
	err := data.Deserialize(&buf)
	require.NoError(t, err)
	// SaveSessionInfo is ignored
}

func TestData_DeserializeUpdate(t *testing.T) {
	header := newShareDataHeader(66538, 1007, TypeData, Type2Update)
	header.ShareControlHeader.TotalLength = 18
	header.UncompressedLength = 4

	buf := bytes.Buffer{}
	buf.Write(header.Serialize())

	var data Data
	err := data.Deserialize(&buf)
	require.NoError(t, err)
	// Update is handled via fastpath, ignored here
}

func TestData_DeserializePointer(t *testing.T) {
	header := newShareDataHeader(66538, 1007, TypeData, Type2Pointer)
	header.ShareControlHeader.TotalLength = 18
	header.UncompressedLength = 4

	buf := bytes.Buffer{}
	buf.Write(header.Serialize())

	var data Data
	err := data.Deserialize(&buf)
	require.NoError(t, err)
	// Pointer updates are ignored
}

func TestData_DeserializeUnknownType(t *testing.T) {
	header := newShareDataHeader(66538, 1007, TypeData, Type2(0xFF))
	header.ShareControlHeader.TotalLength = 18
	header.UncompressedLength = 4

	buf := bytes.Buffer{}
	buf.Write(header.Serialize())

	var data Data
	err := data.Deserialize(&buf)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown data pdu")
}

func TestSynchronizePDUData_SerializeDeserialize(t *testing.T) {
	tests := []struct {
		name        string
		messageType MessageType
	}{
		{"Sync", MessageTypeSync},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := &SynchronizePDUData{MessageType: tt.messageType}
			serialized := pdu.Serialize()
			require.Len(t, serialized, 4)

			var deserialized SynchronizePDUData
			err := deserialized.Deserialize(bytes.NewReader(serialized))
			require.NoError(t, err)
			require.Equal(t, tt.messageType, deserialized.MessageType)
		})
	}
}

func TestControlPDUData_SerializeDeserialize(t *testing.T) {
	tests := []struct {
		name      string
		action    ControlAction
		grantID   uint16
		controlID uint32
	}{
		{"Cooperate", ControlActionCooperate, 0, 0},
		{"RequestControl", ControlActionRequestControl, 0, 0},
		{"GrantedControl", ControlActionGrantedControl, 1007, 12345},
		{"Detach", ControlActionDetach, 1004, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := &ControlPDUData{
				Action:    tt.action,
				GrantID:   tt.grantID,
				ControlID: tt.controlID,
			}
			serialized := pdu.Serialize()
			require.Len(t, serialized, 8)

			var deserialized ControlPDUData
			err := deserialized.Deserialize(bytes.NewReader(serialized))
			require.NoError(t, err)
			require.Equal(t, tt.action, deserialized.Action)
			require.Equal(t, tt.grantID, deserialized.GrantID)
			require.Equal(t, tt.controlID, deserialized.ControlID)
		})
	}
}

func TestFontListPDUData_Serialize(t *testing.T) {
	pdu := &FontListPDUData{}
	serialized := pdu.Serialize()
	require.Len(t, serialized, 8)
	// Expected: numberFonts(0), totalNumFonts(0), listFlags(3), entrySize(0x32)
	expected := []byte{0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x32, 0x00}
	require.Equal(t, expected, serialized)
}

func TestFontMapPDUData_Deserialize(t *testing.T) {
	// FontMap data: numberEntries(0), totalNumEntries(0), mapFlags(3), entrySize(4)
	data := []byte{0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x04, 0x00}

	var pdu FontMapPDUData
	err := pdu.Deserialize(bytes.NewReader(data))
	require.NoError(t, err)
}

func TestFontMapPDUData_DeserializeErrors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"Empty", []byte{}},
		{"TooShort", []byte{0x00, 0x00}},
		{"PartialData", []byte{0x00, 0x00, 0x00, 0x00, 0x03}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pdu FontMapPDUData
			err := pdu.Deserialize(bytes.NewReader(tt.data))
			require.Error(t, err)
		})
	}
}

func TestNewShareControlHeader(t *testing.T) {
	header := newShareControlHeader(TypeDemandActive, 1007)
	require.Equal(t, TypeDemandActive, header.PDUType)
	require.Equal(t, uint16(1007), header.PDUSource)
	require.Equal(t, uint16(0), header.TotalLength) // Not set until serialize
}

func TestNewShareDataHeader(t *testing.T) {
	header := newShareDataHeader(66538, 1007, TypeData, Type2Synchronize)
	require.Equal(t, uint32(66538), header.ShareID)
	require.Equal(t, TypeData, header.ShareControlHeader.PDUType)
	require.Equal(t, uint16(1007), header.ShareControlHeader.PDUSource)
	require.Equal(t, Type2Synchronize, header.PDUType2)
	require.Equal(t, uint8(0x01), header.StreamID)
}

func TestSynchronizePDUData_DeserializeError(t *testing.T) {
	// Test with EOF
	var pdu SynchronizePDUData
	err := pdu.Deserialize(bytes.NewReader([]byte{}))
	require.ErrorIs(t, err, io.EOF)
}

func TestControlPDUData_DeserializeError(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"Empty", []byte{}},
		{"PartialAction", []byte{0x01}},
		{"MissingControlID", []byte{0x01, 0x00, 0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pdu ControlPDUData
			err := pdu.Deserialize(bytes.NewReader(tt.data))
			require.Error(t, err)
		})
	}
}
