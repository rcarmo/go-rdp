/**
 * End-to-end browser tests for RDP HTML5 Client
 * 
 * These tests verify:
 * - Page loads correctly
 * - WASM module initializes
 * - WebSocket connections work
 * - Cross-browser compatibility
 * 
 * Run: npx playwright test
 * Run single browser: npx playwright test --project=firefox
 */

const { test, expect } = require('@playwright/test');

test.describe('Page Load', () => {
  test('loads index.html with required elements', async ({ page }) => {
    await page.goto('/');
    
    // Check page title
    await expect(page).toHaveTitle(/RDP/);
    
    // Check canvas exists (may be hidden initially)
    const canvas = page.locator('#canvas');
    await expect(canvas).toHaveCount(1);
    
    // Check connection form elements
    await expect(page.locator('#host')).toBeVisible();
    await expect(page.locator('#user')).toBeVisible();
    await expect(page.locator('#password')).toBeVisible();
    await expect(page.locator('#connect-btn')).toBeVisible();
  });

  test('loads JavaScript bundle', async ({ page }) => {
    const errors = [];
    page.on('pageerror', err => errors.push(err.message));
    
    await page.goto('/');
    await page.waitForTimeout(500);
    
    // Check RDP global is defined
    const rdpDefined = await page.evaluate(() => typeof window.RDP !== 'undefined');
    expect(rdpDefined).toBe(true);
    
    // No JS errors on load
    expect(errors).toHaveLength(0);
  });
});

test.describe('WASM Module', () => {
  test('WASM is supported', async ({ page }) => {
    await page.goto('/');
    
    const wasmSupported = await page.evaluate(() => {
      return typeof WebAssembly === 'object' && typeof WebAssembly.instantiate === 'function';
    });
    expect(wasmSupported).toBe(true);
  });

  test('WASM codec initializes successfully', async ({ page }) => {
    await page.goto('/');
    
    // Wait for WASM to load
    const result = await page.evaluate(async () => {
      // Give WASM time to initialize
      await new Promise(r => setTimeout(r, 1000));
      
      if (typeof window.RDP === 'undefined') {
        return { error: 'RDP not defined' };
      }
      
      const codec = window.RDP.WASMCodec;
      if (!codec) {
        return { error: 'WASMCodec not defined' };
      }
      
      // Try to initialize if not already
      if (!codec.isReady()) {
        try {
          await codec.init('js/rle/rle.wasm');
        } catch (e) {
          return { error: `Init failed: ${e.message}` };
        }
      }
      
      return {
        supported: codec.isSupported(),
        ready: codec.isReady(),
        error: codec.getInitError()
      };
    });
    
    expect(result.error).toBeNull();
    expect(result.supported).toBe(true);
    expect(result.ready).toBe(true);
  });

  test('goRLE global is available after WASM init', async ({ page }) => {
    await page.goto('/');
    
    const goRLEAvailable = await page.evaluate(async () => {
      await new Promise(r => setTimeout(r, 1000));
      
      if (window.RDP?.WASMCodec && !window.RDP.WASMCodec.isReady()) {
        await window.RDP.WASMCodec.init('js/rle/rle.wasm');
      }
      
      return typeof window.goRLE !== 'undefined';
    });
    
    expect(goRLEAvailable).toBe(true);
  });
});

test.describe('WebSocket Handling', () => {
  test('WebSocket binaryType is set correctly', async ({ page }) => {
    await page.goto('/');
    
    // Fill in dummy connection details
    await page.fill('#host', 'test.example.com');
    await page.fill('#user', 'testuser');
    await page.fill('#password', 'testpass');
    
    // Intercept WebSocket creation to check binaryType
    const binaryType = await page.evaluate(() => {
      return new Promise((resolve, reject) => {
        const OriginalWebSocket = window.WebSocket;
        window.WebSocket = function(url, protocols) {
          const ws = new OriginalWebSocket(url, protocols);
          // Check binaryType after a tick (it's set after construction)
          setTimeout(() => {
            resolve(ws.binaryType);
            ws.close();
          }, 100);
          return ws;
        };
        window.WebSocket.prototype = OriginalWebSocket.prototype;
        
        // Trigger connection via button click
        const btn = document.getElementById('connect-btn');
        if (!btn) {
          reject(new Error('connect-btn not found'));
          return;
        }
        btn.click();
      });
    });
    
    expect(binaryType).toBe('arraybuffer');
  });
});

test.describe('Fallback Codec', () => {
  test('FallbackCodec is available', async ({ page }) => {
    await page.goto('/');
    
    const fallbackAvailable = await page.evaluate(() => {
      return typeof window.RDP?.FallbackCodec !== 'undefined';
    });
    
    expect(fallbackAvailable).toBe(true);
  });

  test('FallbackCodec can convert RGB565 to RGBA', async ({ page }) => {
    await page.goto('/');
    
    const result = await page.evaluate(() => {
      const codec = window.RDP?.FallbackCodec;
      if (!codec) return { error: 'FallbackCodec not available' };
      
      // Test RGB565 conversion: blue pixel (0x001F)
      const src = new Uint8Array([0x1F, 0x00]); // Little-endian blue
      const dst = new Uint8Array(4);
      
      const success = codec.rgb565ToRGBA(src, dst);
      return {
        success,
        r: dst[0],
        g: dst[1],
        b: dst[2],
        a: dst[3]
      };
    });
    
    expect(result.success).toBe(true);
    expect(result.b).toBeGreaterThan(200); // Blue should be high
    expect(result.r).toBeLessThan(50);     // Red should be low
    expect(result.a).toBe(255);            // Alpha should be 255
  });
});
