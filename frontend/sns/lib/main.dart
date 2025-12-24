import 'package:flutter/material.dart';

import 'features/home/presentation/page/home_page.dart';
import 'core/ui/theme/app_theme.dart';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'sns',
      theme: AppTheme.light(),
      home: const HomePage(),
    );
  }
}
