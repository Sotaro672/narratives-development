// frontend/sns/lib/main.dart
import 'package:flutter/material.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:go_router/go_router.dart';

import 'firebase_options.dart';
import 'core/ui/theme/app_theme.dart';
import 'app/routing/router.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  Object? initError;

  try {
    // ✅ 全プラットフォーム共通で firebase_options.dart を使用
    await Firebase.initializeApp(
      options: DefaultFirebaseOptions.currentPlatform,
    );
  } catch (e) {
    initError = e;
  }

  // ✅ Firebase が初期化できた場合のみ Auth 連動 router を使う
  final GoRouter router = (initError == null)
      ? buildAppRouter()
      : buildPublicOnlyRouter(initError: initError);

  runApp(MyApp(router: router));
}

class MyApp extends StatelessWidget {
  const MyApp({super.key, required this.router});

  final GoRouter router;

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'sns',
      theme: AppTheme.light(),
      routerConfig: router,
    );
  }
}
