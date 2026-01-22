# Codebase Audit Report

**Date:** January 2026  
**Scope:** Full codebase audit - security, modularity, redundancy, documentation, quality  
**Project:** RDP HTML5 Client (Go backend + JavaScript/WASM frontend)

---

## Executive Summary

Overall the codebase is **well-structured** with good test coverage (average ~80%) and minimal external dependencies. However, there are several security concerns that should be addressed before production deployment, and some code quality improvements that would benefit maintainability.

| Category | Rating | Critical Issues |
|----------|--------|-----------------|
| Security | ⚠️ Medium | 3 issues requiring attention |
| Modularity | ✅ Good | Minor coupling concerns |
| Redundancy | ⚠️ Medium | RLE codec duplication |
| Documentation | ✅ Good | Minor gaps |
| Test Coverage | ✅ Good | 45-100% across packages |
| Code Quality | ⚠️ Medium | Large functions, global state |

---

## 1. Security Audit

### 🔴 HIGH: JSON Injection in Error Messages

**Location:** `internal/handler/connect.go:379`

```go
errMsg := fmt.Sprintf(`{"type":"error","message":"%s"}`, message)
```

**Issue:** Error messages are interpolated directly into JSON without escaping. If an error message contains quotes or special characters, it could break JSON structure or enable injection.

**Recommendation:** Use `json.Marshal()` or escape the message:
```go
type errorResponse struct {
    Type    string `json:"type"`
    Message string `json:"message"`
}
errMsg, _ := json.Marshal(errorResponse{Type: "error", Message: message})
```

---

### 🔴 HIGH: TLS Certificate Validation Bypass

**Location:** `internal/rdp/tls.go:56, 81` and `internal/rdp/nla.go:250`

**Issue:** `InsecureSkipVerify` can be enabled via configuration, allowing MITM attacks on RDP connections.

```go
InsecureSkipVerify: insecureSkipVerify, // RDP servers use self-signed certs
```

**Current Mitigations:**
- Requires explicit configuration (`TLS_SKIP_VERIFY=true`)
- Comment documents the reason (RDP self-signed certs)

**Recommendations:**
1. Log a warning when `InsecureSkipVerify` is enabled
2. Consider certificate pinning for known RDP servers
3. Document security implications in deployment guide

---

### 🟡 MEDIUM: Weak Cryptography (Legacy Protocol)

**Location:** `internal/auth/ntlm.go`

**Issue:** Uses MD4 and MD5 for NTLM authentication. These are cryptographically weak.

**Mitigation:** This is required by the NTLM protocol specification (MS-NLMP). Cannot be changed without breaking compatibility.

**Recommendation:** Document that NTLM authentication relies on legacy crypto and recommend NLA/CredSSP where possible (which adds TLS layer).

---

### 🟡 MEDIUM: Incomplete Checksum Verification

**Location:** `internal/auth/ntlm.go:509`

```go
// TODO: Verify checksum using verifyKey and sequence number
```

**Issue:** NTLM message integrity verification is not fully implemented.

**Recommendation:** Complete the implementation or document the security implications.

---

### 🟢 LOW: Information Disclosure in Error Messages

**Location:** `internal/handler/connect.go:113-118`

TLS error messages may reveal whether certificate validation is enabled or hostname mismatches exist.

**Recommendation:** Use generic error messages in production mode.

---

### ✅ Security Positives

- No hardcoded credentials in source code
- Input validation on all user parameters (width, height, colorDepth, hostname, username)
- Password length limits enforced (max 255 chars)
- CSRF token generation implemented in JavaScript client
- Security headers properly set (CSP, X-Frame-Options, HSTS, etc.)
- Rate limiting middleware available (placeholder implementation)
- CORS properly configured with origin validation

---

## 2. Modularity Audit

### Package Structure

The codebase follows a clean layered architecture:

```
cmd/server          → Entry point
internal/handler    → HTTP/WebSocket bridge
internal/rdp        → RDP client orchestration
internal/protocol/* → Protocol layers (tpkt, x224, mcs, pdu, etc.)
internal/codec      → Bitmap decompression
internal/auth       → NTLM/CredSSP authentication
internal/config     → Configuration management
internal/logging    → Logging utilities
web/                → Frontend assets
```

### ✅ Strengths

- **Minimal external dependencies:** Only `testify` and `golang.org/x/net`
- **No circular dependencies detected**
- **Clean protocol layer separation:** tpkt → x224 → mcs → pdu
- **Interface abstractions:** `rdpConn` interface for testability

### ⚠️ Areas for Improvement

1. **`internal/rdp` package is large** (44+ files)
   - Handles: connection, TLS, NLA, audio, RAIL, virtual channels
   - Consider splitting into subpackages: `rdp/auth`, `rdp/channels`, `rdp/session`

2. **`protocol/pdu` is monolithic** (38 files)
   - Contains all PDU types mixed together
   - Consider grouping: `pdu/capabilities`, `pdu/connection`, `pdu/licensing`

3. **Handler depends on internal RDP types**
   - `handler.go` imports `rdp.Update`, `rdp.ServerCapabilityInfo`
   - Consider defining stable public API in rdp package

---

## 3. Redundancy Audit

### 🔴 HIGH: RLE Codec Duplication

**Location:** `internal/codec/rle*.go`

Five nearly-identical files with pixel-width variations:
- `rle8.go`, `rle15.go`, `rle16.go`, `rle24.go`, `rle32.go`

Each contains duplicate functions:
- `ReadPixel*` / `WritePixel*`
- `WriteFgBgImage*`
- `DecompressRLE*`

**Lines of duplicate code:** ~1,500 lines

**Recommendation:** Refactor using generics or shared helper functions:
```go
type PixelCodec[T uint8 | uint16 | uint32] interface {
    ReadPixel(data []byte, offset int) T
    WritePixel(data []byte, offset int, pixel T)
}
```

---

### 🟡 MEDIUM: Test Helper Duplication

**Location:** Various `*_test.go` files

Similar mock structures defined in multiple test files:
- `mockConn`, `mockReader`, `mockWriter` variants

**Recommendation:** Create `internal/testutil` package with shared test helpers.

---

### ✅ No Significant Redundancy

- Protocol packages are well-separated
- Configuration loading is centralized
- Logging is unified through single package

---

## 4. Documentation Audit

### README Coverage: ✅ Excellent

20 packages have README.md files with:
- Architecture diagrams
- Usage examples
- API documentation

### ⚠️ Missing Godoc Comments

**`cmd/server/main.go`** - Multiple exported functions lack documentation:
- `createServer()`
- `applySecurityMiddleware()`
- `securityHeadersMiddleware()`
- `corsMiddleware()`
- `isOriginAllowed()`

**`internal/protocol/pdu/*.go`** - Many exported types and methods undocumented

### ⚠️ Incomplete TODO Items

| Location | TODO |
|----------|------|
| `internal/auth/ntlm.go:509` | Verify checksum using verifyKey |
| `internal/rdp/rail.go:467` | RAIL implementation incomplete |

### ⚠️ Typo Found

**Location:** `internal/protocol/pdu/errors.go`

```go
ErrDeactiateAll  // Should be: ErrDeactivateAll
```

---

## 5. Test Coverage Audit

### Coverage by Package

| Package | Coverage | Status |
|---------|----------|--------|
| `protocol/tpkt` | 100% | ✅ Excellent |
| `protocol/x224` | 100% | ✅ Excellent |
| `protocol/encoding` | 97.4% | ✅ Excellent |
| `auth` | 95.6% | ✅ Excellent |
| `protocol/mcs` | 95.2% | ✅ Excellent |
| `cmd/server` | 91.1% | ✅ Excellent |
| `config` | 90.9% | ✅ Excellent |
| `logging` | 88.6% | ✅ Good |
| `codec/rfx` | 84.6% | ✅ Good |
| `protocol/fastpath` | 84.8% | ✅ Good |
| `protocol/pdu` | 84.4% | ✅ Good |
| `protocol/gcc` | 83.3% | ✅ Good |
| `protocol/audio` | 75.6% | 🟡 Acceptable |
| `handler` | 59.8% | 🟡 Needs improvement |
| `codec` | 49.4% | 🟡 Needs improvement |
| `rdp` | 45.3% | 🟡 Needs improvement |
| `web` | 0.0% | ⚠️ Go embed only |

### ⚠️ Skipped Tests

Some test files contain `t.Skip()` calls:
- `internal/rdp/client_extended_test.go`
- `internal/codec/rle_test.go`

**Recommendation:** Either fix or remove skipped tests.

---

## 6. Code Quality Audit

### 🔴 Large Functions

**`internal/handler/connect.go:handleWebSocket()`** - 175+ lines

This function handles:
- Credential validation
- RDP connection setup
- Goroutine management for bidirectional communication
- Error handling and cleanup

**Recommendation:** Extract into smaller functions:
- `validateCredentials()`
- `setupRDPConnection()`
- `startBidirectionalRelay()`

---

### 🔴 Global Mutable State

**`internal/codec/bitmap.go:29`**
```go
var currentPalette [256][4]byte
```

This global palette is modified during bitmap processing. Not thread-safe for concurrent connections.

**Recommendation:** Move palette into connection context or use sync.Mutex.

**`internal/rdp/get_update.go`**
```go
var updateCounter int
```

Similar thread-safety concern.

---

### 🟡 Ignored Errors

**`internal/handler/connect.go:261`**
```go
_ = wsConn.SetReadDeadline(time.Time{})
```

Deadline setting errors are silently ignored.

**Recommendation:** At minimum, log ignored errors at debug level.

---

### ✅ Quality Positives

- No `panic()` calls in non-test code
- Consistent error wrapping with `fmt.Errorf("context: %w", err)`
- Good use of constants over magic numbers
- Proper resource cleanup with `defer`

---

## 7. Recommendations Summary

### Immediate (Security)

1. **Fix JSON injection** in `sendError()` - use `json.Marshal()`
2. **Log warning** when `InsecureSkipVerify` is enabled
3. **Complete NTLM checksum verification** or document limitations

### Short-term (Quality)

4. **Refactor RLE codecs** to reduce duplication (~1,500 lines)
5. **Split `handleWebSocket()`** into smaller functions
6. **Fix global mutable state** in codec package
7. **Add godoc comments** to exported functions in cmd/server

### Medium-term (Architecture)

8. **Split `internal/rdp`** into subpackages (auth, channels, session)
9. **Reorganize `protocol/pdu`** by logical concern
10. **Create `internal/testutil`** for shared test helpers
11. **Improve test coverage** in handler (59.8%) and rdp (45.3%) packages

### Low Priority (Polish)

12. Fix typo: `ErrDeactiateAll` → `ErrDeactivateAll`
13. Remove or fix skipped tests
14. Complete RAIL implementation or document as unsupported

---

## Appendix: Files Reviewed

- 87 test files
- 44+ Go files in internal/rdp
- 38 Go files in protocol/pdu
- 20 README.md files
- All JavaScript files in web/src/js
- Configuration and build files (Makefile, Dockerfile, CI workflows)
