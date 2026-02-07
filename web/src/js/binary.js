export default function BinaryReader(arrayBuffer) {
    this.arrayBuffer = arrayBuffer;
    this.dv = new DataView(arrayBuffer);
    this.offset = 0;
    this.length = arrayBuffer.byteLength;
}

BinaryReader.prototype.remaining = function() {
    return this.length - this.offset;
};

BinaryReader.prototype.hasBytes = function(count) {
    return this.offset + count <= this.length;
};

BinaryReader.prototype.uint8 = function () {
    if (!this.hasBytes(1)) {
        throw new RangeError('BinaryReader: uint8 read beyond buffer');
    }
    const data = this.dv.getUint8(this.offset);
    this.offset += 1;

    return data;
};

BinaryReader.prototype.uint16 = function (littleEndian) {
    if (!this.hasBytes(2)) {
        throw new RangeError('BinaryReader: uint16 read beyond buffer');
    }
    const data = this.dv.getUint16(this.offset, littleEndian);
    this.offset += 2;

    return data;
};

BinaryReader.prototype.uint32 = function (littleEndian) {
    if (!this.hasBytes(4)) {
        throw new RangeError('BinaryReader: uint32 read beyond buffer');
    }
    const data = this.dv.getUint32(this.offset, littleEndian);
    this.offset += 4;

    return data;
};

BinaryReader.prototype.blob = function (length) {
    if (!this.hasBytes(length)) {
        throw new RangeError('BinaryReader: blob read beyond buffer');
    }
    const data = new Uint8Array(this.arrayBuffer, this.offset, length);
    this.offset += length;

    return data;
};

BinaryReader.prototype.string = function (length) {
    if (!this.hasBytes(length)) {
        throw new RangeError('BinaryReader: string read beyond buffer');
    }
    const dec = new TextDecoder();
    const data = dec.decode(this.arrayBuffer.slice(this.offset, this.offset + length));
    this.offset += length;

    return data;
};

BinaryReader.prototype.skip = function(length) {
    if (!this.hasBytes(length)) {
        throw new RangeError('BinaryReader: skip beyond buffer');
    }
    this.offset += length;
};

export function BinaryWriter(array) {
    this.offset = 0;
    this.array = array;
    this.dv = new DataView(this.array);
}

BinaryWriter.prototype._checkBounds = function(bytes) {
    if (this.offset + bytes > this.array.byteLength) {
        throw new RangeError(`BinaryWriter: write of ${bytes} bytes at offset ${this.offset} exceeds buffer length ${this.array.byteLength}`);
    }
};

BinaryWriter.prototype.uint8 = function (value) {
    this._checkBounds(1);
    this.dv.setUint8(this.offset, value)

    this.offset += 1;
};

BinaryWriter.prototype.uint16 = function (value, littleEndian) {
    this._checkBounds(2);
    this.dv.setUint16(this.offset, value, littleEndian);

    this.offset += 2;
};

BinaryWriter.prototype.uint32 = function (value, littleEndian) {
    this._checkBounds(4);
    this.dv.setUint32(this.offset, value, littleEndian);

    this.offset += 4;
};

BinaryWriter.prototype.bytes = function (bytes) {
    const arr = new Uint8Array(bytes);
    this._checkBounds(arr.length);

    for (let i = 0; i < arr.length; i++) {
        this.dv.setUint8(this.offset, arr[i]);
        this.offset += 1;
    }
}

BinaryWriter.prototype.skip = function(length) {
    this._checkBounds(length);
    this.offset += length;
};
