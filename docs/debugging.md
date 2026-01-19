# Debugging

## Client-side debug logging

The browser client has a centralized logger that can be enabled/disabled for debugging.

Enable debug logging in the browser console:

```javascript
Logger.enable()  // Enables debug logging and saves preference
```

Disable debug logging:

```javascript
Logger.disable()  // Disables debug logging and saves preference
```

Check status:

```javascript
Logger.isEnabled()  // Returns true/false
```

### What gets logged

Debug logs include:

- RDP protocol updates (bitmaps, cursors, etc.)
- Cursor changes and cache operations
- WebSocket connection events

The setting persists in localStorage across sessions.
