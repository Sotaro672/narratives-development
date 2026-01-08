// frontend\mall\lib\main.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'core/ui/theme/app_theme.dart';
import 'app/bootstrap/bootstrap.dart';

Future<void> main() async {
  final BootstrapResult boot = await bootstrapApp();

  runApp(MyApp(router: boot.router));
}

class MyApp extends StatelessWidget {
  const MyApp({super.key, required this.router});

  final GoRouter router;

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'mall',
      theme: AppTheme.light(),
      routerConfig: router,
    );
  }
}
