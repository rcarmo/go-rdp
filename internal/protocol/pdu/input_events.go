package pdu

import (
	"bytes"
	"encoding/binary"
)

// EventCode represents a fast-path input event code (MS-RDPBCGR 2.2.8.1.2.2).
type EventCode uint8

const (
	//EventCodeScanCode FASTPATH_INPUT_EVENT_SCANCODE
	EventCodeScanCode EventCode = 0

	//EventCodeMouse FASTPATH_INPUT_EVENT_MOUSE
	EventCodeMouse EventCode = 1

	//EventCodeMouseX FASTPATH_INPUT_EVENT_MOUSEX
	EventCodeMouseX EventCode = 2

	//EventCodeSync FASTPATH_INPUT_EVENT_SYNC
	EventCodeSync EventCode = 3

	//EventCodeUnicode FASTPATH_INPUT_EVENT_UNICODE
	EventCodeUnicode EventCode = 4

	//EventCodeQoETimestamp FASTPATH_INPUT_EVENT_QOE_TIMESTAMP
	EventCodeQoETimestamp EventCode = 6
)

// InputEvent represents a fast-path input event (MS-RDPBCGR 2.2.8.1.2.2).
type InputEvent struct {
	EventFlags           uint8
	EventCode            EventCode
	keyboardEvent        *keyboardEvent
	unicodeKeyboardEvent *unicodeKeyboardEvent
	mouseEvent           *mouseEvent
	extendedMouseEvent   *extendedMouseEvent
	qualityOfExperience  *qualityOfExperience
}

// Serialize encodes the input event to wire format.
func (e *InputEvent) Serialize() []byte {
	buf := new(bytes.Buffer)

	// event flags in higher 5 bits
	// event code in lower 3 bits
	header := (e.EventFlags&0x1f)<<3 | uint8(e.EventCode)&0x7

	var data []byte

	switch e.EventCode {
	case EventCodeScanCode:
		data = e.keyboardEvent.Serialize()
	case EventCodeUnicode:
		data = e.unicodeKeyboardEvent.Serialize()
	case EventCodeMouse:
		data = e.mouseEvent.Serialize()
	case EventCodeMouseX:
		data = e.extendedMouseEvent.Serialize()
	case EventCodeSync: // do nothing
	case EventCodeQoETimestamp:
		data = e.qualityOfExperience.Serialize()
	}

	_ = binary.Write(buf, binary.LittleEndian, header)
	buf.Write(data)

	return buf.Bytes()
}

type keyboardEvent struct {
	KeyCode uint8
}

const (
	// KBDFlagsRelease FASTPATH_INPUT_KBDFLAGS_RELEASE
	KBDFlagsRelease uint8 = 0x01

	// KBDFlagsExtended FASTPATH_INPUT_KBDFLAGS_EXTENDED
	KBDFlagsExtended uint8 = 0x02

	// KBDFlagsExtended1 FASTPATH_INPUT_KBDFLAGS_EXTENDED1
	KBDFlagsExtended1 uint8 = 0x04
)

// NewKeyboardEvent creates a new keyboard scancode input event.
func NewKeyboardEvent(flags uint8, keyCode uint8) *InputEvent {
	return &InputEvent{
		EventFlags: flags,
		EventCode:  EventCodeScanCode,
		keyboardEvent: &keyboardEvent{
			KeyCode: keyCode,
		},
	}
}

func (e *keyboardEvent) Serialize() []byte {
	return []byte{e.KeyCode}
}

type unicodeKeyboardEvent struct {
	UnicodeCode uint16
}

// NewUnicodeKeyboardEvent creates a new Unicode keyboard input event.
func NewUnicodeKeyboardEvent(unicodeCode uint16) *InputEvent {
	return &InputEvent{
		EventFlags: KBDFlagsRelease,
		EventCode:  EventCodeUnicode,
		unicodeKeyboardEvent: &unicodeKeyboardEvent{
			UnicodeCode: unicodeCode,
		},
	}
}

func (e *unicodeKeyboardEvent) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, e.UnicodeCode)

	return buf.Bytes()
}

const (
	PTRFlagsHWheel        uint16 = 0x0400
	PTRFlagsWheel         uint16 = 0x0200
	PTRFlagsWheelNegative uint16 = 0x0100
	PTRFlagsMove          uint16 = 0x0800
	PTRFlagsDown          uint16 = 0x8000
	PTRFlagsButton1       uint16 = 0x1000
	PTRFlagsButton2       uint16 = 0x2000
	PTRFlagsButton3       uint16 = 0x4000
)

type mouseEvent struct {
	pointerFlags uint16
	xPos         uint16
	yPos         uint16
}

// NewMouseEvent creates a new mouse input event.
func NewMouseEvent(pointerFlags, xPos, yPos uint16) *InputEvent {
	return &InputEvent{
		EventCode: EventCodeMouse,
		mouseEvent: &mouseEvent{
			pointerFlags: pointerFlags,
			xPos:         xPos,
			yPos:         yPos,
		},
	}
}

func (e *mouseEvent) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, e.pointerFlags)
	_ = binary.Write(buf, binary.LittleEndian, e.xPos)
	_ = binary.Write(buf, binary.LittleEndian, e.yPos)

	return buf.Bytes()
}

const (
	PTRXFlagsDown    uint16 = 0x8000
	PTRXFlagsButton1 uint16 = 0x0001
	PTRXFlagsButton2 uint16 = 0x0002
)

type extendedMouseEvent struct {
	pointerFlags uint16
	xPos         uint16
	yPos         uint16
}

// NewExtendedMouseEvent creates a new extended mouse input event for X buttons.
func NewExtendedMouseEvent(pointerFlags, xPos, yPos uint16) *InputEvent {
	return &InputEvent{
		EventCode: EventCodeMouseX,
		extendedMouseEvent: &extendedMouseEvent{
			pointerFlags: pointerFlags,
			xPos:         xPos,
			yPos:         yPos,
		},
	}
}

func (e *extendedMouseEvent) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, e.pointerFlags)
	_ = binary.Write(buf, binary.LittleEndian, e.xPos)
	_ = binary.Write(buf, binary.LittleEndian, e.yPos)

	return buf.Bytes()
}

const (
	SyncScrollLock uint8 = 0x01
	SyncNumLock    uint8 = 0x02
	SyncCapsLock   uint8 = 0x04
	SyncKanaLock   uint8 = 0x08
)

// NewSynchronizeEvent creates a new synchronize input event for lock key state.
func NewSynchronizeEvent(eventFlags uint8) *InputEvent {
	return &InputEvent{
		EventFlags: eventFlags,
		EventCode:  EventCodeSync,
	}
}

type qualityOfExperience struct {
	timestamp uint32
}

// NewQualityOfExperienceEvent creates a new QoE timestamp input event.
func NewQualityOfExperienceEvent(timestamp uint32) *InputEvent {
	return &InputEvent{
		EventCode: EventCodeQoETimestamp,
		qualityOfExperience: &qualityOfExperience{
			timestamp: timestamp,
		},
	}
}

func (e *qualityOfExperience) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, e.timestamp)

	return buf.Bytes()
}
