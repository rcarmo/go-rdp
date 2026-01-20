/**
 * RDP Web Client - Entry point
 * Exports the Client class and Logger for use in browser
 * @module index
 */

import { Client, Logger } from './client.js';

// Export to global scope for browser use
if (typeof window !== 'undefined') {
    window.Client = Client;
    window.Logger = Logger;
}

export { Client, Logger };
export default Client;
