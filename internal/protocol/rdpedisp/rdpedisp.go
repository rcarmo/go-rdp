// Package rdpedisp implements the Display Control Virtual Channel Extension (MS-RDPEDISP).
// This protocol allows dynamic display resolution changes without reconnecting.
package rdpedisp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Dynamic channel name for display control
const ChannelName = "Microsoft::Windows::RDS::DisplayControl"

// PDU types (MS-RDPEDISP 2.2.2)
const (
	PDUTypeCaps          uint32 = 0x00000005 // CYCNVC_CAPS_PDU
	PDUTypeMonitorLayout uint32 = 0x00000002 // DISPLAYCONTROL_MONITOR_LAYOUT_PDU
)

// Monitor flags
const (
	MonitorFlagPrimary uint32 = 0x00000001
)

// Orientation values
const (
	OrientationLandscape        uint32 = 0
	OrientationPortrait         uint32 = 90
	OrientationLandscapeFlipped uint32 = 180
	OrientationPortraitFlipped  uint32 = 270
)

// CapsPDU represents DISPLAYCONTROL_CAPS_PDU (MS-RDPEDISP 2.2.2.1)
// Sent by server to client after channel is created
type CapsPDU struct {
	MaxNumMonitors     uint32 // Maximum number of monitors supported
	MaxMonitorAreaSize uint32 // Maximum total monitor area in pixels (width * height)
}

// Serialize encodes CapsPDU to wire format
func (c *CapsPDU) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, PDUTypeCaps)
	_ = binary.Write(buf, binary.LittleEndian, uint32(12)) // Length
	_ = binary.Write(buf, binary.LittleEndian, c.MaxNumMonitors)
	_ = binary.Write(buf, binary.LittleEndian, c.MaxMonitorAreaSize)

	return buf.Bytes()
}

// Deserialize decodes CapsPDU from wire format
func (c *CapsPDU) Deserialize(r io.Reader) error {
	var pduType, length uint32

	if err := binary.Read(r, binary.LittleEndian, &pduType); err != nil {
		return fmt.Errorf("caps pdu type: %w", err)
	}
	if pduType != PDUTypeCaps {
		return fmt.Errorf("unexpected PDU type: 0x%08X (expected 0x%08X)", pduType, PDUTypeCaps)
	}

	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return fmt.Errorf("caps length: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &c.MaxNumMonitors); err != nil {
		return fmt.Errorf("caps max monitors: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &c.MaxMonitorAreaSize); err != nil {
		return fmt.Errorf("caps max area: %w", err)
	}

	return nil
}

// MonitorDef represents DISPLAYCONTROL_MONITOR_LAYOUT (MS-RDPEDISP 2.2.2.2.1)
type MonitorDef struct {
	Flags              uint32 // MonitorFlagPrimary if primary monitor
	Left               int32  // X position of top-left corner
	Top                int32  // Y position of top-left corner
	Width              uint32 // Width in pixels
	Height             uint32 // Height in pixels
	PhysicalWidth      uint32 // Physical width in millimeters
	PhysicalHeight     uint32 // Physical height in millimeters
	Orientation        uint32 // Rotation: 0, 90, 180, or 270
	DesktopScaleFactor uint32 // Desktop scale factor (100-500)
	DeviceScaleFactor  uint32 // Device scale factor (100, 140, or 180)
}

// Serialize encodes MonitorDef to wire format
func (m *MonitorDef) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, m.Flags)
	_ = binary.Write(buf, binary.LittleEndian, m.Left)
	_ = binary.Write(buf, binary.LittleEndian, m.Top)
	_ = binary.Write(buf, binary.LittleEndian, m.Width)
	_ = binary.Write(buf, binary.LittleEndian, m.Height)
	_ = binary.Write(buf, binary.LittleEndian, m.PhysicalWidth)
	_ = binary.Write(buf, binary.LittleEndian, m.PhysicalHeight)
	_ = binary.Write(buf, binary.LittleEndian, m.Orientation)
	_ = binary.Write(buf, binary.LittleEndian, m.DesktopScaleFactor)
	_ = binary.Write(buf, binary.LittleEndian, m.DeviceScaleFactor)

	return buf.Bytes()
}

// Deserialize decodes MonitorDef from wire format
func (m *MonitorDef) Deserialize(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &m.Flags); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &m.Left); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &m.Top); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &m.Width); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &m.Height); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &m.PhysicalWidth); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &m.PhysicalHeight); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &m.Orientation); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &m.DesktopScaleFactor); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &m.DeviceScaleFactor); err != nil {
		return err
	}
	return nil
}

// MonitorLayoutPDU represents DISPLAYCONTROL_MONITOR_LAYOUT_PDU (MS-RDPEDISP 2.2.2.2)
// Sent by client to server to request display reconfiguration
type MonitorLayoutPDU struct {
	Monitors []MonitorDef
}

// MonitorLayoutSize is the fixed size of DISPLAYCONTROL_MONITOR_LAYOUT structure (40 bytes)
const MonitorLayoutSize = 40

// Serialize encodes MonitorLayoutPDU to wire format
// Per MS-RDPEDISP 2.2.2.2 and FreeRDP disp_main.c:disp_send_display_control_monitor_layout_pdu
func (m *MonitorLayoutPDU) Serialize() []byte {
	buf := new(bytes.Buffer)

	monitorCount := uint32(len(m.Monitors))
	// Header is 8 bytes (Type + Length), then 8 bytes (MonitorLayoutSize + NumMonitors), then monitors
	length := uint32(8 + 8 + monitorCount*MonitorLayoutSize)

	_ = binary.Write(buf, binary.LittleEndian, PDUTypeMonitorLayout)
	_ = binary.Write(buf, binary.LittleEndian, length)
	_ = binary.Write(buf, binary.LittleEndian, uint32(MonitorLayoutSize)) // MonitorLayoutSize field
	_ = binary.Write(buf, binary.LittleEndian, monitorCount)

	for i := range m.Monitors {
		// FreeRDP enforces constraints: Width must be even, min 200, max 8192
		mon := m.Monitors[i]
		mon.Width -= mon.Width % 2 // Make even
		if mon.Width < 200 {
			mon.Width = 200
		}
		if mon.Width > 8192 {
			mon.Width = 8192
		}
		if mon.Height < 200 {
			mon.Height = 200
		}
		if mon.Height > 8192 {
			mon.Height = 8192
		}
		buf.Write(mon.Serialize())
	}

	return buf.Bytes()
}

// Deserialize decodes MonitorLayoutPDU from wire format
func (m *MonitorLayoutPDU) Deserialize(r io.Reader) error {
	var pduType, length, monitorLayoutSize, monitorCount uint32

	if err := binary.Read(r, binary.LittleEndian, &pduType); err != nil {
		return fmt.Errorf("layout pdu type: %w", err)
	}
	if pduType != PDUTypeMonitorLayout {
		return fmt.Errorf("unexpected PDU type: 0x%08X (expected 0x%08X)", pduType, PDUTypeMonitorLayout)
	}

	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return fmt.Errorf("layout length: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &monitorLayoutSize); err != nil {
		return fmt.Errorf("monitor layout size: %w", err)
	}
	
	if monitorLayoutSize != MonitorLayoutSize {
		return fmt.Errorf("unexpected monitor layout size: %d (expected %d)", monitorLayoutSize, MonitorLayoutSize)
	}

	if err := binary.Read(r, binary.LittleEndian, &monitorCount); err != nil {
		return fmt.Errorf("layout monitor count: %w", err)
	}

	// Sanity check
	if monitorCount > 64 {
		return fmt.Errorf("too many monitors: %d", monitorCount)
	}

	m.Monitors = make([]MonitorDef, monitorCount)
	for i := uint32(0); i < monitorCount; i++ {
		if err := m.Monitors[i].Deserialize(r); err != nil {
			return fmt.Errorf("monitor %d: %w", i, err)
		}
	}

	return nil
}

// NewSingleMonitorLayout creates a MonitorLayoutPDU for a single primary monitor
func NewSingleMonitorLayout(width, height uint32) *MonitorLayoutPDU {
	return &MonitorLayoutPDU{
		Monitors: []MonitorDef{
			{
				Flags:              MonitorFlagPrimary,
				Left:               0,
				Top:                0,
				Width:              width,
				Height:             height,
				PhysicalWidth:      0, // Let server calculate
				PhysicalHeight:     0, // Let server calculate
				Orientation:        OrientationLandscape,
				DesktopScaleFactor: 100,
				DeviceScaleFactor:  100,
			},
		},
	}
}

// ParsePDU determines the PDU type from the first 4 bytes
func ParsePDUType(data []byte) (uint32, error) {
	if len(data) < 4 {
		return 0, fmt.Errorf("data too short for PDU type")
	}
	return binary.LittleEndian.Uint32(data[:4]), nil
}
