package pdu

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorInfoPDUData_Deserialize(t *testing.T) {
	tests := []struct {
		name      string
		errorCode uint32
		data      []byte
	}{
		{
			name:      "None",
			errorCode: 0x00000000,
			data:      []byte{0x00, 0x00, 0x00, 0x00},
		},
		{
			name:      "RPCInitiatedDisconnect",
			errorCode: 0x00000001,
			data:      []byte{0x01, 0x00, 0x00, 0x00},
		},
		{
			name:      "RPCInitiatedLogoff",
			errorCode: 0x00000002,
			data:      []byte{0x02, 0x00, 0x00, 0x00},
		},
		{
			name:      "IdleTimeout",
			errorCode: 0x00000003,
			data:      []byte{0x03, 0x00, 0x00, 0x00},
		},
		{
			name:      "LogonTimeout",
			errorCode: 0x00000004,
			data:      []byte{0x04, 0x00, 0x00, 0x00},
		},
		{
			name:      "DisconnectedByOtherConnection",
			errorCode: 0x00000005,
			data:      []byte{0x05, 0x00, 0x00, 0x00},
		},
		{
			name:      "OutOfMemory",
			errorCode: 0x00000006,
			data:      []byte{0x06, 0x00, 0x00, 0x00},
		},
		{
			name:      "ServerDeniedConnection",
			errorCode: 0x00000007,
			data:      []byte{0x07, 0x00, 0x00, 0x00},
		},
		{
			name:      "LicenseInternal",
			errorCode: 0x00000100,
			data:      []byte{0x00, 0x01, 0x00, 0x00},
		},
		{
			name:      "UnknownPDUType2",
			errorCode: 0x000010C9,
			data:      []byte{0xC9, 0x10, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pdu ErrorInfoPDUData
			err := pdu.Deserialize(bytes.NewReader(tt.data))
			require.NoError(t, err)
			require.Equal(t, tt.errorCode, pdu.ErrorInfo)
		})
	}
}

func TestErrorInfoPDUData_DeserializeError(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"Empty", []byte{}},
		{"TooShort", []byte{0x01, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pdu ErrorInfoPDUData
			err := pdu.Deserialize(bytes.NewReader(tt.data))
			require.Error(t, err)
		})
	}
}

func TestErrorInfoPDUData_String(t *testing.T) {
	tests := []struct {
		name      string
		errorCode uint32
		expected  string
	}{
		{"None", 0x00000000, "ERRINFO_NONE"},
		{"RPCInitiatedDisconnect", 0x00000001, "ERRINFO_RPC_INITIATED_DISCONNECT"},
		{"RPCInitiatedLogoff", 0x00000002, "ERRINFO_RPC_INITIATED_LOGOFF"},
		{"IdleTimeout", 0x00000003, "ERRINFO_IDLE_TIMEOUT"},
		{"LogonTimeout", 0x00000004, "ERRINFO_LOGON_TIMEOUT"},
		{"DisconnectedByOther", 0x00000005, "ERRINFO_DISCONNECTED_BY_OTHERCONNECTION"},
		{"OutOfMemory", 0x00000006, "ERRINFO_OUT_OF_MEMORY"},
		{"ServerDenied", 0x00000007, "ERRINFO_SERVER_DENIED_CONNECTION"},
		{"InsufficientPrivileges", 0x00000009, "ERRINFO_SERVER_INSUFFICIENT_PRIVILEGES"},
		{"FreshCredentials", 0x0000000A, "ERRINFO_SERVER_FRESH_CREDENTIALS_REQUIRED"},
		{"RPCDisconnectByUser", 0x0000000B, "ERRINFO_RPC_INITIATED_DISCONNECT_BYUSER"},
		{"LogoffByUser", 0x0000000C, "ERRINFO_LOGOFF_BY_USER"},
		{"CloseStackDriverNotReady", 0x0000000F, "ERRINFO_CLOSE_STACK_ON_DRIVER_NOT_READY"},
		{"ServerDWMCrash", 0x00000010, "ERRINFO_SERVER_DWM_CRASH"},
		{"CloseStackDriverFailure", 0x00000011, "ERRINFO_CLOSE_STACK_ON_DRIVER_FAILURE"},
		{"CloseStackDriverIfaceFailure", 0x00000012, "ERRINFO_CLOSE_STACK_ON_DRIVER_IFACE_FAILURE"},
		{"WinlogonCrash", 0x00000017, "ERRINFO_SERVER_WINLOGON_CRASH"},
		{"CsrssCrash", 0x00000018, "ERRINFO_SERVER_CSRSS_CRASH"},
		{"ServerShutdown", 0x00000019, "ERRINFO_SERVER_SHUTDOWN"},
		{"ServerReboot", 0x0000001A, "ERRINFO_SERVER_REBOOT"},
		{"LicenseInternal", 0x00000100, "ERRINFO_LICENSE_INTERNAL"},
		{"LicenseNoServer", 0x00000101, "ERRINFO_LICENSE_NO_LICENSE_SERVER"},
		{"LicenseNoLicense", 0x00000102, "ERRINFO_LICENSE_NO_LICENSE"},
		{"LicenseBadClientMsg", 0x00000103, "ERRINFO_LICENSE_BAD_CLIENT_MSG"},
		{"LicenseHWIDMismatch", 0x00000104, "ERRINFO_LICENSE_HWID_DOESNT_MATCH_LICENSE"},
		{"LicenseBadClientLicense", 0x00000105, "ERRINFO_LICENSE_BAD_CLIENT_LICENSE"},
		{"LicenseCantFinish", 0x00000106, "ERRINFO_LICENSE_CANT_FINISH_PROTOCOL"},
		{"LicenseClientEnded", 0x00000107, "ERRINFO_LICENSE_CLIENT_ENDED_PROTOCOL"},
		{"LicenseBadEncryption", 0x00000108, "ERRINFO_LICENSE_BAD_CLIENT_ENCRYPTION"},
		{"LicenseCantUpgrade", 0x00000109, "ERRINFO_LICENSE_CANT_UPGRADE_LICENSE"},
		{"LicenseNoRemote", 0x0000010A, "ERRINFO_LICENSE_NO_REMOTE_CONNECTIONS"},
		{"CBDestNotFound", 0x00000400, "ERRINFO_CB_DESTINATION_NOT_FOUND"},
		{"CBLoadingDest", 0x00000402, "ERRINFO_CB_LOADING_DESTINATION"},
		{"CBRedirecting", 0x00000404, "ERRINFO_CB_REDIRECTING_TO_DESTINATION"},
		{"CBSessionOnlineVMWake", 0x00000405, "ERRINFO_CB_SESSION_ONLINE_VM_WAKE"},
		{"CBSessionOnlineVMBoot", 0x00000406, "ERRINFO_CB_SESSION_ONLINE_VM_BOOT"},
		{"CBSessionOnlineVMNoDNS", 0x00000407, "ERRINFO_CB_SESSION_ONLINE_VM_NO_DNS"},
		{"CBDestPoolNotFree", 0x00000408, "ERRINFO_CB_DESTINATION_POOL_NOT_FREE"},
		{"CBConnectionCancelled", 0x00000409, "ERRINFO_CB_CONNECTION_CANCELLED"},
		{"CBInvalidSettings", 0x00000410, "ERRINFO_CB_CONNECTION_ERROR_INVALID_SETTINGS"},
		{"CBVMBootTimeout", 0x00000411, "ERRINFO_CB_SESSION_ONLINE_VM_BOOT_TIMEOUT"},
		{"CBSessmonFailed", 0x00000412, "ERRINFO_CB_SESSION_ONLINE_VM_SESSMON_FAILED"},
		{"UnknownPDUType2", 0x000010C9, "ERRINFO_UNKNOWNPDUTYPE2"},
		{"UnknownPDUType", 0x000010CA, "ERRINFO_UNKNOWNPDUTYPE"},
		{"DataPDUSequence", 0x000010CB, "ERRINFO_DATAPDUSEQUENCE"},
		{"ControlPDUSequence", 0x000010CD, "ERRINFO_CONTROLPDUSEQUENCE"},
		{"InvalidControlAction", 0x000010CE, "ERRINFO_INVALIDCONTROLPDUACTION"},
		{"InvalidInputPDUType", 0x000010CF, "ERRINFO_INVALIDINPUTPDUTYPE"},
		{"InvalidInputPDUMouse", 0x000010D0, "ERRINFO_INVALIDINPUTPDUMOUSE"},
		{"InvalidRefreshRect", 0x000010D1, "ERRINFO_INVALIDREFRESHRECTPDU"},
		{"CreateUserDataFailed", 0x000010D2, "ERRINFO_CREATEUSERDATAFAILED"},
		{"ConnectFailed", 0x000010D3, "ERRINFO_CONNECTFAILED"},
		{"ConfirmActiveWrongShareID", 0x000010D4, "ERRINFO_CONFIRMACTIVEWRONGSHAREID"},
		{"ConfirmActiveWrongOriginator", 0x000010D5, "ERRINFO_CONFIRMACTIVEWRONGORIGINATOR"},
		{"PersistentKeyBadLength", 0x000010DA, "ERRINFO_PERSISTENTKEYPDUBADLENGTH"},
		{"PersistentKeyIllegalFirst", 0x000010DB, "ERRINFO_PERSISTENTKEYPDUILLEGALFIRST"},
		{"PersistentKeyTooManyTotal", 0x000010DC, "ERRINFO_PERSISTENTKEYPDUTOOMANYTOTALKEYS"},
		{"PersistentKeyTooManyCache", 0x000010DD, "ERRINFO_PERSISTENTKEYPDUTOOMANYCACHEKEYS"},
		{"InputPDUBadLength", 0x000010DE, "ERRINFO_INPUTPDUBADLENGTH"},
		{"BitmapCacheErrorBadLength", 0x000010DF, "ERRINFO_BITMAPCACHEERRORPDUBADLENGTH"},
		{"SecurityDataTooShort", 0x000010E0, "ERRINFO_SECURITYDATATOOSHORT"},
		{"VChannelDataTooShort", 0x000010E1, "ERRINFO_VCHANNELDATATOOSHORT"},
		{"ShareDataTooShort", 0x000010E2, "ERRINFO_SHAREDATATOOSHORT"},
		{"BadSupressOutput", 0x000010E3, "ERRINFO_BADSUPRESSOUTPUTPDU"},
		{"ConfirmActiveTooShort", 0x000010E5, "ERRINFO_CONFIRMACTIVEPDUTOOSHORT"},
		{"CapabilitySetTooSmall", 0x000010E7, "ERRINFO_CAPABILITYSETTOOSMALL"},
		{"CapabilitySetTooLarge", 0x000010E8, "ERRINFO_CAPABILITYSETTOOLARGE"},
		{"NoCursorCache", 0x000010E9, "ERRINFO_NOCURSORCACHE"},
		{"BadCapabilities", 0x000010EA, "ERRINFO_BADCAPABILITIES"},
		{"VChannelDecompressErr", 0x000010EC, "ERRINFO_VIRTUALCHANNELDECOMPRESSIONERR"},
		{"InvalidVCCompression", 0x000010ED, "ERRINFO_INVALIDVCCOMPRESSIONTYPE"},
		{"InvalidChannelID", 0x000010EF, "ERRINFO_INVALIDCHANNELID"},
		{"VChannelsTooMany", 0x000010F0, "ERRINFO_VCHANNELSTOOMANY"},
		{"RemoteAppsNotEnabled", 0x000010F3, "ERRINFO_REMOTEAPPSNOTENABLED"},
		{"CacheCapNotSet", 0x000010F4, "ERRINFO_CACHECAPNOTSET"},
		{"BitmapCacheErrorBadLength2", 0x000010F5, "ERRINFO_BITMAPCACHEERRORPDUBADLENGTH2"},
		{"OffscreenCacheError", 0x000010F6, "ERRINFO_OFFSCRCACHEERRORPDUBADLENGTH"},
		{"DngCacheError", 0x000010F7, "ERRINFO_DNGCACHEERRORPDUBADLENGTH"},
		{"GdiplusPDUBadLength", 0x000010F8, "ERRINFO_GDIPLUSPDUBADLENGTH"},
		{"SecurityData2", 0x00001111, "ERRINFO_SECURITYDATATOOSHORT2"},
		{"SecurityData3", 0x00001112, "ERRINFO_SECURITYDATATOOSHORT3"},
		{"SecurityData4", 0x00001113, "ERRINFO_SECURITYDATATOOSHORT4"},
		{"SecurityData5", 0x00001114, "ERRINFO_SECURITYDATATOOSHORT5"},
		{"SecurityData6", 0x00001115, "ERRINFO_SECURITYDATATOOSHORT6"},
		{"SecurityData7", 0x00001116, "ERRINFO_SECURITYDATATOOSHORT7"},
		{"SecurityData8", 0x00001117, "ERRINFO_SECURITYDATATOOSHORT8"},
		{"SecurityData9", 0x00001118, "ERRINFO_SECURITYDATATOOSHORT9"},
		{"SecurityData10", 0x00001119, "ERRINFO_SECURITYDATATOOSHORT10"},
		{"SecurityData11", 0x0000111A, "ERRINFO_SECURITYDATATOOSHORT11"},
		{"SecurityData12", 0x0000111B, "ERRINFO_SECURITYDATATOOSHORT12"},
		{"SecurityData13", 0x0000111C, "ERRINFO_SECURITYDATATOOSHORT13"},
		{"SecurityData14", 0x0000111D, "ERRINFO_SECURITYDATATOOSHORT14"},
		{"SecurityData15", 0x0000111E, "ERRINFO_SECURITYDATATOOSHORT15"},
		{"SecurityData16", 0x0000111F, "ERRINFO_SECURITYDATATOOSHORT16"},
		{"SecurityData17", 0x00001120, "ERRINFO_SECURITYDATATOOSHORT17"},
		{"SecurityData18", 0x00001121, "ERRINFO_SECURITYDATATOOSHORT18"},
		{"SecurityData19", 0x00001122, "ERRINFO_SECURITYDATATOOSHORT19"},
		{"SecurityData20", 0x00001123, "ERRINFO_SECURITYDATATOOSHORT20"},
		{"SecurityData21", 0x00001124, "ERRINFO_SECURITYDATATOOSHORT21"},
		{"SecurityData22", 0x00001125, "ERRINFO_SECURITYDATATOOSHORT22"},
		{"SecurityData23", 0x00001126, "ERRINFO_SECURITYDATATOOSHORT23"},
		{"BadMonitorData", 0x00001129, "ERRINFO_BADMONITORDATA"},
		{"VCDecompReassembleFailed", 0x0000112A, "ERRINFO_VCDECOMPRESSEDREASSEMBLEFAILED"},
		{"VCDataTooLong", 0x0000112B, "ERRINFO_VCDATATOOLONG"},
		{"BadFrameAckData", 0x0000112C, "ERRINFO_BAD_FRAME_ACK_DATA"},
		{"GraphicsModeNotSupported", 0x0000112D, "ERRINFO_GRAPHICSMODENOTSUPPORTED"},
		{"GraphicsSubsystemResetFailed", 0x0000112E, "ERRINFO_GRAPHICSSUBSYSTEMRESETFAILED"},
		{"GraphicsSubsystemFailed", 0x0000112F, "ERRINFO_GRAPHICSSUBSYSTEMFAILED"},
		{"TimezoneKeyNameTooShort", 0x00001130, "ERRINFO_TIMEZONEKEYNAMELENGTHTOOSHORT"},
		{"TimezoneKeyNameTooLong", 0x00001131, "ERRINFO_TIMEZONEKEYNAMELENGTHTOOLONG"},
		{"DynamicDSTDisabled", 0x00001132, "ERRINFO_DYNAMICDSTDISABLEDFIELDMISSING"},
		{"VCDecodingError", 0x00001133, "ERRINFO_VCDECODINGERROR"},
		{"VirtualDesktopTooLarge", 0x00001134, "ERRINFO_VIRTUALDESKTOPTOOLARGE"},
		{"MonitorGeometryValidation", 0x00001135, "ERRINFO_MONITORGEOMETRYVALIDATIONFAILED"},
		{"InvalidMonitorCount", 0x00001136, "ERRINFO_INVALIDMONITORCOUNT"},
		{"UpdateSessionKeyFailed", 0x00001191, "ERRINFO_UPDATESESSIONKEYFAILED"},
		{"DecryptFailed", 0x00001192, "ERRINFO_DECRYPTFAILED"},
		{"EncryptFailed", 0x00001193, "ERRINFO_ENCRYPTFAILED"},
		{"EncPkgMismatch", 0x00001194, "ERRINFO_ENCPKGMISMATCH"},
		{"DecryptFailed2", 0x00001195, "ERRINFO_DECRYPTFAILED2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := ErrorInfoPDUData{ErrorInfo: tt.errorCode}
			require.Equal(t, tt.expected, pdu.String())
		})
	}
}

func TestErrorInfoPDUData_StringUnknown(t *testing.T) {
	pdu := ErrorInfoPDUData{ErrorInfo: 0xFFFFFFFF}
	result := pdu.String()
	require.Contains(t, result, "unknown code")
}

func TestErrors(t *testing.T) {
	require.NotNil(t, ErrInvalidCorrelationID)
	require.NotNil(t, ErrDeactiateAll)
	require.Contains(t, ErrInvalidCorrelationID.Error(), "correlationId")
	require.Contains(t, ErrDeactiateAll.Error(), "deactivate")
}

func TestErrorInfoPDUData_DeserializeEOF(t *testing.T) {
	var pdu ErrorInfoPDUData
	err := pdu.Deserialize(bytes.NewReader([]byte{}))
	require.ErrorIs(t, err, io.EOF)
}
