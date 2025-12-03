// frontend/inspector/lib/main.dart
import 'package:flutter/material.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_auth/firebase_auth.dart';

import 'firebase_options.dart';
import 'screens/login_screen.dart';
import 'screens/inspection_scan_screen.dart';
import 'screens/inspection_detail_screen.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  await Firebase.initializeApp(options: DefaultFirebaseOptions.currentPlatform);

  runApp(const InspectorApp());
}

class InspectorApp extends StatelessWidget {
  const InspectorApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Narratives Inspector',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.indigo),
        useMaterial3: true,
      ),

      // ★ ここを routes ではなく onGenerateRoute に変更
      onGenerateRoute: (settings) {
        if (settings.name == '/detail') {
          final productId = settings.arguments as String;
          return MaterialPageRoute(
            builder: (_) => InspectionDetailScreen(productId: productId),
          );
        }

        return MaterialPageRoute(builder: (_) => const _RootPage());
      },

      initialRoute: '/',
    );
  }
}

/// ログイン状態に応じて画面を出し分ける
class _RootPage extends StatelessWidget {
  const _RootPage();

  @override
  Widget build(BuildContext context) {
    return StreamBuilder<User?>(
      stream: FirebaseAuth.instance.authStateChanges(),
      builder: (context, snapshot) {
        // ローディング中
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
        }

        final user = snapshot.data;

        // 未ログイン → ログイン画面
        if (user == null) {
          return const LoginScreen();
        }

        // ログイン済み → 検品用 QR スキャン画面
        return const InspectionScanScreen();
      },
    );
  }
}
