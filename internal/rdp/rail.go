package rdp

// RAIL (Remote Application Integrated Locally) - NOT SUPPORTED
//
// RAIL allows running individual Windows applications remotely while they
// appear as local windows. This feature is NOT SUPPORTED in the HTML5 client
// because browsers cannot create native OS windows that integrate with the
// desktop taskbar and window manager.
//
// The code below contains partial PDU structures for protocol compatibility,
// but the feature is non-functional. Use full desktop sessions instead.

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/rcarmo/go-rdp/internal/codec"
)

// RailState represents the current state of the RAIL protocol state machine.
type RailState uint8

// RailState constants define the RAIL protocol state machine states.
const (
	// RailStateUninitialized indicates RAIL is not initialized.
	RailStateUninitialized RailState = iota
	// RailStateInitializing indicates RAIL handshake is in progress.
	RailStateInitializing
	// RailStateSyncDesktop indicates desktop synchronization phase.
	RailStateSyncDesktop
	// RailStateWaitForData indicates waiting for server data.
	RailStateWaitForData
	// RailStateExecuteApp indicates application execution phase.
	RailStateExecuteApp
)

func (c *Client) handleRail(wire io.Reader) error {
	if c.remoteApp == nil {
		return nil
	}

	var (
		input RailPDU
		err   error
	)

	if err = input.Deserialize(wire); err != nil {
		return err
	}

	if input.header.OrderType == RailOrderSysParam {
		return nil
	}

	switch c.railState {
	case RailStateInitializing:
		return c.railHandshake(&input)
	case RailStateExecuteApp:
		return c.railReceiveRemoteAppStatus(&input)
	}

	return nil
}

// RailOrder represents a RAIL PDU order type as defined in MS-RDPERP.
type RailOrder uint16

const (
	// RailOrderExec TS_RAIL_ORDER_EXEC
	RailOrderExec RailOrder = 0x0001

	// RailOrderActivate TS_RAIL_ORDER_ACTIVATE
	RailOrderActivate RailOrder = 0x0002

	// RailOrderSysParam TS_RAIL_ORDER_SYSPARAM
	RailOrderSysParam RailOrder = 0x0003

	// RailOrderSysCommand TS_RAIL_ORDER_SYSCOMMAND
	RailOrderSysCommand RailOrder = 0x0004

	// RailOrderHandshake TS_RAIL_ORDER_HANDSHAKE
	RailOrderHandshake RailOrder = 0x0005

	// RailOrderNotifyEvent TS_RAIL_ORDER_NOTIFY_EVENT
	RailOrderNotifyEvent RailOrder = 0x0006

	// RailOrderWindowMove TS_RAIL_ORDER_WINDOWMOVE
	RailOrderWindowMove RailOrder = 0x0008

	// RailOrderLocalMoveSize TS_RAIL_ORDER_LOCALMOVESIZE
	RailOrderLocalMoveSize RailOrder = 0x0009

	// RailOrderMinMaxInfo TS_RAIL_ORDER_MINMAXINFO
	RailOrderMinMaxInfo RailOrder = 0x000a

	// RailOrderClientStatus TS_RAIL_ORDER_CLIENTSTATUS
	RailOrderClientStatus RailOrder = 0x000b

	// RailOrderSysMenu TS_RAIL_ORDER_SYSMENU
	RailOrderSysMenu RailOrder = 0x000c

	// RailOrderLangBarInfo TS_RAIL_ORDER_LANGBARINFO
	RailOrderLangBarInfo RailOrder = 0x000d

	// RailOrderExecResult TS_RAIL_ORDER_EXEC_RESULT
	RailOrderExecResult RailOrder = 0x0080

	// RailOrderGetAppIDReq TS_RAIL_ORDER_GET_APPID_REQ
	RailOrderGetAppIDReq RailOrder = 0x000E

	// RailOrderAppIDResp TS_RAIL_ORDER_GET_APPID_RESP
	RailOrderAppIDResp RailOrder = 0x000F

	// RailOrderTaskBarInfo TS_RAIL_ORDER_TASKBARINFO
	RailOrderTaskBarInfo RailOrder = 0x0010

	// RailOrderLanguageIMEInfo TS_RAIL_ORDER_LANGUAGEIMEINFO
	RailOrderLanguageIMEInfo RailOrder = 0x0011

	// RailOrderCompartmentInfo TS_RAIL_ORDER_COMPARTMENTINFO
	RailOrderCompartmentInfo RailOrder = 0x0012

	// RailOrderHandshakeEx TS_RAIL_ORDER_HANDSHAKE_EX
	RailOrderHandshakeEx RailOrder = 0x0013

	// RailOrderZOrderSync TS_RAIL_ORDER_ZORDER_SYNC
	RailOrderZOrderSync RailOrder = 0x0014

	// RailOrderCloak TS_RAIL_ORDER_CLOAK
	RailOrderCloak RailOrder = 0x0015

	// RailOrderPowerDisplayRequest TS_RAIL_ORDER_POWER_DISPLAY_REQUEST
	RailOrderPowerDisplayRequest RailOrder = 0x0016

	// RailOrderSnapArrange TS_RAIL_ORDER_SNAP_ARRANGE
	RailOrderSnapArrange RailOrder = 0x0017

	// RailOrderGetAppIDRespEx TS_RAIL_ORDER_GET_APPID_RESP_EX
	RailOrderGetAppIDRespEx RailOrder = 0x0018

	// RailOrderTextScaleInfo TS_RAIL_ORDER_TEXTSCALEINFO
	RailOrderTextScaleInfo RailOrder = 0x0019

	// RailOrderCaretBlinkInfo TS_RAIL_ORDER_CARETBLINKINFO
	RailOrderCaretBlinkInfo RailOrder = 0x001A
)

// RailPDU represents a RAIL protocol data unit.
type RailPDU struct {
	channelHeader                  ChannelPDUHeader
	header                         RailPDUHeader
	RailPDUHandshake               *RailPDUHandshake
	RailPDUClientInfo              *RailPDUClientInfo
	RailPDUClientExecute           *RailPDUClientExecute
	RailPDUSystemParameters        *RailPDUSystemParameters
	RailPDUExecResult              *RailPDUExecResult
	RailPDUClientSystemParamUpdate *RailPDUClientSystemParamUpdate
}

// Serialize encodes the RailPDU to wire format.
func (pdu *RailPDU) Serialize() []byte {
	var data []byte

	switch pdu.header.OrderType {
	case RailOrderHandshake:
		data = pdu.RailPDUHandshake.Serialize()
	case RailOrderExec:
		data = pdu.RailPDUClientExecute.Serialize()
	case RailOrderSysParam:
		data = pdu.RailPDUClientSystemParamUpdate.Serialize()
	}

	pdu.header.OrderLength = uint16(8 + 4 + len(data))

	buf := new(bytes.Buffer)

	buf.Write(pdu.channelHeader.Serialize())
	buf.Write(pdu.header.Serialize())
	buf.Write(data)

	return buf.Bytes()
}

// Deserialize decodes a RailPDU from wire format.
func (pdu *RailPDU) Deserialize(wire io.Reader) error {
	var err error

	err = pdu.channelHeader.Deserialize(wire)
	if err != nil {
		return err
	}

	err = pdu.header.Deserialize(wire)
	if err != nil {
		return err
	}

	switch pdu.header.OrderType {
	case RailOrderHandshake:
		pdu.RailPDUHandshake = &RailPDUHandshake{}

		return pdu.RailPDUHandshake.Deserialize(wire)
	case RailOrderSysParam:
		pdu.RailPDUSystemParameters = &RailPDUSystemParameters{}

		return pdu.RailPDUSystemParameters.Deserialize(wire)
	case RailOrderExecResult:
		pdu.RailPDUExecResult = &RailPDUExecResult{}

		return pdu.RailPDUExecResult.Deserialize(wire)
	}

	return nil
}

// RailPDUHeader contains the header fields for a RAIL PDU.
type RailPDUHeader struct {
	OrderType   RailOrder
	OrderLength uint16
}

// Serialize encodes the RailPDUHeader to wire format.
func (h *RailPDUHeader) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, uint16(h.OrderType))
	_ = binary.Write(buf, binary.LittleEndian, h.OrderLength)

	return buf.Bytes()
}

// Deserialize decodes a RailPDUHeader from wire format.
func (h *RailPDUHeader) Deserialize(wire io.Reader) error {
	var err error

	var orderType uint16
	err = binary.Read(wire, binary.LittleEndian, &orderType)
	if err != nil {
		return err
	}
	h.OrderType = RailOrder(orderType)

	err = binary.Read(wire, binary.LittleEndian, &h.OrderLength)
	if err != nil {
		return err
	}

	return nil
}

// RailPDUHandshake represents a RAIL handshake PDU.
type RailPDUHandshake struct {
	buildNumber uint32
}

// NewRailHandshakePDU creates a new RAIL handshake PDU.
func NewRailHandshakePDU() *RailPDU {
	return &RailPDU{
		channelHeader: ChannelPDUHeader{
			Flags: ChannelFlagFirst | ChannelFlagLast,
		},
		header: RailPDUHeader{
			OrderType: RailOrderHandshake,
		},
		RailPDUHandshake: &RailPDUHandshake{
			buildNumber: 0x00001DB0,
		},
	}
}

// Serialize encodes the RailPDUHandshake to wire format.
func (pdu *RailPDUHandshake) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, pdu.buildNumber)

	return buf.Bytes()
}

// Deserialize decodes a RailPDUHandshake from wire format.
func (pdu *RailPDUHandshake) Deserialize(wire io.Reader) error {
	err := binary.Read(wire, binary.LittleEndian, &pdu.buildNumber)
	if err != nil {
		return err
	}

	return nil
}

// RailPDUClientInfo represents a RAIL client info PDU.
type RailPDUClientInfo struct {
	Flags uint32
}

// NewRailClientInfoPDU creates a new RAIL client info PDU.
func NewRailClientInfoPDU() *RailPDU {
	return &RailPDU{
		channelHeader: ChannelPDUHeader{
			Flags: ChannelFlagFirst | ChannelFlagLast,
		},
		header: RailPDUHeader{
			OrderType: RailOrderClientStatus,
		},
		RailPDUClientInfo: &RailPDUClientInfo{
			Flags: 0, // none of the features are supported
		},
	}
}

// Serialize encodes the RailPDUClientInfo to wire format.
func (pdu *RailPDUClientInfo) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, pdu.Flags)

	return buf.Bytes()
}

// RailPDUClientSystemParamUpdate represents a RAIL system parameter update PDU.
type RailPDUClientSystemParamUpdate struct {
	SystemParam uint32
	Body        uint8
}

// NewRailPDUClientSystemParamUpdate creates a new RAIL system parameter update PDU.
func NewRailPDUClientSystemParamUpdate(systemParam uint32, body uint8) *RailPDU {
	return &RailPDU{
		channelHeader: ChannelPDUHeader{
			Flags: ChannelFlagFirst | ChannelFlagLast,
		},
		header: RailPDUHeader{
			OrderType: RailOrderSysParam,
		},
		RailPDUClientSystemParamUpdate: &RailPDUClientSystemParamUpdate{
			SystemParam: systemParam,
			Body:        body,
		},
	}
}

// Serialize encodes the RailPDUClientSystemParamUpdate to wire format.
func (pdu *RailPDUClientSystemParamUpdate) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, pdu.SystemParam)
	_ = binary.Write(buf, binary.LittleEndian, pdu.Body)

	return buf.Bytes()
}

func (c *Client) railHandshake(*RailPDU) error {
	var (
		err error
	)

	clientHandshake := NewRailHandshakePDU()
	err = c.mcsLayer.Send(c.userID, c.channelIDMap["rail"], clientHandshake.Serialize())
	if err != nil {
		return err
	}

	clientInfo := NewRailClientInfoPDU()
	err = c.mcsLayer.Send(c.userID, c.channelIDMap["rail"], clientInfo.Serialize())
	if err != nil {
		return err
	}

	c.railState = RailStateWaitForData

	return c.railStartRemoteApp()
}

// RailPDUSystemParameters represents a RAIL system parameters PDU from the server.
type RailPDUSystemParameters struct {
	SystemParameter uint32
	Body            uint8
}

// Deserialize decodes a RailPDUSystemParameters from wire format.
func (pdu *RailPDUSystemParameters) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &pdu.SystemParameter)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &pdu.Body)
	if err != nil {
		return err
	}

	return nil
}

// RailPDUClientExecute represents a RAIL client execute PDU.
type RailPDUClientExecute struct {
	Flags      uint16
	ExeOrFile  string
	WorkingDir string
	Arguments  string
}

// NewRailClientExecutePDU creates a new RAIL client execute PDU.
func NewRailClientExecutePDU(app, workDir, args string) *RailPDU {
	return &RailPDU{
		channelHeader: ChannelPDUHeader{
			Flags: ChannelFlagFirst | ChannelFlagLast,
		},
		header: RailPDUHeader{
			OrderType: RailOrderExec,
		},
		RailPDUClientExecute: &RailPDUClientExecute{
			ExeOrFile:  app,
			WorkingDir: workDir,
			Arguments:  args,
		},
	}
}

// Serialize encodes the RailPDUClientExecute to wire format.
func (pdu *RailPDUClientExecute) Serialize() []byte {
	exeOrFile := codec.Encode(pdu.ExeOrFile)
	exeOrFileLength := uint16(len(exeOrFile))

	workingDir := codec.Encode(pdu.WorkingDir)
	workingDirLength := uint16(len(workingDir))

	arguments := codec.Encode(pdu.Arguments)
	argumentsLen := uint16(len(arguments))

	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, pdu.Flags)
	_ = binary.Write(buf, binary.LittleEndian, exeOrFileLength)
	_ = binary.Write(buf, binary.LittleEndian, workingDirLength)
	_ = binary.Write(buf, binary.LittleEndian, argumentsLen)
	_ = binary.Write(buf, binary.LittleEndian, exeOrFile)
	_ = binary.Write(buf, binary.LittleEndian, workingDir)
	_ = binary.Write(buf, binary.LittleEndian, arguments)

	return buf.Bytes()
}

func (c *Client) railStartRemoteApp() error {
	c.railState = RailStateExecuteApp

	clientExecute := NewRailClientExecutePDU(c.remoteApp.App, c.remoteApp.WorkingDir, c.remoteApp.Args)

	return c.mcsLayer.Send(c.userID, c.channelIDMap["rail"], clientExecute.Serialize())
}

// RailPDUExecResult represents a RAIL execution result PDU from the server.
type RailPDUExecResult struct {
	Flags      uint16
	ExecResult uint16
	RawResult  uint32
	ExeOrFile  string
}

// Deserialize decodes a RailPDUExecResult from wire format.
func (pdu *RailPDUExecResult) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &pdu.Flags)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &pdu.ExecResult)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &pdu.RawResult)
	if err != nil {
		return err
	}

	var padding uint16
	err = binary.Read(wire, binary.LittleEndian, &padding)
	if err != nil {
		return err
	}

	var exeOrFileLength uint16
	err = binary.Read(wire, binary.LittleEndian, &exeOrFileLength)
	if err != nil {
		return err
	}

	exeOrFile := make([]byte, exeOrFileLength)
	_, err = wire.Read(exeOrFile)
	if err != nil {
		return err
	}
	pdu.ExeOrFile = string(exeOrFile)

	return nil
}

func (c *Client) railReceiveRemoteAppStatus(*RailPDU) error {
	c.railState = RailStateWaitForData

	// RAIL (RemoteApp) is not supported in HTML5 client.
	// Browsers cannot create native OS windows.

	return nil
}
