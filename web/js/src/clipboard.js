/**
 * Clipboard handling for RDP client
 * Provides clipboard buffer UI integration for text transfer to remote
 * @module clipboard
 */

import { Logger } from './logger.js';

/**
 * Clipboard handling mixin - adds clipboard functionality to Client
 */
export const ClipboardMixin = {
    /**
     * Initialize clipboard support
     */
    initClipboardSupport() {
        this.clipboardApiSupported = !!(navigator.clipboard && navigator.clipboard.writeText);
        Logger.info("Clipboard", `API supported: ${this.clipboardApiSupported}`);
    },
    
    /**
     * Type text to remote by sending key events
     * This simulates typing the text character by character
     * @param {string} text
     */
    typeTextToRemote(text) {
        if (!this.connected || !text) return;
        
        // Limit text length to prevent flooding
        const maxLength = 4096;
        if (text.length > maxLength) {
            text = text.substring(0, maxLength);
            this.showUserWarning('Text truncated to ' + maxLength + ' characters');
        }
        
        Logger.info("Clipboard", `Typing ${text.length} chars to remote`);
        
        // Type each character with a small delay to prevent overwhelming the connection
        let index = 0;
        const typeNext = () => {
            if (index >= text.length || !this.connected) return;
            
            const char = text[index];
            this.sendCharacter(char);
            index++;
            
            // Small delay between characters for reliability
            if (index < text.length) {
                setTimeout(typeNext, 10);
            }
        };
        
        typeNext();
    },
    
    /**
     * Send a single character as key press
     * @param {string} char
     */
    sendCharacter(char) {
        const code = this.charToKeyCode(char);
        if (!code) {
            Logger.debug("Clipboard", `No mapping for char code: ${char.charCodeAt(0)}`);
            return;
        }
        
        const needsShift = this.charNeedsShift(char);
        
        // Press shift if needed
        if (needsShift) {
            const shiftDown = new KeyboardEventKeyDown('ShiftLeft');
            if (shiftDown.keyCode !== undefined) {
                this.queueInput(shiftDown.serialize(), false);
            }
        }
        
        // Send keydown then keyup for the character
        const keyDown = new KeyboardEventKeyDown(code);
        const keyUp = new KeyboardEventKeyUp(code);
        
        if (keyDown.keyCode !== undefined) {
            this.queueInput(keyDown.serialize(), false);
            this.queueInput(keyUp.serialize(), false);
        }
        
        // Release shift if it was pressed
        if (needsShift) {
            const shiftUp = new KeyboardEventKeyUp('ShiftLeft');
            if (shiftUp.keyCode !== undefined) {
                this.queueInput(shiftUp.serialize(), false);
            }
        }
    },
    
    /**
     * Check if character needs shift key
     * @param {string} char
     * @returns {boolean}
     */
    charNeedsShift(char) {
        // Uppercase letters
        if (char >= 'A' && char <= 'Z') return true;
        
        // Shifted symbols
        const shiftedChars = '!@#$%^&*()_+{}|:"<>?~';
        return shiftedChars.includes(char);
    },
    
    /**
     * Map character to JavaScript key code
     * @param {string} char
     * @returns {string|null}
     */
    charToKeyCode(char) {
        const charCode = char.charCodeAt(0);
        
        // Letters (a-z, A-Z) - use uppercase key code
        if ((charCode >= 65 && charCode <= 90) || (charCode >= 97 && charCode <= 122)) {
            return 'Key' + char.toUpperCase();
        }
        
        // Numbers (0-9)
        if (charCode >= 48 && charCode <= 57) {
            return 'Digit' + char;
        }
        
        // Special characters - map to their key codes
        const specialMap = {
            ' ': 'Space',
            '\n': 'Enter',
            '\r': 'Enter',
            '\t': 'Tab',
            '.': 'Period',
            ',': 'Comma',
            ';': 'Semicolon',
            ':': 'Semicolon',  // Shift+;
            "'": 'Quote',
            '"': 'Quote',      // Shift+'
            '/': 'Slash',
            '?': 'Slash',      // Shift+/
            '\\': 'Backslash',
            '|': 'Backslash',  // Shift+\
            '[': 'BracketLeft',
            '{': 'BracketLeft', // Shift+[
            ']': 'BracketRight',
            '}': 'BracketRight', // Shift+]
            '-': 'Minus',
            '_': 'Minus',      // Shift+-
            '=': 'Equal',
            '+': 'Equal',      // Shift+=
            '`': 'Backquote',
            '~': 'Backquote',  // Shift+`
            '!': 'Digit1',     // Shift+1
            '@': 'Digit2',     // Shift+2
            '#': 'Digit3',     // Shift+3
            '$': 'Digit4',     // Shift+4
            '%': 'Digit5',     // Shift+5
            '^': 'Digit6',     // Shift+6
            '&': 'Digit7',     // Shift+7
            '*': 'Digit8',     // Shift+8
            '(': 'Digit9',     // Shift+9
            ')': 'Digit0',     // Shift+0
            '<': 'Comma',      // Shift+,
            '>': 'Period',     // Shift+.
        };
        
        return specialMap[char] || null;
    },
    
    /**
     * Copy text to local clipboard (for UI use)
     * @param {string} text
     */
    async copyToLocalClipboard(text) {
        if (!this.clipboardApiSupported) return false;
        
        try {
            await navigator.clipboard.writeText(text);
            return true;
        } catch (err) {
            Logger.warn("Clipboard", `Write failed: ${err.message}`);
            return false;
        }
    }
};

export default ClipboardMixin;
