package x224

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockTpktConn implements the tpktConn interface for testing
type mockTpktConn struct {
	sendData    []byte
	sendErr     error
	receiveData io.Reader
	receiveErr  error
}

func (m *mockTpktConn) Send(pduData []byte) error {
	m.sendData = pduData
	return m.sendErr
}

func (m *mockTpktConn) Receive() (io.Reader, error) {
	return m.receiveData, m.receiveErr
}

// =============================================================================
// ConnectionRequest.Serialize() Tests
// =============================================================================

func Test_ConnectionRequest_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		req      ConnectionRequest
		expected []byte
	}{
		{
			name: "with user data",
			req: ConnectionRequest{
				CRCDT:        0xE0,
				DSTREF:       0,
				SRCREF:       0,
				ClassOption:  0,
				VariablePart: nil,
				UserData: []byte{
					0x43, 0x6f, 0x6f, 0x6b, 0x69, 0x65, 0x3a, 0x20, 0x6d, 0x73, 0x74, 0x73, 0x68, 0x61, 0x73, 0x68,
					0x3d, 0x65, 0x6c, 0x74, 0x6f, 0x6e, 0x73, 0x0d, 0x0a, 0x01, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00,
					0x00,
				},
			},
			expected: []byte{
				0x27, 0xe0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x43, 0x6f, 0x6f, 0x6b, 0x69, 0x65, 0x3a, 0x20, 0x6d,
				0x73, 0x74, 0x73, 0x68, 0x61, 0x73, 0x68, 0x3d, 0x65, 0x6c, 0x74, 0x6f, 0x6e, 0x73, 0x0d, 0x0a,
				0x01, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			name: "empty user data",
			req: ConnectionRequest{
				CRCDT:        0xE0,
				DSTREF:       0,
				SRCREF:       0,
				ClassOption:  0,
				VariablePart: nil,
				UserData:     nil,
			},
			expected: []byte{0x06, 0xe0, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name: "with non-zero refs",
			req: ConnectionRequest{
				CRCDT:        0xE0,
				DSTREF:       0x1234,
				SRCREF:       0x5678,
				ClassOption:  0x01,
				VariablePart: nil,
				UserData:     []byte{0xAA, 0xBB},
			},
			expected: []byte{0x08, 0xe0, 0x12, 0x34, 0x56, 0x78, 0x01, 0xAA, 0xBB},
		},
		{
			name: "minimal request",
			req: ConnectionRequest{
				CRCDT:       0xE0,
				ClassOption: 0,
			},
			expected: []byte{0x06, 0xe0, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.req.Serialize()
			require.Equal(t, tt.expected, actual)
		})
	}
}

// =============================================================================
// ConnectionConfirm.Deserialize() Tests
// =============================================================================

func Test_ConnectionConfirm_Deserialize(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expected    ConnectionConfirm
		expectedErr error
	}{
		{
			name: "valid connection confirm",
			input: []byte{
				0x0e, 0xd0, 0x00, 0x00,
				0x12, 0x34, 0x00, 0x02,
				0x00, 0x08, 0x00, 0x00,
				0x00, 0x00, 0x00,
			},
			expected: ConnectionConfirm{
				LI:          14,
				CCCDT:       0xd0,
				DSTREF:      0,
				SRCREF:      0x1234,
				ClassOption: 0,
			},
			expectedErr: nil,
		},
		{
			name: "valid with different values",
			input: []byte{
				0x0e, 0xd5, 0xAB, 0xCD,
				0xEF, 0x12, 0x03, 0x02,
				0x00, 0x08, 0x00, 0x00,
				0x00, 0x00, 0x00,
			},
			expected: ConnectionConfirm{
				LI:          14,
				CCCDT:       0xd5,
				DSTREF:      0xABCD,
				SRCREF:      0xEF12,
				ClassOption: 0x03,
			},
			expectedErr: nil,
		},
		{
			name:        "empty input - EOF on LI",
			input:       []byte{},
			expected:    ConnectionConfirm{},
			expectedErr: io.EOF,
		},
		{
			name:        "wrong length - too small",
			input:       []byte{0x05, 0xd0, 0x00, 0x00, 0x12, 0x34, 0x00},
			expected:    ConnectionConfirm{LI: 0x05},
			expectedErr: ErrSmallConnectionConfirmLength,
		},
		{
			name:        "wrong length - too large",
			input:       []byte{0x10, 0xd0, 0x00, 0x00, 0x12, 0x34, 0x00},
			expected:    ConnectionConfirm{LI: 0x10},
			expectedErr: ErrSmallConnectionConfirmLength,
		},
		{
			name:        "wrong connection confirm code",
			input:       []byte{0x0e, 0xE0, 0x00, 0x00, 0x12, 0x34, 0x00},
			expected:    ConnectionConfirm{LI: 0x0e, CCCDT: 0xE0},
			expectedErr: ErrWrongConnectionConfirmCode,
		},
		{
			name:        "truncated after LI",
			input:       []byte{0x0e},
			expected:    ConnectionConfirm{LI: 0x0e},
			expectedErr: io.EOF,
		},
		{
			name:        "truncated after CCCDT",
			input:       []byte{0x0e, 0xd0},
			expected:    ConnectionConfirm{LI: 0x0e, CCCDT: 0xd0},
			expectedErr: io.EOF, // binary.Read returns EOF for partial reads
		},
		{
			name:        "truncated after DSTREF",
			input:       []byte{0x0e, 0xd0, 0x00, 0x00},
			expected:    ConnectionConfirm{LI: 0x0e, CCCDT: 0xd0, DSTREF: 0},
			expectedErr: io.EOF, // binary.Read returns EOF for partial reads
		},
		{
			name:        "truncated after SRCREF",
			input:       []byte{0x0e, 0xd0, 0x00, 0x00, 0x12, 0x34},
			expected:    ConnectionConfirm{LI: 0x0e, CCCDT: 0xd0, DSTREF: 0, SRCREF: 0x1234},
			expectedErr: io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actual ConnectionConfirm
			input := bytes.NewBuffer(tt.input)
			err := actual.Deserialize(input)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, actual)
			}
		})
	}
}

// =============================================================================
// Data PDU Tests
// =============================================================================

func Test_Data_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		data     Data
		expected []byte
	}{
		{
			name: "standard data PDU",
			data: Data{
				LI:       0x02,
				DTROA:    0xF0,
				NREOT:    0x80,
				UserData: []byte{0x01, 0x02, 0x03, 0x04},
			},
			expected: []byte{0x02, 0xF0, 0x80, 0x01, 0x02, 0x03, 0x04},
		},
		{
			name: "empty user data",
			data: Data{
				LI:       0x02,
				DTROA:    0xF0,
				NREOT:    0x80,
				UserData: nil,
			},
			expected: []byte{0x02, 0xF0, 0x80},
		},
		{
			name: "with large user data",
			data: Data{
				LI:       0x02,
				DTROA:    0xF0,
				NREOT:    0x80,
				UserData: bytes.Repeat([]byte{0xAA}, 100),
			},
			expected: append([]byte{0x02, 0xF0, 0x80}, bytes.Repeat([]byte{0xAA}, 100)...),
		},
		{
			name: "non-standard values",
			data: Data{
				LI:       0x05,
				DTROA:    0xAB,
				NREOT:    0xCD,
				UserData: []byte{0xEF},
			},
			expected: []byte{0x05, 0xAB, 0xCD, 0xEF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.data.Serialize()
			require.Equal(t, tt.expected, actual)
		})
	}
}

func Test_Data_Deserialize(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expected    Data
		expectedErr error
	}{
		{
			name:  "valid data PDU",
			input: []byte{0x02, 0xF0, 0x80, 0x01, 0x02, 0x03},
			expected: Data{
				LI:    0x02,
				DTROA: 0xF0,
				NREOT: 0x80,
			},
			expectedErr: nil,
		},
		{
			name:  "valid minimal",
			input: []byte{0x02, 0xF0, 0x80},
			expected: Data{
				LI:    0x02,
				DTROA: 0xF0,
				NREOT: 0x80,
			},
			expectedErr: nil,
		},
		{
			name:        "empty input",
			input:       []byte{},
			expected:    Data{},
			expectedErr: io.EOF,
		},
		{
			name:        "wrong data length",
			input:       []byte{0x03, 0xF0, 0x80},
			expected:    Data{LI: 0x03},
			expectedErr: ErrWrongDataLength,
		},
		{
			name:        "wrong data length - zero",
			input:       []byte{0x00, 0xF0, 0x80},
			expected:    Data{LI: 0x00},
			expectedErr: ErrWrongDataLength,
		},
		{
			name:        "truncated after LI",
			input:       []byte{0x02},
			expected:    Data{LI: 0x02},
			expectedErr: io.EOF,
		},
		{
			name:        "truncated after DTROA",
			input:       []byte{0x02, 0xF0},
			expected:    Data{LI: 0x02, DTROA: 0xF0},
			expectedErr: io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actual Data
			input := bytes.NewBuffer(tt.input)
			err := actual.Deserialize(input)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, actual)
			}
		})
	}
}

// =============================================================================
// Protocol Tests
// =============================================================================

func Test_New(t *testing.T) {
	// New() requires a *tpkt.Protocol, but we can test that it returns non-nil
	p := New(nil)
	require.NotNil(t, p)
}

func Test_NewWithConn(t *testing.T) {
	mock := &mockTpktConn{}
	p := NewWithConn(mock)
	require.NotNil(t, p)
}

// =============================================================================
// Protocol.Connect() Tests
// =============================================================================

func Test_Protocol_Connect(t *testing.T) {
	tests := []struct {
		name         string
		userData     []byte
		mockSetup    func(*mockTpktConn)
		expectedErr  string
		validateSent func(*testing.T, []byte)
	}{
		{
			name:     "successful connection",
			userData: []byte{0x01, 0x02, 0x03},
			mockSetup: func(m *mockTpktConn) {
				m.sendErr = nil
				m.receiveData = bytes.NewBuffer([]byte{
					0x0e, 0xd0, 0x00, 0x00,
					0x12, 0x34, 0x00, 0x02,
					0x00, 0x08, 0x00, 0x00,
					0x00, 0x00, 0x00,
				})
				m.receiveErr = nil
			},
			expectedErr: "",
			validateSent: func(t *testing.T, sent []byte) {
				// Verify the sent data is a valid ConnectionRequest
				require.True(t, len(sent) > 6)
				require.Equal(t, byte(0xE0), sent[1]) // CRCDT
			},
		},
		{
			name:     "send error",
			userData: []byte{0x01, 0x02},
			mockSetup: func(m *mockTpktConn) {
				m.sendErr = errors.New("send failed")
			},
			expectedErr: "client connection request: send failed",
		},
		{
			name:     "receive error",
			userData: []byte{0x01},
			mockSetup: func(m *mockTpktConn) {
				m.sendErr = nil
				m.receiveErr = errors.New("receive failed")
			},
			expectedErr: "recieve connection response: receive failed",
		},
		{
			name:     "invalid response - wrong code",
			userData: nil,
			mockSetup: func(m *mockTpktConn) {
				m.sendErr = nil
				m.receiveData = bytes.NewBuffer([]byte{0x0e, 0xE0, 0x00, 0x00, 0x12, 0x34, 0x00})
				m.receiveErr = nil
			},
			expectedErr: "server connection confirm: wrong connection confirm code",
		},
		{
			name:     "invalid response - wrong length",
			userData: nil,
			mockSetup: func(m *mockTpktConn) {
				m.sendErr = nil
				m.receiveData = bytes.NewBuffer([]byte{0x05, 0xd0})
				m.receiveErr = nil
			},
			expectedErr: "server connection confirm: small connection confirm length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockTpktConn{}
			tt.mockSetup(mock)

			p := NewWithConn(mock)
			reader, err := p.Connect(tt.userData)

			if tt.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr)
				require.Nil(t, reader)
			} else {
				require.NoError(t, err)
				require.NotNil(t, reader)
			}

			if tt.validateSent != nil && mock.sendData != nil {
				tt.validateSent(t, mock.sendData)
			}
		})
	}
}

// =============================================================================
// Protocol.Send() Tests
// =============================================================================

func Test_Protocol_Send(t *testing.T) {
	tests := []struct {
		name        string
		userData    []byte
		sendErr     error
		expectErr   bool
		validatePDU func(*testing.T, []byte)
	}{
		{
			name:      "successful send with data",
			userData:  []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			sendErr:   nil,
			expectErr: false,
			validatePDU: func(t *testing.T, pdu []byte) {
				require.Equal(t, byte(0x02), pdu[0]) // LI
				require.Equal(t, byte(0xF0), pdu[1]) // DTROA
				require.Equal(t, byte(0x80), pdu[2]) // NREOT
				require.Equal(t, []byte{0x01, 0x02, 0x03, 0x04, 0x05}, pdu[3:])
			},
		},
		{
			name:      "successful send empty data",
			userData:  nil,
			sendErr:   nil,
			expectErr: false,
			validatePDU: func(t *testing.T, pdu []byte) {
				require.Len(t, pdu, 3)
				require.Equal(t, byte(0x02), pdu[0])
				require.Equal(t, byte(0xF0), pdu[1])
				require.Equal(t, byte(0x80), pdu[2])
			},
		},
		{
			name:      "successful send large data",
			userData:  bytes.Repeat([]byte{0xAA}, 1000),
			sendErr:   nil,
			expectErr: false,
			validatePDU: func(t *testing.T, pdu []byte) {
				require.Len(t, pdu, 1003)
			},
		},
		{
			name:      "send error",
			userData:  []byte{0x01},
			sendErr:   errors.New("network error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockTpktConn{sendErr: tt.sendErr}
			p := NewWithConn(mock)

			err := p.Send(tt.userData)

			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validatePDU != nil {
					tt.validatePDU(t, mock.sendData)
				}
			}
		})
	}
}

// =============================================================================
// Protocol.Receive() Tests
// =============================================================================

func Test_Protocol_Receive(t *testing.T) {
	tests := []struct {
		name        string
		receiveData []byte
		receiveErr  error
		expectErr   bool
		expectedErr error
	}{
		{
			name:        "successful receive",
			receiveData: []byte{0x02, 0xF0, 0x80, 0x01, 0x02, 0x03},
			receiveErr:  nil,
			expectErr:   false,
		},
		{
			name:        "receive error",
			receiveData: nil,
			receiveErr:  errors.New("connection reset"),
			expectErr:   true,
		},
		{
			name:        "invalid data PDU - wrong LI",
			receiveData: []byte{0x05, 0xF0, 0x80},
			receiveErr:  nil,
			expectErr:   true,
			expectedErr: ErrWrongDataLength,
		},
		{
			name:        "empty data",
			receiveData: []byte{},
			receiveErr:  nil,
			expectErr:   true,
			expectedErr: io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockTpktConn{
				receiveErr: tt.receiveErr,
			}
			if tt.receiveData != nil {
				mock.receiveData = bytes.NewBuffer(tt.receiveData)
			}

			p := NewWithConn(mock)
			reader, err := p.Receive()

			if tt.expectErr {
				require.Error(t, err)
				require.Nil(t, reader)
				if tt.expectedErr != nil {
					require.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, reader)
			}
		})
	}
}

// =============================================================================
// Error Tests
// =============================================================================

func Test_Errors(t *testing.T) {
	// Test that error variables are properly defined
	require.NotNil(t, ErrSmallConnectionConfirmLength)
	require.NotNil(t, ErrWrongDataLength)
	require.NotNil(t, ErrWrongConnectionConfirmCode)

	// Test error messages
	require.Equal(t, "small connection confirm length", ErrSmallConnectionConfirmLength.Error())
	require.Equal(t, "wrong data length", ErrWrongDataLength.Error())
	require.Equal(t, "wrong connection confirm code", ErrWrongConnectionConfirmCode.Error())
}

// =============================================================================
// Integration-style Tests
// =============================================================================

func Test_ConnectionRequest_RoundTrip(t *testing.T) {
	// Test that serialization produces expected wire format
	req := ConnectionRequest{
		CRCDT:       0xE0,
		DSTREF:      0x1234,
		SRCREF:      0x5678,
		ClassOption: 0x00,
		UserData:    []byte("test data"),
	}

	serialized := req.Serialize()

	// Verify LI is calculated correctly: 6 + len(UserData)
	expectedLI := byte(6 + len(req.UserData))
	require.Equal(t, expectedLI, serialized[0])

	// Verify CRCDT
	require.Equal(t, byte(0xE0), serialized[1])

	// Verify DSTREF (big endian)
	require.Equal(t, byte(0x12), serialized[2])
	require.Equal(t, byte(0x34), serialized[3])

	// Verify SRCREF (big endian)
	require.Equal(t, byte(0x56), serialized[4])
	require.Equal(t, byte(0x78), serialized[5])

	// Verify ClassOption
	require.Equal(t, byte(0x00), serialized[6])

	// Verify UserData
	require.Equal(t, []byte("test data"), serialized[7:])
}

func Test_Data_RoundTrip(t *testing.T) {
	// Serialize
	original := Data{
		LI:       0x02,
		DTROA:    0xF0,
		NREOT:    0x80,
		UserData: []byte("payload"),
	}

	serialized := original.Serialize()

	// Deserialize header
	var parsed Data
	err := parsed.Deserialize(bytes.NewBuffer(serialized))
	require.NoError(t, err)

	// Verify header fields match
	require.Equal(t, original.LI, parsed.LI)
	require.Equal(t, original.DTROA, parsed.DTROA)
	require.Equal(t, original.NREOT, parsed.NREOT)
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func Test_ConnectionConfirm_AllValidCodes(t *testing.T) {
	// CCCDT must have upper nibble 0xD (0xD0-0xDF are valid)
	validCodes := []byte{0xD0, 0xD1, 0xD5, 0xDA, 0xDF}

	for _, code := range validCodes {
		t.Run(string([]byte{code}), func(t *testing.T) {
			input := bytes.NewBuffer([]byte{
				0x0e, code, 0x00, 0x00,
				0x12, 0x34, 0x00, 0x02,
				0x00, 0x08, 0x00, 0x00,
				0x00, 0x00, 0x00,
			})
			var cc ConnectionConfirm
			err := cc.Deserialize(input)
			require.NoError(t, err)
			require.Equal(t, code, cc.CCCDT)
		})
	}
}

func Test_ConnectionConfirm_InvalidCodes(t *testing.T) {
	// Codes without 0xD0 upper nibble should fail
	invalidCodes := []byte{0xE0, 0xC0, 0xA0, 0x00, 0xFF}

	for _, code := range invalidCodes {
		t.Run(string([]byte{code}), func(t *testing.T) {
			input := bytes.NewBuffer([]byte{
				0x0e, code, 0x00, 0x00,
				0x12, 0x34, 0x00,
			})
			var cc ConnectionConfirm
			err := cc.Deserialize(input)
			require.ErrorIs(t, err, ErrWrongConnectionConfirmCode)
		})
	}
}

// ============================================================================
// Microsoft Protocol Test Suite Validation Tests
// Reference: MS-RDPBCGR_ClientTestDesignSpecification.md - S1_Connection
// ============================================================================

// TestBVT_ConnectionInitiation_X224Request validates per MS test case:
// "BVT_ConnectionTest_ConnectionInitiation_PositiveTest"
func TestBVT_ConnectionInitiation_X224Request(t *testing.T) {
	// Per MS-RDPBCGR Section 2.2.1.1:
	// Client X.224 Connection Request PDU requirements
	req := ConnectionRequest{
		CRCDT:       0xE0, // Connection Request
		DSTREF:      0,    // Must be 0 per spec
		SRCREF:      0,    // May be 0
		ClassOption: 0,    // Class 0
	}

	data := req.Serialize()

	// Verify TPDU header is correct
	// Per X.224: LI (length indicator), CRCDT, DSTREF, SRCREF, ClassOption
	require.True(t, len(data) >= 7, "X.224 CR PDU minimum size is 7 bytes")
	require.Equal(t, byte(0xE0), data[1], "CRCDT must be 0xE0 for Connection Request")
}

// TestBVT_ConnectionInitiation_X224Confirm validates per MS test case:
// "SUT can process the valid Server X.224 Connection Confirm PDU correctly"
func TestBVT_ConnectionInitiation_X224Confirm(t *testing.T) {
	// Per MS-RDPBCGR Section 2.2.1.2:
	// Server X.224 Connection Confirm PDU
	confirmData := []byte{
		0x0e,       // LI = 14
		0xD0,       // CCCDT (Connection Confirm)
		0x00, 0x00, // DSTREF
		0x12, 0x34, // SRCREF
		0x00,       // Class 0
		// RDP Negotiation Response
		0x02,             // type: TYPE_RDP_NEG_RSP
		0x00,             // flags
		0x08, 0x00,       // length = 8
		0x00, 0x00, 0x00, 0x00, // selectedProtocol = 0 (RDP)
	}

	var cc ConnectionConfirm
	err := cc.Deserialize(bytes.NewBuffer(confirmData))
	require.NoError(t, err)
	require.Equal(t, byte(0xD0), cc.CCCDT)
}

// TestS1_Connection_NegotiationFlags validates per MS-RDPBCGR Section 2.2.1.2.1
func TestS1_Connection_NegotiationFlags(t *testing.T) {
	// RDP Negotiation Response flags
	tests := []struct {
		name        string
		flags       byte
		description string
	}{
		{"NoFlags", 0x00, "No special flags"},
		{"EXTENDED_CLIENT_DATA_SUPPORTED", 0x01, "Server supports extended client data"},
		{"DYNVC_GFX_PROTOCOL_SUPPORTED", 0x02, "Server supports GFX protocol"},
		{"RDP_NEGRSP_RESERVED", 0x04, "Reserved flag"},
		{"RESTRICTED_ADMIN_MODE_SUPPORTED", 0x08, "Restricted admin mode"},
		{"REDIRECTED_AUTHENTICATION_MODE_SUPPORTED", 0x10, "Redirected auth"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			confirmData := []byte{
				0x0e, 0xD0, 0x00, 0x00, 0x12, 0x34, 0x00,
				0x02,                   // TYPE_RDP_NEG_RSP
				tc.flags,               // flags
				0x08, 0x00,             // length
				0x00, 0x00, 0x00, 0x00, // protocol
			}

			var cc ConnectionConfirm
			err := cc.Deserialize(bytes.NewBuffer(confirmData))
			require.NoError(t, err, "Failed to parse with flags: %s", tc.description)
		})
	}
}

// TestS1_Connection_ProtocolSelection validates protocol selection per spec
func TestS1_Connection_ProtocolSelection(t *testing.T) {
	// Per MS-RDPBCGR Section 2.2.1.1.1
	protocols := []struct {
		value uint32
		name  string
	}{
		{0x00000000, "PROTOCOL_RDP"},
		{0x00000001, "PROTOCOL_SSL"},
		{0x00000002, "PROTOCOL_HYBRID"},
		{0x00000004, "PROTOCOL_RDSTLS"},
		{0x00000008, "PROTOCOL_HYBRID_EX"},
		{0x00000010, "PROTOCOL_RDSAAD"},
	}

	for _, p := range protocols {
		t.Run(p.name, func(t *testing.T) {
			confirmData := make([]byte, 15)
			confirmData[0] = 0x0e
			confirmData[1] = 0xD0
			confirmData[7] = 0x02 // TYPE_RDP_NEG_RSP
			confirmData[9] = 0x08
			// Set protocol in little-endian
			confirmData[11] = byte(p.value)
			confirmData[12] = byte(p.value >> 8)
			confirmData[13] = byte(p.value >> 16)
			confirmData[14] = byte(p.value >> 24)

			var cc ConnectionConfirm
			err := cc.Deserialize(bytes.NewBuffer(confirmData))
			require.NoError(t, err, "Failed for protocol %s", p.name)
		})
	}
}
