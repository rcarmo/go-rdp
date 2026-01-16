const FASTPATH_INPUT_KBDFLAGS_RELEASE = 0x01;

// Make functions available globally
window.KeyboardEventKeyDown = function KeyboardEventKeyDown(code) {
    this.keyCode = KeyMap[code];
};

window.KeyboardEventKeyDown.prototype.serialize = function () {
    const data = new ArrayBuffer(3);
    const w = new BinaryWriter(data);

    const eventFlags = 0;
    const eventCode = (FASTPATH_INPUT_EVENT_SCANCODE & 0x3) << 5;
    const eventHeader = eventFlags | eventCode;

    w.uint8(eventHeader);
    // w.uint16(this.keyCode, true);
    w.uint8(this.keyCode);

    return data;
};

window.KeyboardEventKeyUp = function KeyboardEventKeyUp(code) {
    this.keyCode = KeyMap[code];
};

window.KeyboardEventKeyUp.prototype.serialize = function () {
    const data = new ArrayBuffer(3);
    const w = new BinaryWriter(data);

    const eventFlags = (FASTPATH_INPUT_KBDFLAGS_RELEASE) & 0x1f;
    const eventCode = (FASTPATH_INPUT_EVENT_SCANCODE & 0x7) << 5;
    const eventHeader = eventFlags | eventCode;

    w.uint8(eventHeader);
    // w.uint16(this.keyCode, true);
    w.uint8(this.keyCode);

    return data;
};
