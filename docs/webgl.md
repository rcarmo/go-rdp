# WebGL Renderer

## Summary
The WebGL-based renderer accelerates bitmap display while preserving the 2D canvas path as a fallback. The WebGL renderer *only* blits already-decoded RGBA frames to the screen (no shader-based decoding). WebGL2 is preferred, with WebGL1 as a fallback for browser compatibility.

## Implementation Status: ✅ Complete

The WebGL renderer has been implemented in:
- `web/src/js/renderer.js` - Renderer interface + CanvasRenderer
- `web/src/js/webgl-renderer.js` - WebGL1/2 renderer implementation

## Goals
- Reduce per-frame render time by replacing `putImageData(...)` with GPU texture uploads and draw calls.
- Keep the existing decode path (WASM/JS) unchanged.
- Preserve current behavior for non-WebGL browsers (fallback to canvas).
- Provide visibility and safe fallback on WebGL errors or context loss.

## Non-Goals
- No shader-based decoding, colorspace conversion, or bitmap decompression in GPU.
- No change to RDP protocol handling, codec negotiation, or server behavior.
- No new browser dependencies beyond WebGL support.

## Feasibility: Shader-Based Decode (RLE/Planar/NSCodec/RemoteFX)
This section evaluates moving decode stages from WASM/JS to GPU shaders. These ideas are *feasible* in theory but have high complexity and limited cross-browser reliability compared to the current CPU/WASM decode path.

### 1) RGBA Blit Only (baseline)
**Feasible:** Yes  
**Complexity:** Low  
**Notes:** This is the plan above. It avoids shader decoding and uses GPU only for presentation.

### 2) Interleaved RLE / Planar Decode in Shaders
**Feasible:** Limited and costly  
**Complexity:** High  
**Risks / Constraints:**
- RLE is inherently sequential and branch-heavy, which is inefficient in fragment shaders.
- WebGL does not expose general-purpose compute shaders; workarounds require multiple render passes, custom packing, and careful texture layouts.
- Planar codecs require per-plane RLE expansion and combining planes; still branch-heavy.
- Debugging shader decode is difficult and error-prone.

**Recommendation:** Not recommended for this project. Keep RLE/Planar decode in WASM/JS and only blit RGBA.

### 3) NSCodec Decode in Shaders
**Feasible:** Very limited  
**Complexity:** Very high  
**Risks / Constraints:**
- NSCodec uses AYCoCg transform + RLE planes with variable-length segments.
- Requires multiple dependent reads, branching, and potential recursive logic.
- Would need a multi-pass pipeline to unpack bitstreams, reconstruct planes, then transform to RGB.
- Likely slower or unstable compared to WASM in real browsers.

**Recommendation:** Not feasible within reasonable effort.

### 4) RemoteFX Decode in Shaders
**Feasible:** Theoretical, but not practical in WebGL  
**Complexity:** Extremely high  
**Risks / Constraints:**
- RemoteFX involves RLGR entropy decode + DWT + inverse quantization.
- RLGR is bitstream-heavy and sequential; DWT requires multiple passes with local dependencies.
- WebGL fragment shaders are not designed for this class of algorithm.

**Recommendation:** Not feasible. Keep RemoteFX decode in WASM.

### Summary Table
| Decode Stage | Feasibility | Complexity | Recommendation |
|-------------|-------------|------------|----------------|
| RGBA blit only | High | Low | ✅ Do it |
| RLE/Planar in shader | Low | High | ❌ Avoid |
| NSCodec in shader | Very low | Very high | ❌ Avoid |
| RemoteFX in shader | Very low | Extreme | ❌ Avoid |

### Pragmatic Path Forward
1. Implement WebGL renderer for RGBA blit only.
2. Continue using WASM for decode (RLE/Planar/NSCodec/RemoteFX).
3. Measure CPU decode vs GPU blit overhead to quantify gains.

## Current Rendering Pipeline (Baseline)
Location: `web/src/js/graphics.js`

1. `handleBitmap()` → `parseBitmapUpdate(...)`
2. `processBitmapData(...)` decodes to RGBA using:
   - WASM: `WASMCodec.processBitmap(...)`
   - Fallback: `FallbackCodec.processBitmap(...)`
3. Render: `ctx.putImageData(new ImageData(rgba, width, height), x, y)`

Main bottleneck: `putImageData` is CPU-bound and causes large memory copies at high resolutions.

## Proposed Architecture

### Renderer Abstraction
Introduce a renderer interface with two implementations:
- **CanvasRenderer** (current behavior)
- **WebGLRenderer** (new path)

Suggested interface:
```js
interface Renderer {
  init(canvas: HTMLCanvasElement): boolean;
  resize(width: number, height: number): void;
  drawRGBA(x: number, y: number, width: number, height: number, rgba: Uint8Array): void;
  clear(): void;
  destroy(): void;
}
```

### WebGL Renderer Strategy (RGBA Blit)
- Create a single RGBA texture sized to the desktop resolution.
- For each bitmap rectangle:
  - Use `texSubImage2D(...)` to upload the sub-rectangle.
  - Draw a textured quad that covers the full canvas (or use scissoring to limit updates).
- Maintain WebGL1 fallback:
  - Use `gl.texImage2D`/`gl.texSubImage2D` with `gl.RGBA`, `gl.UNSIGNED_BYTE`.

### Feature Detection & Fallback
1. Attempt WebGL2 context creation.
2. If WebGL2 fails, try WebGL1.
3. On error or context loss → log and fallback to CanvasRenderer.

WebGL context loss handler:
```js
canvas.addEventListener('webglcontextlost', (e) => {
  e.preventDefault();
  // fall back to CanvasRenderer
});
```

## Integration Points

### GraphicsMixin.initGraphics()
Add renderer selection:
```js
this.renderer = new WebGLRenderer();
if (!this.renderer.init(this.canvas)) {
  this.renderer = new CanvasRenderer(this.canvas, this.ctx);
}
```

### GraphicsMixin.processBitmapData(...)
Replace direct canvas draw with:
```js
this.renderer.drawRGBA(bitmapData.destLeft, bitmapData.destTop, width, height, rgba);
```

### GraphicsMixin.clearCanvas(...)
Delegate to renderer:
```js
this.renderer.clear();
```

### Connection teardown
Call `renderer.destroy()` in `deinitialize()` to release GL resources.

## Performance Considerations

### Texture Sizing
- Use a single full-size texture sized to desktop resolution.
- Recreate texture on resize or resolution change.

### Dirty Rectangles
- Use rectangle-based `texSubImage2D(...)`.
- Optional: `gl.scissor(...)` to limit draw region and reduce fill cost.

### Batching
- If multiple rectangles per update, perform multiple `texSubImage2D(...)` calls and a single draw.

### Memory
- RGBA textures are large: width × height × 4 bytes.
- Ensure GPU memory budget is respected (log warnings on failures).

### Metrics to Track (Debug Mode)
- Texture upload time per frame
- Draw time per frame
- Frames per second (FPS) and average bandwidth

## Phased Implementation Plan

### Phase 1: Infrastructure
- Add `web/src/js/renderer.js` with CanvasRenderer wrapper.
- Add `web/src/js/webgl-renderer.js` with basic WebGL init + single texture upload.
- Keep all existing rendering intact, only change draw call target.

### Phase 2: Wiring
- Modify `graphics.js` to use renderer abstraction.
- Ensure WebGL fallback to CanvasRenderer on init failure.
- Add logs: `"[RDP Client] Active renderer: WebGL2/WebGL1/Canvas"`.

### Phase 3: Hardening
- Handle WebGL context loss.
- Add defensive checks for texture creation/upload failures.
- Ensure `destroy()` cleans GL resources.

### Phase 4: Optimization
- Implement scissor rects for dirty rectangles.
- Avoid full redraw when only a small area changes.
- Consider double-buffering if tearing appears.

### Phase 5: Testing and Rollback
- **Unit tests**: renderer selection logic, fallback on init failure.
- **Integration tests**: ensure CanvasRenderer works when WebGL disabled.
- **Manual tests**: large resolutions, long sessions, context loss.
- **Rollback**: keep `CanvasRenderer` path as default if WebGL fails.

## Config and UX
Optional configuration:
- `?disableWebGL=1` to force CanvasRenderer.
- Console log to indicate active renderer and fallback events.

## File-Level Changes (Implemented)
- `web/src/js/renderer.js`: renderer interface + CanvasRenderer.
- `web/src/js/webgl-renderer.js`: WebGL1/2 renderer.
- `web/src/js/graphics.js`: routes drawing to renderer.
- `web/src/js/client.js`: cleanup on disconnect.
- `docs/webgl.md`: this document.

## Acceptance Criteria
- WebGL path renders correctly at standard resolutions (1080p/1440p).
- If WebGL fails or context is lost, session continues via CanvasRenderer.
- No behavior regression for non-WebGL browsers.
- Performance improvement visible in large resolutions or high frame rates.
