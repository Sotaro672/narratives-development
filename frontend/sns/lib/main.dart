// frontend/sns/lib/main.dart
import 'package:flutter/material.dart';

import 'core/ui/theme/app_theme.dart';
import 'app/routing/router.dart';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'sns',
      theme: AppTheme.light(),

      // ✅ AppShell は router（ShellRoute）側で包むので、main.dart の builder は使わない
      routerConfig: appRouter,
    );
  }
}
