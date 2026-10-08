import 'dart:io';
import 'dart:typed_data';

/// Minimal streaming SHA-256 (FIPS 180-4), so the wrapper has no
/// dependencies. Used to verify bundled and downloaded core binaries.
class Sha256 {
  static const List<int> _k = [
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, //
    0x923f82a4, 0xab1c5ed5, 0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
    0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174, 0xe49b69c1, 0xefbe4786,
    0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147,
    0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
    0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85, 0xa2bfe8a1, 0xa81a664b,
    0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a,
    0x5b9cca4f, 0x682e6ff3, 0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
    0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
  ];

  final Uint32List _h = Uint32List.fromList([
    0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, //
    0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
  ]);
  final Uint8List _block = Uint8List(64);
  final Uint32List _w = Uint32List(64);
  int _blockLen = 0;
  int _total = 0;
  bool _closed = false;

  /// Feeds more bytes.
  void add(List<int> data) {
    if (_closed) throw StateError('Sha256 already closed');
    _total += data.length;
    var i = 0;
    if (_blockLen > 0) {
      while (_blockLen < 64 && i < data.length) {
        _block[_blockLen++] = data[i++];
      }
      if (_blockLen < 64) return;
      _compress(_block, 0);
      _blockLen = 0;
    }
    final bytes = data is Uint8List ? data : Uint8List.fromList(data);
    while (data.length - i >= 64) {
      _compress(bytes, i);
      i += 64;
    }
    while (i < data.length) {
      _block[_blockLen++] = data[i++];
    }
  }

  /// Finishes and returns the lowercase hex digest.
  String close() {
    if (_closed) throw StateError('Sha256 already closed');
    final bits = _total * 8;
    final pad = Uint8List(((_blockLen < 56) ? 56 : 120) - _blockLen + 8);
    pad[0] = 0x80;
    final bd = ByteData.sublistView(pad);
    bd.setUint32(pad.length - 8, (bits ~/ 0x100000000) & 0xffffffff);
    bd.setUint32(pad.length - 4, bits & 0xffffffff);
    final saved = _total;
    add(pad);
    _total = saved;
    _closed = true;
    final sb = StringBuffer();
    for (final v in _h) {
      sb.write(v.toRadixString(16).padLeft(8, '0'));
    }
    return sb.toString();
  }

  static int _rotr(int x, int n) => ((x >> n) | (x << (32 - n))) & 0xffffffff;

  void _compress(Uint8List b, int off) {
    final w = _w;
    for (var t = 0; t < 16; t++) {
      final j = off + t * 4;
      w[t] = (b[j] << 24) | (b[j + 1] << 16) | (b[j + 2] << 8) | b[j + 3];
    }
    for (var t = 16; t < 64; t++) {
      final x = w[t - 15], y = w[t - 2];
      final s0 = _rotr(x, 7) ^ _rotr(x, 18) ^ (x >> 3);
      final s1 = _rotr(y, 17) ^ _rotr(y, 19) ^ (y >> 10);
      w[t] = (w[t - 16] + s0 + w[t - 7] + s1) & 0xffffffff;
    }
    var a = _h[0], bb = _h[1], c = _h[2], d = _h[3];
    var e = _h[4], f = _h[5], g = _h[6], h = _h[7];
    for (var t = 0; t < 64; t++) {
      final s1 = _rotr(e, 6) ^ _rotr(e, 11) ^ _rotr(e, 25);
      final ch = (e & f) ^ ((~e & 0xffffffff) & g);
      final t1 = (h + s1 + ch + _k[t] + w[t]) & 0xffffffff;
      final s0 = _rotr(a, 2) ^ _rotr(a, 13) ^ _rotr(a, 22);
      final maj = (a & bb) ^ (a & c) ^ (bb & c);
      final t2 = (s0 + maj) & 0xffffffff;
      h = g;
      g = f;
      f = e;
      e = (d + t1) & 0xffffffff;
      d = c;
      c = bb;
      bb = a;
      a = (t1 + t2) & 0xffffffff;
    }
    _h[0] = (_h[0] + a) & 0xffffffff;
    _h[1] = (_h[1] + bb) & 0xffffffff;
    _h[2] = (_h[2] + c) & 0xffffffff;
    _h[3] = (_h[3] + d) & 0xffffffff;
    _h[4] = (_h[4] + e) & 0xffffffff;
    _h[5] = (_h[5] + f) & 0xffffffff;
    _h[6] = (_h[6] + g) & 0xffffffff;
    _h[7] = (_h[7] + h) & 0xffffffff;
  }
}

/// SHA-256 of [data] as lowercase hex.
String sha256Hex(List<int> data) => (Sha256()..add(data)).close();

/// SHA-256 of a file, streamed.
Future<String> sha256OfFile(File file) async {
  final s = Sha256();
  await for (final chunk in file.openRead()) {
    s.add(chunk);
  }
  return s.close();
}
