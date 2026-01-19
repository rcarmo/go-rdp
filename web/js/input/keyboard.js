const FASTPATH_INPUT_KBDFLAGS_RELEASE = 0x01;

// Make functions available globally
window.KeyboardEventKeyDown = function KeyboardEventKeyDown(code) {
    this.keyCode = KeyMap[code];
};

window.KeyboardEventKeyDown.prototype.serialize = function () {
    const data = new ArrayBuffer(2);
    const w = new BinaryWriter(data);

    const eventFlags = 0;
    const eventHeader = ((FASTPATH_INPUT_EVENT_SCANCODE & 0x7) << 5) | (eventFlags & 0x1f);

    w.uint8(eventHeader);
    w.uint8(this.keyCode);

    return data;
};

window.KeyboardEventKeyUp = function KeyboardEventKeyUp(code) {
    this.keyCode = KeyMap[code];
};

window.KeyboardEventKeyUp.prototype.serialize = function () {
    const data = new ArrayBuffer(2);
    const w = new BinaryWriter(data);

    const eventFlags = FASTPATH_INPUT_KBDFLAGS_RELEASE;
    const eventHeader = ((FASTPATH_INPUT_EVENT_SCANCODE & 0x7) << 5) | (eventFlags & 0x1f);

    w.uint8(eventHeader);
    w.uint8(this.keyCode);

    return data;
};
