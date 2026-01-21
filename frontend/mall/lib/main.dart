// frontend\mall\lib\main.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

// ✅ Web の URL を "#/" ではなく "/" にする
import 'package:flutter_web_plugins/url_strategy.dart';

import 'core/ui/theme/app_theme.dart';
import 'app/bootstrap/bootstrap.dart';

Future<void> main() async {
  // ✅ Flutter / plugin 初期化（Web plugin を使う前に必須）
  WidgetsFlutterBinding.ensureInitialized();

  // ✅ Hash -> Path (Webのみ有効。モバイルには影響しません)
  usePathUrlStrategy();

  try {
    final BootstrapResult boot = await bootstrapApp();
    runApp(MyApp(router: boot.router));
  } catch (e, st) {
    // bootstrap 失敗時にも画面が真っ白にならないようにする（原因表示）
    runApp(_BootstrapErrorApp(error: e, stackTrace: st));
  }
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

class _BootstrapErrorApp extends StatelessWidget {
  const _BootstrapErrorApp({required this.error, required this.stackTrace});

  final Object error;
  final StackTrace stackTrace;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'mall (bootstrap error)',
      theme: AppTheme.light(),
      home: Scaffold(
        body: SafeArea(
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 720),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: SingleChildScrollView(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      const Text(
                        'Bootstrap failed',
                        style: TextStyle(
                          fontSize: 20,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      const SizedBox(height: 12),
                      Text(error.toString()),
                      const SizedBox(height: 12),
                      Text(
                        stackTrace.toString(),
                        style: const TextStyle(fontSize: 12),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
