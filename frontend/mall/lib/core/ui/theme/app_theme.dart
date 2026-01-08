// frontend\mall\lib\core\ui\theme\app_theme.dart
import 'package:flutter/material.dart';

class AppTheme {
  static ThemeData light() {
    final scheme = ColorScheme.fromSeed(
      seedColor: Colors.pink,
      brightness: Brightness.light,
    );

    return ThemeData(
      useMaterial3: true,

      // ✅ white-based theme
      brightness: Brightness.light,
      colorScheme: scheme,
      scaffoldBackgroundColor: Colors.white,

      // ✅ apply custom font (pubspec.yaml の family 名と一致させる)
      fontFamily: 'NotoSansJP',

      // ✅ clean AppBar
      appBarTheme: const AppBarTheme(
        backgroundColor: Colors.white,
        foregroundColor: Colors.black87,
        elevation: 0,
        scrolledUnderElevation: 0,
        surfaceTintColor: Colors.transparent,
      ),

      // ✅ Flutter SDK に合わせて *Data を使う
      cardTheme: const CardThemeData(
        color: Colors.white,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        margin: EdgeInsets.zero,
      ),
      dialogTheme: const DialogThemeData(
        backgroundColor: Colors.white,
        surfaceTintColor: Colors.transparent,
      ),

      // ✅ inputs
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: Colors.white,
        border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
      ),

      // ✅ avoid gray tint on M3 surfaces
      canvasColor: Colors.white,
    );
  }
}
