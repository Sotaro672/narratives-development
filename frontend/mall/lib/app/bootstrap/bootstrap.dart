// frontend/sns/lib/app/bootstrap/bootstrap.dart
import 'package:flutter/widgets.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:go_router/go_router.dart';

import '../../firebase_options.dart';
import '../routing/router.dart';

class BootstrapResult {
  const BootstrapResult({
    required this.router,
    required this.firebaseReady,
    this.initError,
  });

  final GoRouter router;
  final bool firebaseReady;
  final Object? initError;
}

/// アプリ起動の初期化（Firebase init + router 決定）
Future<BootstrapResult> bootstrapApp({void Function(String s)? logger}) async {
  WidgetsFlutterBinding.ensureInitialized();

  Object? initError;
  bool firebaseReady = false;

  try {
    await Firebase.initializeApp(
      options: DefaultFirebaseOptions.currentPlatform,
    );
    firebaseReady = true;
  } catch (e) {
    initError = e;
    firebaseReady = false;
  }

  logger?.call(
    '[bootstrap] firebaseReady=$firebaseReady initError=${initError == null ? "-" : initError.toString()}',
  );

  final router = buildRouter(
    firebaseReady: firebaseReady,
    initError: initError,
  );

  return BootstrapResult(
    router: router,
    firebaseReady: firebaseReady,
    initError: initError,
  );
}
