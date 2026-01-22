package pdu

import (
	"bytes"
	"encoding/binary"
	"io"
)

// MessageType represents the type of synchronization message.
type MessageType uint16

const (
	// MessageTypeSync indicates a synchronization message.
	MessageTypeSync MessageType = 1
)

// SynchronizePDUData represents the TS_SYNCHRONIZE_PDU structure (MS-RDPBCGR 2.2.1.14).
type SynchronizePDUData struct {
	MessageType MessageType
}

// NewSynchronize creates a new Client Synchronize PDU (MS-RDPBCGR 2.2.1.14).
func NewSynchronize(shareID uint32, userId uint16) *Data {
	return &Data{
		ShareDataHeader: *newShareDataHeader(shareID, userId, TypeData, Type2Synchronize),
		SynchronizePDUData: &SynchronizePDUData{
			MessageType: MessageTypeSync,
		},
	}
}

// ServerChannelID is the default MCS channel ID for the server (IO Channel).
const ServerChannelID uint16 = 1002

// Serialize encodes the PDU data to wire format.
func (pdu *SynchronizePDUData) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, uint16(pdu.MessageType))
	_ = binary.Write(buf, binary.LittleEndian, ServerChannelID) // targetUser

	return buf.Bytes()
}

// Deserialize decodes the PDU data from wire format.
func (pdu *SynchronizePDUData) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &pdu.MessageType)
	if err != nil {
		return err
	}

	var targetUser uint16
	err = binary.Read(wire, binary.LittleEndian, &targetUser)
	if err != nil {
		return err
	}

	return nil
}

// ControlAction represents the action field in a Control PDU (MS-RDPBCGR 2.2.1.15).
type ControlAction uint16

const (
	// ControlActionRequestControl CTRLACTION_REQUEST_CONTROL
	ControlActionRequestControl ControlAction = 0x0001

	// ControlActionGrantedControl CTRLACTION_GRANTED_CONTROL
	ControlActionGrantedControl ControlAction = 0x0002

	// ControlActionDetach CTRLACTION_DETACH
	ControlActionDetach ControlAction = 0x0003

	// ControlActionCooperate CTRLACTION_COOPERATE
	ControlActionCooperate ControlAction = 0x0004
)

// ControlPDUData represents the TS_CONTROL_PDU structure (MS-RDPBCGR 2.2.1.15).
type ControlPDUData struct {
	Action    ControlAction
	GrantID   uint16
	ControlID uint32
}

// NewControl creates a new Client Control PDU (MS-RDPBCGR 2.2.1.15).
func NewControl(shareID uint32, userId uint16, action ControlAction) *Data {
	return &Data{
		ShareDataHeader: *newShareDataHeader(shareID, userId, TypeData, Type2Control),
		ControlPDUData: &ControlPDUData{
			Action: action,
		},
	}
}

// Serialize encodes the PDU data to wire format.
func (pdu *ControlPDUData) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, uint16(pdu.Action))
	_ = binary.Write(buf, binary.LittleEndian, pdu.GrantID)
	_ = binary.Write(buf, binary.LittleEndian, pdu.ControlID)

	return buf.Bytes()
}

// Deserialize decodes the PDU data from wire format.
func (pdu *ControlPDUData) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &pdu.Action)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &pdu.GrantID)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &pdu.ControlID)
	if err != nil {
		return err
	}

	return nil
}

// FontListPDUData represents the TS_FONT_LIST_PDU structure (MS-RDPBCGR 2.2.1.18).
type FontListPDUData struct{}

// NewFontList creates a new Client Font List PDU (MS-RDPBCGR 2.2.1.18).
func NewFontList(shareID uint32, userId uint16) *Data {
	return &Data{
		ShareDataHeader: *newShareDataHeader(shareID, userId, TypeData, Type2Fontlist),
		FontListPDUData: &FontListPDUData{},
	}
}

// Serialize encodes the PDU data to wire format.
func (pdu *FontListPDUData) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0000)) // numberFonts
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0000)) // totalNumFonts
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0003)) // listFlags = FONTLIST_FIRST | FONTLIST_LAST
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0032)) // entrySize

	return buf.Bytes()
}

// FontMapPDUData represents the TS_FONT_MAP_PDU structure (MS-RDPBCGR 2.2.1.22).
type FontMapPDUData struct{}

// Deserialize decodes the PDU data from wire format.
func (pdu *FontMapPDUData) Deserialize(wire io.Reader) error {
	var (
		numberEntries   uint16
		totalNumEntries uint16
		mapFlags        uint16
		entrySize       uint16
		err             error
	)

	err = binary.Read(wire, binary.LittleEndian, &numberEntries)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &totalNumEntries)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &mapFlags)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &entrySize)
	if err != nil {
		return err
	}

	return nil
}
