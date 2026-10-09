import 'dart:convert';

/// Small TextMate grammars for the code blocks used in the docs (the
/// highlighter only ships a Dart grammar).
final Map<String, String> docGrammars = () {
  String g(String name, List<Map<String, String>> patterns) =>
      jsonEncode({'name': name, 'scopeName': 'source.$name', 'patterns': patterns});

  final text = g('text', []);
  final bash = g('bash', [
    {'match': r'(^|\s)#.*$', 'name': 'comment.line.number-sign.shell'},
    {'match': r'"(?:[^"\\]|\\.)*"', 'name': 'string.quoted.double.shell'},
    {'match': r"'[^']*'", 'name': 'string.quoted.single.shell'},
    {'match': r'\$\{[^}]*\}|\$[A-Za-z_][A-Za-z0-9_]*', 'name': 'variable.other.shell'},
    {'match': r'(?<=^|\s)--?[A-Za-z][\w-]*', 'name': 'constant.other.option.shell'},
    {'match': r'^\s*(?:fpack|dart|flutter|export|cd|git|sudo|base64|xcrun|security|keytool|echo|wget|chmod)\b', 'name': 'keyword.control.shell'},
  ]);
  final yaml = g('yaml', [
    {'match': r'(^|\s)#.*$', 'name': 'comment.line.number-sign.yaml'},
    {'match': r'^\s*-?\s*[\w.$-]+(?=\s*:)', 'name': 'entity.name.tag.yaml'},
    {'match': r'"(?:[^"\\]|\\.)*"', 'name': 'string.quoted.double.yaml'},
    {'match': r"'[^']*'", 'name': 'string.quoted.single.yaml'},
    {'match': r'\$\{[^}]*\}', 'name': 'variable.other.yaml'},
    {'match': r'\b(?:true|false|null)\b|\b\d+(?:\.\d+)?\b', 'name': 'constant.language.yaml'},
  ]);
  return {
    'text': text, 'plain': text, 'txt': text, 'console': bash,
    'bash': bash, 'sh': bash, 'shell': bash, 'zsh': bash, 'powershell': bash,
    'yaml': yaml, 'yml': yaml,
  };
}();
