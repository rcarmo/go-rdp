/**
 * RDP Web Client - Entry point
 * Exports the Client class, Logger, and WASM codec for use in browser
 * @module index
 */

import { Client, Logger } from './client.js';
import { WASMCodec, RFXDecoder, isWASMSupported } from './wasm.js';
import { FallbackCodec } from './codec-fallback.js';

// Export to global scope for browser use
if (typeof window !== 'undefined') {
    window.Client = Client;
    window.Logger = Logger;
    window.WASMCodec = WASMCodec;
    window.RFXDecoder = RFXDecoder;
    window.FallbackCodec = FallbackCodec;
    window.isWASMSupported = isWASMSupported;
}

export { Client, Logger, WASMCodec, RFXDecoder, FallbackCodec, isWASMSupported };
export default Client;
