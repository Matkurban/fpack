import 'dart:io';

import 'package:fpack/fpack.dart';

Future<void> main(List<String> arguments) async {
  final code = await runFpack(arguments);
  // Exit explicitly so signal subscriptions don't keep the VM alive.
  await stdout.flush();
  await stderr.flush();
  exit(code);
}
