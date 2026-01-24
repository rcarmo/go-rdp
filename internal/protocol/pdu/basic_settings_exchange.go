package pdu

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/rcarmo/go-rdp/internal/codec"
)

const (
	rdpVersion5Plus             = 0x00080004
	keyboardTypeIBM101or102Keys = 0x00000004
	projectName                 = "go-rdp"
)

// earlyCapabilityFlags
const (
	ECFSupportErrInfoPDU        uint16 = 0x0001
	ECFWant32BPPSession         uint16 = 0x0002
	ECFSupportStatusInfoPDU     uint16 = 0x0004
	ECFStrongAsymmetricKeys     uint16 = 0x0008
	ECFUnused                   uint16 = 0x0010
	ECFValidConnectionType      uint16 = 0x0020
	ECFSupportMonitorLayoutPDU  uint16 = 0x0040
	ECFSupportNetCharAutodetect uint16 = 0x0080
	ECFSupportDynvcGFXProtocol  uint16 = 0x0100
	ECFSupportDynamicTimeZone   uint16 = 0x0200
	ECFSupportHeartbeatPDU      uint16 = 0x0400
)

// ClientCoreData contains client core settings sent during the Basic Settings Exchange phase.
// See MS-RDPBCGR section 2.2.1.3.2 for the Client Core Data (TS_UD_CS_CORE) structure.
type ClientCoreData struct {
	Version                uint32
	DesktopWidth           uint16
	DesktopHeight          uint16
	ColorDepth             uint16
	SASSequence            uint16
	KeyboardLayout         uint32
	ClientBuild            uint32
	ClientName             [32]byte
	KeyboardType           uint32
	KeyboardSubType        uint32
	KeyboardFunctionKey    uint32
	ImeFileName            [64]byte
	PostBeta2ColorDepth    uint16
	ClientProductId        uint16
	SerialNumber           uint32
	HighColorDepth         uint16
	SupportedColorDepths   uint16
	EarlyCapabilityFlags   uint16
	ClientDigProductId     [64]byte
	ConnectionType         uint8
	Pad1octet              uint8
	ServerSelectedProtocol uint32
	// Optional extended fields (MS-RDPBCGR 2.2.1.3.2)
	DesktopPhysicalWidth  uint32
	DesktopPhysicalHeight uint32
	DesktopOrientation    uint16
	DesktopScaleFactor    uint32
	DeviceScaleFactor     uint32
}

// Color depth constants from MS-RDPBCGR
const (
	// HighColorDepth values
	HighColor4BPP  uint16 = 0x0004
	HighColor8BPP  uint16 = 0x0008
	HighColor15BPP uint16 = 0x000F
	HighColor16BPP uint16 = 0x0010
	HighColor24BPP uint16 = 0x0018

	// SupportedColorDepths flags
	RNS_UD_24BPP_SUPPORT uint16 = 0x0001
	RNS_UD_16BPP_SUPPORT uint16 = 0x0002
	RNS_UD_15BPP_SUPPORT uint16 = 0x0004
	RNS_UD_32BPP_SUPPORT uint16 = 0x0008
)

func newClientCoreData(selectedProtocol uint32, desktopWidth, desktopHeight uint16, colorDepth int) *ClientCoreData {
	// Set color depth values based on requested depth
	var highColorDepth uint16
	var supportedColorDepths uint16
	earlyCapabilityFlags := ECFSupportErrInfoPDU

	switch colorDepth {
	case 32:
		highColorDepth = HighColor24BPP // 32-bit uses 24-bit high color + ECFWant32BPPSession flag
		supportedColorDepths = RNS_UD_32BPP_SUPPORT | RNS_UD_24BPP_SUPPORT | RNS_UD_16BPP_SUPPORT
		earlyCapabilityFlags |= ECFWant32BPPSession
	case 24:
		highColorDepth = HighColor24BPP
		supportedColorDepths = RNS_UD_24BPP_SUPPORT | RNS_UD_16BPP_SUPPORT
	case 15:
		highColorDepth = HighColor15BPP
		supportedColorDepths = RNS_UD_15BPP_SUPPORT | RNS_UD_16BPP_SUPPORT
	case 8:
		highColorDepth = HighColor8BPP
		supportedColorDepths = RNS_UD_16BPP_SUPPORT // Server may upgrade
	default: // 16-bit
		highColorDepth = HighColor16BPP
		supportedColorDepths = RNS_UD_16BPP_SUPPORT
	}

	data := ClientCoreData{
		Version:                rdpVersion5Plus,
		DesktopWidth:           desktopWidth,
		DesktopHeight:          desktopHeight,
		ColorDepth:             0xCA01,     // RNS_UD_COLOR_8BPP (ignored when HighColorDepth is set)
		SASSequence:            0xAA03,     // RNS_UD_SAS_DEL
		KeyboardLayout:         0x00000409, // US
		ClientBuild:            0xece,
		ClientName:             [32]byte{},
		KeyboardType:           keyboardTypeIBM101or102Keys,
		KeyboardSubType:        0x00000000,
		KeyboardFunctionKey:    12,
		ImeFileName:            [64]byte{},
		PostBeta2ColorDepth:    0xCA03, // RNS_UD_COLOR_16BPP_565 (ignored when HighColorDepth is set)
		ClientProductId:        0x0001,
		SerialNumber:           0x00000000,
		HighColorDepth:         highColorDepth,
		SupportedColorDepths:   supportedColorDepths,
		EarlyCapabilityFlags:   earlyCapabilityFlags,
		ClientDigProductId:     [64]byte{},
		ConnectionType:         0x00,
		Pad1octet:              0x00,
		ServerSelectedProtocol: selectedProtocol,
		// Physical dimensions (in millimeters) - assume standard 96 DPI monitor
		// desktopWidth pixels * 25.4mm/inch / 96 DPI
		DesktopPhysicalWidth:  uint32(float64(desktopWidth) * 25.4 / 96.0),
		DesktopPhysicalHeight: uint32(float64(desktopHeight) * 25.4 / 96.0),
		DesktopOrientation:    0,   // ORIENTATION_LANDSCAPE
		DesktopScaleFactor:    100, // 100% scaling
		DeviceScaleFactor:     100, // 100% scaling
	}

	copy(data.ClientName[:], codec.Encode(projectName))

	return &data
}

const (
	// EncryptionFlag40Bit ENCRYPTION_FLAG_40BIT
	EncryptionFlag40Bit uint32 = 0x00000001

	// EncryptionFlag128Bit ENCRYPTION_FLAG_128BIT
	EncryptionFlag128Bit uint32 = 0x00000002

	// EncryptionFlag56Bit ENCRYPTION_FLAG_56BIT
	EncryptionFlag56Bit uint32 = 0x00000008

	// FIPSEncryptionFlag FIPS_ENCRYPTION_FLAG
	FIPSEncryptionFlag uint32 = 0x00000010
)

// ClientSecurityData contains client security settings for encryption negotiation.
// See MS-RDPBCGR section 2.2.1.3.3 for the Client Security Data (TS_UD_CS_SEC) structure.
type ClientSecurityData struct {
	EncryptionMethods    uint32
	ExtEncryptionMethods uint32
}

func newClientSecurityData() *ClientSecurityData {
	data := ClientSecurityData{
		EncryptionMethods:    0,
		ExtEncryptionMethods: 0,
	}

	return &data
}

// ChannelDefinitionStructure defines a static virtual channel requested by the client.
// See MS-RDPBCGR section 2.2.1.3.4.1 for the Channel Definition Structure (CHANNEL_DEF).
type ChannelDefinitionStructure struct {
	Name    [8]byte // seven ANSI chars with null-termination char in the end
	Options uint32
}

// ClientNetworkData contains the list of static virtual channels requested by the client.
// See MS-RDPBCGR section 2.2.1.3.4 for the Client Network Data (TS_UD_CS_NET) structure.
type ClientNetworkData struct {
	ChannelCount    uint32
	ChannelDefArray []ChannelDefinitionStructure
}

// ClientClusterData contains client cluster settings for session redirection.
// See MS-RDPBCGR section 2.2.1.3.5 for the Client Cluster Data (TS_UD_CS_CLUSTER) structure.
type ClientClusterData struct {
	Flags               uint32
	RedirectedSessionID uint32
}

// ClientUserDataSet aggregates all client GCC user data blocks sent to the server.
type ClientUserDataSet struct {
	ClientCoreData     *ClientCoreData
	ClientSecurityData *ClientSecurityData
	ClientNetworkData  *ClientNetworkData
	ClientClusterData  *ClientClusterData
}

// NewClientUserDataSet creates a new ClientUserDataSet with the specified connection parameters.
func NewClientUserDataSet(selectedProtocol uint32,
	desktopWidth, desktopHeight uint16,
	colorDepth int,
	channelNames []string) *ClientUserDataSet {
	return &ClientUserDataSet{
		ClientCoreData:     newClientCoreData(selectedProtocol, desktopWidth, desktopHeight, colorDepth),
		ClientSecurityData: newClientSecurityData(),
		ClientNetworkData:  newClientNetworkData(channelNames),
	}
}

// Serialize encodes the ClientCoreData into its wire format with a CS_CORE header.
func (data ClientCoreData) Serialize() []byte {
	// Updated dataLen to include optional extended fields:
	// Base: 216 bytes (header + fields up to ServerSelectedProtocol)
	// + DesktopPhysicalWidth (4) + DesktopPhysicalHeight (4) + DesktopOrientation (2)
	// + DesktopScaleFactor (4) + DeviceScaleFactor (4) = 234 bytes
	const dataLen uint16 = 234

	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, uint16(0xC001)) // header type CS_CORE
	_ = binary.Write(buf, binary.LittleEndian, dataLen)        // packet size

	_ = binary.Write(buf, binary.LittleEndian, data.Version)
	_ = binary.Write(buf, binary.LittleEndian, data.DesktopWidth)
	_ = binary.Write(buf, binary.LittleEndian, data.DesktopHeight)
	_ = binary.Write(buf, binary.LittleEndian, data.ColorDepth)
	_ = binary.Write(buf, binary.LittleEndian, data.SASSequence)
	_ = binary.Write(buf, binary.LittleEndian, data.KeyboardLayout)
	_ = binary.Write(buf, binary.LittleEndian, data.ClientBuild)
	_ = binary.Write(buf, binary.LittleEndian, data.ClientName)
	_ = binary.Write(buf, binary.LittleEndian, data.KeyboardType)
	_ = binary.Write(buf, binary.LittleEndian, data.KeyboardSubType)
	_ = binary.Write(buf, binary.LittleEndian, data.KeyboardFunctionKey)
	_ = binary.Write(buf, binary.LittleEndian, data.ImeFileName)
	_ = binary.Write(buf, binary.LittleEndian, data.PostBeta2ColorDepth)
	_ = binary.Write(buf, binary.LittleEndian, data.ClientProductId)
	_ = binary.Write(buf, binary.LittleEndian, data.SerialNumber)
	_ = binary.Write(buf, binary.LittleEndian, data.HighColorDepth)
	_ = binary.Write(buf, binary.LittleEndian, data.SupportedColorDepths)
	_ = binary.Write(buf, binary.LittleEndian, data.EarlyCapabilityFlags)
	_ = binary.Write(buf, binary.LittleEndian, data.ClientDigProductId)
	_ = binary.Write(buf, binary.LittleEndian, data.ConnectionType)
	_ = binary.Write(buf, binary.LittleEndian, data.Pad1octet)
	_ = binary.Write(buf, binary.LittleEndian, data.ServerSelectedProtocol)
	// Optional extended fields (MS-RDPBCGR 2.2.1.3.2)
	_ = binary.Write(buf, binary.LittleEndian, data.DesktopPhysicalWidth)
	_ = binary.Write(buf, binary.LittleEndian, data.DesktopPhysicalHeight)
	_ = binary.Write(buf, binary.LittleEndian, data.DesktopOrientation)
	_ = binary.Write(buf, binary.LittleEndian, data.DesktopScaleFactor)
	_ = binary.Write(buf, binary.LittleEndian, data.DeviceScaleFactor)

	return buf.Bytes()
}

const (
	// EncryptionMethodFlag40Bit 40BIT_ENCRYPTION_FLAG
	EncryptionMethodFlag40Bit uint32 = 0x00000001

	// EncryptionMethodFlag56Bit 56BIT_ENCRYPTION_FLAG
	EncryptionMethodFlag56Bit uint32 = 0x00000008

	// EncryptionMethodFlag128Bit 128BIT_ENCRYPTION_FLAG
	EncryptionMethodFlag128Bit uint32 = 0x00000002

	// EncryptionMethodFlagFIPS FIPS_ENCRYPTION_FLAG
	EncryptionMethodFlagFIPS uint32 = 0x00000010
)

// Serialize encodes the ClientSecurityData into its wire format with a CS_SECURITY header.
func (data ClientSecurityData) Serialize() []byte {
	const dataLen uint16 = 12

	buf := bytes.NewBuffer(make([]byte, 0, 6))

	_ = binary.Write(buf, binary.LittleEndian, uint16(0xC002)) // header type CS_SECURITY
	_ = binary.Write(buf, binary.LittleEndian, dataLen)        // packet size

	_ = binary.Write(buf, binary.LittleEndian, data.EncryptionMethods)
	_ = binary.Write(buf, binary.LittleEndian, data.ExtEncryptionMethods)

	return buf.Bytes()
}

// Serialize encodes the ChannelDefinitionStructure into its wire format.
func (s ChannelDefinitionStructure) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.Name)
	_ = binary.Write(buf, binary.LittleEndian, s.Options)

	return buf.Bytes()
}

func newClientNetworkData(channelNames []string) *ClientNetworkData {
	data := ClientNetworkData{
		ChannelCount: uint32(len(channelNames)),
	}

	if data.ChannelCount == 0 {
		return &data
	}

	for _, channelName := range channelNames {
		channelDefinition := ChannelDefinitionStructure{}
		copy(channelDefinition.Name[:], channelName)

		data.ChannelDefArray = append(data.ChannelDefArray, channelDefinition)
	}

	return &data
}

// Serialize encodes the ClientNetworkData into its wire format with a CS_NET header.
func (data ClientNetworkData) Serialize() []byte {
	const headerLen = 8

	chBuf := new(bytes.Buffer)

	for _, channelDef := range data.ChannelDefArray {
		chBuf.Write(channelDef.Serialize())
	}

	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, uint16(0xC003))                // header type CS_NET
	_ = binary.Write(buf, binary.LittleEndian, uint16(headerLen+chBuf.Len())) // packet size

	_ = binary.Write(buf, binary.LittleEndian, data.ChannelCount)

	buf.Write(chBuf.Bytes())

	return buf.Bytes()
}

// Serialize encodes the ClientClusterData into its wire format with a CS_CLUSTER header.
func (d ClientClusterData) Serialize() []byte {
	const dataLen uint16 = 12

	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, uint16(0xC004)) // header type CS_CLUSTER
	_ = binary.Write(buf, binary.LittleEndian, dataLen)        // packet size

	_ = binary.Write(buf, binary.LittleEndian, d.Flags)
	_ = binary.Write(buf, binary.LittleEndian, d.RedirectedSessionID)

	return buf.Bytes()
}

// Serialize encodes all client user data blocks into their combined wire format.
func (ud ClientUserDataSet) Serialize() []byte {
	buf := new(bytes.Buffer)

	buf.Write(ud.ClientCoreData.Serialize())

	if ud.ClientClusterData != nil {
		buf.Write(ud.ClientClusterData.Serialize())
	}

	buf.Write(ud.ClientSecurityData.Serialize())
	buf.Write(ud.ClientNetworkData.Serialize())

	return buf.Bytes()
}

// ServerCoreData contains server core settings received during the Basic Settings Exchange phase.
// See MS-RDPBCGR section 2.2.1.4.2 for the Server Core Data (TS_UD_SC_CORE) structure.
type ServerCoreData struct {
	Version                  uint32
	ClientRequestedProtocols uint32
	EarlyCapabilityFlags     uint32

	DataLen uint16
}

// Deserialize decodes the ServerCoreData from its wire format.
func (d *ServerCoreData) Deserialize(wire io.Reader) error {
	var err error

	if err = binary.Read(wire, binary.LittleEndian, &d.Version); err != nil {
		return err
	}

	if d.DataLen == 4 {
		return nil
	}

	if err = binary.Read(wire, binary.LittleEndian, &d.ClientRequestedProtocols); err != nil {
		return err
	}

	if d.DataLen == 8 {
		return nil
	}

	if err = binary.Read(wire, binary.LittleEndian, &d.EarlyCapabilityFlags); err != nil {
		return err
	}

	return nil
}

// RSAPublicKey represents an RSA public key used in server proprietary certificates.
// See MS-RDPBCGR section 2.2.1.4.3.1.1.1 for the RSA Public Key (RSA_PUBLIC_KEY) structure.
type RSAPublicKey struct {
	Magic   uint32
	KeyLen  uint32
	BitLen  uint32
	DataLen uint32
	PubExp  uint32
	Modulus []byte
}

// Deserialize decodes the RSAPublicKey from its wire format.
func (k *RSAPublicKey) Deserialize(wire io.Reader) error {
	var err error

	if err = binary.Read(wire, binary.LittleEndian, &k.Magic); err != nil {
		return err
	}

	if err = binary.Read(wire, binary.LittleEndian, &k.KeyLen); err != nil {
		return err
	}

	if err = binary.Read(wire, binary.LittleEndian, &k.BitLen); err != nil {
		return err
	}

	if err = binary.Read(wire, binary.LittleEndian, &k.DataLen); err != nil {
		return err
	}

	if err = binary.Read(wire, binary.LittleEndian, &k.PubExp); err != nil {
		return err
	}

	k.Modulus = make([]byte, k.KeyLen)

	if _, err = wire.Read(k.Modulus); err != nil {
		return err
	}

	return nil
}

// ServerProprietaryCertificate contains the server's proprietary certificate for encryption.
// See MS-RDPBCGR section 2.2.1.4.3.1.1 for the Server Proprietary Certificate structure.
type ServerProprietaryCertificate struct {
	DwSigAlgId        uint32
	DwKeyAlgId        uint32
	PublicKeyBlobType uint16
	PublicKeyBlobLen  uint16
	PublicKeyBlob     RSAPublicKey
	SignatureBlobType uint16
	SignatureBlobLen  uint16
	SignatureBlob     []byte
}

// Deserialize decodes the ServerProprietaryCertificate from its wire format.
func (c *ServerProprietaryCertificate) Deserialize(wire io.Reader) error {
	var err error

	if err = binary.Read(wire, binary.LittleEndian, &c.DwSigAlgId); err != nil {
		return err
	}

	if err = binary.Read(wire, binary.LittleEndian, &c.DwKeyAlgId); err != nil {
		return err
	}

	if err = binary.Read(wire, binary.LittleEndian, &c.PublicKeyBlobType); err != nil {
		return err
	}

	if err = binary.Read(wire, binary.LittleEndian, &c.PublicKeyBlobLen); err != nil {
		return err
	}

	if err = c.PublicKeyBlob.Deserialize(wire); err != nil {
		return err
	}

	if err = binary.Read(wire, binary.LittleEndian, &c.SignatureBlobType); err != nil {
		return err
	}

	if err = binary.Read(wire, binary.LittleEndian, &c.SignatureBlobLen); err != nil {
		return err
	}

	c.SignatureBlob = make([]byte, c.SignatureBlobLen)

	if _, err = wire.Read(c.SignatureBlob); err != nil {
		return err
	}

	return nil
}

// ServerCertificate contains the server's certificate (proprietary or X.509).
// See MS-RDPBCGR section 2.2.1.4.3.1 for the Server Certificate structure.
type ServerCertificate struct {
	DwVersion       uint32
	ProprietaryCert *ServerProprietaryCertificate
	X509Cert        []byte

	ServerCertLen uint32
}

// Deserialize decodes the ServerCertificate from its wire format.
func (c *ServerCertificate) Deserialize(wire io.Reader) error {
	var err error

	if err = binary.Read(wire, binary.LittleEndian, &c.DwVersion); err != nil {
		return err
	}

	if c.DwVersion&0x00000001 == 0x00000001 {
		c.ProprietaryCert = &ServerProprietaryCertificate{}

		return c.ProprietaryCert.Deserialize(wire)
	}

	// Validate certificate length to prevent integer underflow
	if c.ServerCertLen < 4 {
		return fmt.Errorf("invalid certificate length: %d (minimum 4)", c.ServerCertLen)
	}
	c.X509Cert = make([]byte, c.ServerCertLen-4)

	if _, err = wire.Read(c.X509Cert); err != nil {
		return err
	}

	return nil
}

// ServerSecurityData contains server security settings including encryption parameters.
// See MS-RDPBCGR section 2.2.1.4.3 for the Server Security Data (TS_UD_SC_SEC1) structure.
type ServerSecurityData struct {
	EncryptionMethod  uint32
	EncryptionLevel   uint32
	ServerRandomLen   uint32
	ServerCertLen     uint32
	ServerRandom      []byte
	ServerCertificate *ServerCertificate
}

// Deserialize decodes the ServerSecurityData from its wire format.
func (d *ServerSecurityData) Deserialize(wire io.Reader) error {
	var err error

	if err = binary.Read(wire, binary.LittleEndian, &d.EncryptionMethod); err != nil {
		return err
	}

	if err = binary.Read(wire, binary.LittleEndian, &d.EncryptionLevel); err != nil {
		return err
	}

	if d.EncryptionMethod == 0 && d.EncryptionLevel == 0 { // ENCRYPTION_METHOD_NONE and ENCRYPTION_LEVEL_NONE
		return nil
	}

	if err = binary.Read(wire, binary.LittleEndian, &d.ServerRandomLen); err != nil {
		return err
	}

	if err = binary.Read(wire, binary.LittleEndian, &d.ServerCertLen); err != nil {
		return err
	}

	d.ServerRandom = make([]byte, d.ServerRandomLen)

	if _, err = wire.Read(d.ServerRandom); err != nil {
		return err
	}

	if d.ServerCertLen > 0 {
		d.ServerCertificate = &ServerCertificate{
			ServerCertLen: d.ServerCertLen,
		}

		return d.ServerCertificate.Deserialize(wire)
	}

	return nil
}

// ServerNetworkData contains the MCS channel ID and virtual channel IDs assigned by the server.
// See MS-RDPBCGR section 2.2.1.4.4 for the Server Network Data (TS_UD_SC_NET) structure.
type ServerNetworkData struct {
	MCSChannelId   uint16
	ChannelCount   uint16
	ChannelIdArray []uint16
}

// Deserialize decodes the ServerNetworkData from its wire format.
func (d *ServerNetworkData) Deserialize(wire io.Reader) error {
	var err error

	if err = binary.Read(wire, binary.LittleEndian, &d.MCSChannelId); err != nil {
		return err
	}

	if err = binary.Read(wire, binary.LittleEndian, &d.ChannelCount); err != nil {
		return err
	}

	if d.ChannelCount == 0 {
		return nil
	}

	d.ChannelIdArray = make([]uint16, d.ChannelCount)

	if err = binary.Read(wire, binary.LittleEndian, &d.ChannelIdArray); err != nil {
		return err
	}

	if d.ChannelCount%2 == 0 {
		return nil
	}

	padding := make([]byte, 2)

	if _, err = wire.Read(padding); err != nil {
		return err
	}

	return nil
}

// ServerMessageChannelData contains the MCS channel ID for the message channel.
// See MS-RDPBCGR section 2.2.1.4.5 for the Server Message Channel Data (TS_UD_SC_MCS_MSGCHANNEL).
type ServerMessageChannelData struct {
	MCSChannelID uint16
}

// ServerMultitransportChannelData contains multitransport channel flags.
// See MS-RDPBCGR section 2.2.1.4.6 for the Server Multitransport Channel Data (TS_UD_SC_MULTITRANSPORT).
type ServerMultitransportChannelData struct {
	Flags uint32
}

// ServerUserData aggregates all server GCC user data blocks received from the server.
type ServerUserData struct {
	ServerCoreData                  *ServerCoreData
	ServerNetworkData               *ServerNetworkData
	ServerSecurityData              *ServerSecurityData
	ServerMessageChannelData        *ServerMessageChannelData
	ServerMultitransportChannelData *ServerMultitransportChannelData
}

// Deserialize decodes all server user data blocks from their combined wire format.
func (ud *ServerUserData) Deserialize(wire io.Reader) error {
	var (
		dataType uint16
		dataLen  uint16
		err      error
	)

	for {
		err = binary.Read(wire, binary.LittleEndian, &dataType)
		switch err {
		case nil: // pass
		case io.EOF:
			return nil
		default:
			return err
		}

		err = binary.Read(wire, binary.LittleEndian, &dataLen)
		if err != nil {
			return err
		}

		dataLen -= 4 // exclude User Data Header

		switch dataType {
		case 0x0C01:
			ud.ServerCoreData = &ServerCoreData{DataLen: dataLen}

			if err = ud.ServerCoreData.Deserialize(wire); err != nil {
				return err
			}
		case 0x0C02:
			ud.ServerSecurityData = &ServerSecurityData{}

			if err = ud.ServerSecurityData.Deserialize(wire); err != nil {
				return err
			}
		case 0x0C03:
			ud.ServerNetworkData = &ServerNetworkData{}

			if err = ud.ServerNetworkData.Deserialize(wire); err != nil {
				return err
			}
		case 0x0C04:
			ud.ServerMessageChannelData = &ServerMessageChannelData{}

			if err = binary.Read(wire, binary.LittleEndian, &ud.ServerMessageChannelData.MCSChannelID); err != nil {
				return err
			}
		case 0x0C08:
			ud.ServerMultitransportChannelData = &ServerMultitransportChannelData{}

			if err = binary.Read(wire, binary.LittleEndian, &ud.ServerMultitransportChannelData.Flags); err != nil {
				return err
			}
		default:
			return errors.New("unknown header type")
		}
	}
}
