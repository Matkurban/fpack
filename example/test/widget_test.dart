import 'package:flutter_test/flutter_test.dart';
import 'package:fpack_example/main.dart';

void main() {
  testWidgets('shows the notes', (tester) async {
    await tester.pumpWidget(const AuroraApp());
    await tester.pumpAndSettle();
    expect(find.text('Aurora Notes'), findsOneWidget);
    expect(find.text('Welcome to Aurora Notes'), findsOneWidget);
  });
}
