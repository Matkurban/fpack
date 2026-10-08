import 'dart:convert';
import 'dart:io';

import 'package:fpack/src/sha256.dart';
import 'package:test/test.dart';

void main() {
  test('known vectors', () {
    expect(
      sha256Hex([]),
      'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
    );
    expect(
      sha256Hex(utf8.encode('abc')),
      'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad',
    );
    expect(
      sha256Hex(
        utf8.encode('abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq'),
      ),
      '248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1',
    );
  });

  test('streaming in odd chunk sizes equals one-shot', () {
    final data = List<int>.generate(100003, (i) => (i * 31 + 7) & 0xff);
    final whole = sha256Hex(data);
    for (final size in [1, 13, 63, 64, 65, 4096]) {
      final s = Sha256();
      for (var i = 0; i < data.length; i += size) {
        s.add(data.sublist(i, i + size > data.length ? data.length : i + size));
      }
      expect(s.close(), whole, reason: 'chunk $size');
    }
  });

  test('padding boundaries (55/56/64 bytes)', () {
    // Cross-checked with sha256sum.
    expect(
      sha256Hex(List.filled(55, 0x61)),
      '9f4390f8d30c2dd92ec9f095b65e2b9ae9b0a925a5258e241c9f1e910f734318',
    );
    expect(
      sha256Hex(List.filled(56, 0x61)),
      'b35439a4ac6f0948b6d6f9e3c6af0f5f590ce20f1bde7090ef7970686ec6738a',
    );
    expect(
      sha256Hex(List.filled(64, 0x61)),
      'ffe054fe7ae0cb6dc65c3af9b61d5209f439851db43d0ba5997337df154668eb',
    );
  });

  test('file hashing', () async {
    final dir = Directory.systemTemp.createTempSync('fpack_sha');
    addTearDown(() => dir.deleteSync(recursive: true));
    final f = File('${dir.path}/x')..writeAsStringSync('abc');
    expect(
      await sha256OfFile(f),
      'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad',
    );
  });
}
