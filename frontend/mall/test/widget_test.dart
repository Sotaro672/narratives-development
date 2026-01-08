// frontend\mall\test\widget_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:mall/main.dart';

void main() {
  testWidgets('MyApp builds (smoke test)', (WidgetTester tester) async {
    // ✅ テスト用の最小 router
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (context, state) =>
              const Scaffold(body: Center(child: Text('OK'))),
        ),
      ],
    );

    await tester.pumpWidget(MyApp(router: router));
    await tester.pumpAndSettle();

    expect(find.text('OK'), findsOneWidget);
  });
}
