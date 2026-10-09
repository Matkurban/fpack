import 'dart:convert';

import 'package:flutter/material.dart';

/// Compile-time values from `--dart-define-from-file config/<flavor>.json`
/// (set by `build.dart_define_from_file` in fpack.yaml).
const apiBase = String.fromEnvironment('API_BASE', defaultValue: 'not set');
const flavorName = String.fromEnvironment('FLAVOR_NAME', defaultValue: 'none');

/// Set by `build.dart_define` in fpack.yaml (from `${BUILD_CHANNEL}`).
const buildChannel = String.fromEnvironment(
  'BUILD_CHANNEL',
  defaultValue: 'local',
);

void main() => runApp(const AuroraApp());

class AuroraApp extends StatelessWidget {
  const AuroraApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Aurora Notes',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(colorSchemeSeed: Colors.indigo),
      darkTheme: ThemeData(
        colorSchemeSeed: Colors.indigo,
        brightness: Brightness.dark,
      ),
      home: const NotesPage(),
    );
  }
}

class Note {
  const Note(this.title, this.body);

  factory Note.fromJson(Map<String, dynamic> json) =>
      Note(json['title'] as String, json['body'] as String);

  final String title;
  final String body;
}

Future<List<Note>> loadNotes(AssetBundle bundle) async {
  final raw = jsonDecode(await bundle.loadString('assets/data/notes.json'));
  return [
    for (final n in raw as List) Note.fromJson(n as Map<String, dynamic>),
  ];
}

class NotesPage extends StatelessWidget {
  const NotesPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Aurora Notes')),
      body: FutureBuilder<List<Note>>(
        future: loadNotes(DefaultAssetBundle.of(context)),
        builder: (context, snap) => ListView(
          padding: const EdgeInsets.symmetric(vertical: 16),
          children: [
            Center(child: Image.asset('assets/images/logo.png', height: 96)),
            const SizedBox(height: 8),
            ListTile(
              leading: const Icon(Icons.tune),
              title: Text('Flavor: $flavorName · channel: $buildChannel'),
              subtitle: const Text('API: $apiBase'),
            ),
            const Divider(),
            for (final note in snap.data ?? const <Note>[])
              ListTile(
                leading: const Icon(Icons.sticky_note_2_outlined),
                title: Text(note.title),
                subtitle: Text(note.body),
              ),
          ],
        ),
      ),
    );
  }
}
