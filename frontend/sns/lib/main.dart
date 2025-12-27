// frontend/sns/lib/main.dart
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:go_router/go_router.dart';

import 'core/ui/theme/app_theme.dart';
import 'app/routing/router.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  Object? initError;

  try {
    if (kIsWeb) {
      final opts = _firebaseOptionsFromEnv();
      if (opts == null) {
        throw StateError(
          'Firebase web options are missing. '
          'Pass them via --dart-define or generate firebase_options.dart via flutterfire configure.',
        );
      }
      await Firebase.initializeApp(options: opts);
    } else {
      await Firebase.initializeApp();
    }
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

/// Web 用 FirebaseOptions を --dart-define から読む
/// 例:
/// flutter run -d chrome \
///   --dart-define=FIREBASE_API_KEY=... \
///   --dart-define=FIREBASE_AUTH_DOMAIN=... \
///   --dart-define=FIREBASE_PROJECT_ID=... \
///   --dart-define=FIREBASE_STORAGE_BUCKET=... \
///   --dart-define=FIREBASE_MESSAGING_SENDER_ID=... \
///   --dart-define=FIREBASE_APP_ID=...
FirebaseOptions? _firebaseOptionsFromEnv() {
  const apiKey = String.fromEnvironment('FIREBASE_API_KEY');
  const authDomain = String.fromEnvironment('FIREBASE_AUTH_DOMAIN');
  const projectId = String.fromEnvironment('FIREBASE_PROJECT_ID');
  const storageBucket = String.fromEnvironment('FIREBASE_STORAGE_BUCKET');
  const messagingSenderId = String.fromEnvironment(
    'FIREBASE_MESSAGING_SENDER_ID',
  );
  const appId = String.fromEnvironment('FIREBASE_APP_ID');

  // measurementId は無くてもOK（Analytics未使用なら空でOK）
  const measurementId = String.fromEnvironment('FIREBASE_MEASUREMENT_ID');

  final required = [
    apiKey,
    authDomain,
    projectId,
    storageBucket,
    messagingSenderId,
    appId,
  ];
  if (required.any((v) => v.trim().isEmpty)) return null;

  return FirebaseOptions(
    apiKey: apiKey,
    authDomain: authDomain,
    projectId: projectId,
    storageBucket: storageBucket,
    messagingSenderId: messagingSenderId,
    appId: appId,
    measurementId: measurementId.trim().isEmpty ? null : measurementId,
  );
}
