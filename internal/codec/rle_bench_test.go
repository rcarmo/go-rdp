package codec

import (
"testing"
)

// Generate test data for benchmarks
func generateRLETestData(size int) (src []byte, rowDelta int) {
// Create mixed RLE data: color runs + color images + background runs
src = make([]byte, 0, size)
rowDelta = 128 * 2 // 128 pixels wide, 16-bit

// Add various RLE orders to simulate real-world data
for len(src) < size-20 {
// Color run (0x63 = code 3, length 3)
src = append(src, 0x63, 0x12, 0x34) // 16-bit pixel

// Background run (0x05 = code 0, length 5)
src = append(src, 0x05)

// Color image (0x84 = code 4, length 4)
src = append(src, 0x84, 0xAB, 0xCD, 0xEF, 0x12, 0x34, 0x56, 0x78, 0x9A)

// Foreground run (0x22 = code 1, length 2)
src = append(src, 0x22)
}

return src, rowDelta
}

func BenchmarkRLEDecompress16_Generic(b *testing.B) {
src, rowDelta := generateRLETestData(1024)
dest := make([]byte, 64*1024) // Large enough destination

b.ResetTimer()
b.ReportAllocs()

for i := 0; i < b.N; i++ {
RLEDecompress16(src, dest, rowDelta)
}
}

func BenchmarkRLEDecompress8_Generic(b *testing.B) {
// 8-bit test data
src := make([]byte, 0, 1024)
for len(src) < 1000 {
src = append(src, 0x63, 0xAB)       // Color run, len=3, pixel=0xAB
src = append(src, 0x05)              // Background run, len=5
src = append(src, 0x84, 0x11, 0x22, 0x33, 0x44) // Color image, len=4
}
dest := make([]byte, 64*1024)
rowDelta := 128

b.ResetTimer()
b.ReportAllocs()

for i := 0; i < b.N; i++ {
RLEDecompress8(src, dest, rowDelta)
}
}

func BenchmarkRLEDecompress24_Generic(b *testing.B) {
src := make([]byte, 0, 1024)
for len(src) < 1000 {
src = append(src, 0x63, 0x12, 0x34, 0x56) // Color run, len=3, 24-bit pixel
src = append(src, 0x05)                    // Background run
}
dest := make([]byte, 64*1024)
rowDelta := 128 * 3

b.ResetTimer()
b.ReportAllocs()

for i := 0; i < b.N; i++ {
RLEDecompress24(src, dest, rowDelta)
}
}

func BenchmarkRLEDecompress32_Generic(b *testing.B) {
src := make([]byte, 0, 1024)
for len(src) < 1000 {
src = append(src, 0x63, 0x12, 0x34, 0x56, 0x78) // Color run, len=3, 32-bit pixel
src = append(src, 0x05)                          // Background run
}
dest := make([]byte, 64*1024)
rowDelta := 128 * 4

b.ResetTimer()
b.ReportAllocs()

for i := 0; i < b.N; i++ {
RLEDecompress32(src, dest, rowDelta)
}
}

// Benchmark pixel read/write operations directly
func BenchmarkPixelOps16(b *testing.B) {
data := make([]byte, 1024)
b.ResetTimer()

for i := 0; i < b.N; i++ {
for j := 0; j < 512; j++ {
idx := j * 2
pixel := ReadPixel16(data, idx)
WritePixel16(data, idx, pixel+1)
}
}
}

func BenchmarkPixelOps24(b *testing.B) {
data := make([]byte, 1536)
b.ResetTimer()

for i := 0; i < b.N; i++ {
for j := 0; j < 512; j++ {
idx := j * 3
pixel := ReadPixel24(data, idx)
WritePixel24(data, idx, pixel+1)
}
}
}

func BenchmarkPixelOps32(b *testing.B) {
data := make([]byte, 2048)
b.ResetTimer()

for i := 0; i < b.N; i++ {
for j := 0; j < 512; j++ {
idx := j * 4
pixel := ReadPixel32(data, idx)
WritePixel32(data, idx, pixel+1)
}
}
}

// Direct inline implementation for comparison (simulates old code)
func rleDecompress16Inline(src []byte, dest []byte, rowDelta int) bool {
srcIdx := 0
destIdx := 0
var fgPel uint16 = 0xFFFF
fInsertFgPel := false
fFirstLine := true

for srcIdx < len(src) && destIdx < len(dest) {
if fFirstLine && destIdx >= rowDelta {
fFirstLine = false
fInsertFgPel = false
}

code := ExtractCodeID(src[srcIdx])

if code == RegularBgRun || code == MegaMegaBgRun {
runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
srcIdx = nextIdx

if fFirstLine {
if fInsertFgPel {
if destIdx+1 < len(dest) {
dest[destIdx] = byte(fgPel & 0xFF)
dest[destIdx+1] = byte((fgPel >> 8) & 0xFF)
}
destIdx += 2
runLength--
}
for runLength > 0 && destIdx < len(dest) {
dest[destIdx] = 0
dest[destIdx+1] = 0
destIdx += 2
runLength--
}
} else {
if fInsertFgPel {
var prevPel uint16
if destIdx-rowDelta+1 < len(dest) {
prevPel = uint16(dest[destIdx-rowDelta]) | (uint16(dest[destIdx-rowDelta+1]) << 8)
}
xorPel := prevPel ^ fgPel
dest[destIdx] = byte(xorPel & 0xFF)
dest[destIdx+1] = byte((xorPel >> 8) & 0xFF)
destIdx += 2
runLength--
}
for runLength > 0 && destIdx < len(dest) {
var prevPel uint16
if destIdx-rowDelta+1 < len(dest) {
prevPel = uint16(dest[destIdx-rowDelta]) | (uint16(dest[destIdx-rowDelta+1]) << 8)
}
dest[destIdx] = byte(prevPel & 0xFF)
dest[destIdx+1] = byte((prevPel >> 8) & 0xFF)
destIdx += 2
runLength--
}
}
fInsertFgPel = true
continue
}

fInsertFgPel = false

if code == RegularColorRun || code == MegaMegaColorRun {
runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
srcIdx = nextIdx
var pixel uint16
if srcIdx+1 < len(src) {
pixel = uint16(src[srcIdx]) | (uint16(src[srcIdx+1]) << 8)
}
srcIdx += 2
for runLength > 0 && destIdx < len(dest) {
dest[destIdx] = byte(pixel & 0xFF)
dest[destIdx+1] = byte((pixel >> 8) & 0xFF)
destIdx += 2
runLength--
}
continue
}

srcIdx++
}
return true
}

func BenchmarkRLEDecompress16_Inline(b *testing.B) {
src, rowDelta := generateRLETestData(1024)
dest := make([]byte, 64*1024)

b.ResetTimer()
b.ReportAllocs()

for i := 0; i < b.N; i++ {
rleDecompress16Inline(src, dest, rowDelta)
}
}
