// frontend/sns/lib/app/routing/router.dart
import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../shell/presentation/layout/app_shell.dart';

// pages
import '../../features/home/presentation/page/home_page.dart';
import '../../features/home/presentation/page/catalog.dart';

// ✅ SnsListItem 型
import '../../features/list/infrastructure/list_repository_http.dart';

// auth page
import '../../features/auth/presentation/page/login_page.dart';

/// ✅ Firebase OK 前提のルーター（Auth 連動）
GoRouter buildAppRouter() {
  return GoRouter(
    initialLocation: '/',

    refreshListenable: GoRouterRefreshStream(
      FirebaseAuth.instance.authStateChanges(),
    ),

    redirect: (context, state) {
      final isLoggedIn = FirebaseAuth.instance.currentUser != null;
      final path = state.uri.path;
      final isLoginRoute = path == '/login';

      // ログイン済みで /login に来たら from に戻す
      if (isLoggedIn && isLoginRoute) {
        final from = state.uri.queryParameters['from'];
        if (from != null && from.trim().isNotEmpty) return from;
        return '/';
      }
      return null;
    },

    routes: _routes(),
    errorBuilder: (context, state) => AppShell(
      title: 'Not Found',
      showBack: true,
      child: Center(child: Text(state.error?.toString() ?? 'Not Found')),
    ),
  );
}

/// ✅ Firebase 初期化に失敗した場合の “閲覧専用” ルーター
/// - catalog 閲覧は可能
/// - login は「Firebase未設定」画面にする（落とさない）
GoRouter buildPublicOnlyRouter({required Object initError}) {
  return GoRouter(
    initialLocation: '/',
    routes: [
      GoRoute(
        path: '/login',
        name: 'login',
        pageBuilder: (context, state) {
          return NoTransitionPage(
            child: _FirebaseInitErrorPage(error: initError),
          );
        },
      ),
      ShellRoute(
        builder: (context, state, child) {
          return AppShell(
            title: _titleFor(state),
            showBack: true,
            child: child,
          );
        },
        routes: [
          GoRoute(
            path: '/',
            name: HomePage.pageName,
            pageBuilder: (context, state) =>
                const NoTransitionPage(child: HomePage()),
          ),
          GoRoute(
            path: '/catalog/:listId',
            name: 'catalog',
            builder: (context, state) {
              final listId = state.pathParameters['listId'] ?? '';
              final extra = state.extra;
              final initialItem = extra is SnsListItem ? extra : null;
              return CatalogPage(listId: listId, initialItem: initialItem);
            },
          ),
        ],
      ),
    ],
    errorBuilder: (context, state) => AppShell(
      title: 'Not Found',
      showBack: true,
      child: Center(child: Text(state.error?.toString() ?? 'Not Found')),
    ),
  );
}

List<RouteBase> _routes() {
  return [
    GoRoute(
      path: '/login',
      name: 'login',
      pageBuilder: (context, state) {
        final from = state.uri.queryParameters['from'];
        final intent = state.uri.queryParameters['intent'];
        return NoTransitionPage(
          child: LoginPage(from: from, intent: intent),
        );
      },
    ),
    ShellRoute(
      builder: (context, state, child) {
        return AppShell(title: _titleFor(state), showBack: true, child: child);
      },
      routes: [
        GoRoute(
          path: '/',
          name: HomePage.pageName,
          pageBuilder: (context, state) =>
              const NoTransitionPage(child: HomePage()),
        ),
        GoRoute(
          path: '/catalog/:listId',
          name: 'catalog',
          builder: (context, state) {
            final listId = state.pathParameters['listId'] ?? '';
            final extra = state.extra;
            final initialItem = extra is SnsListItem ? extra : null;
            return CatalogPage(listId: listId, initialItem: initialItem);
          },
        ),
      ],
    ),
  ];
}

String? _titleFor(GoRouterState state) {
  final loc = state.uri.path;
  if (loc == '/') return null;
  if (loc.startsWith('/catalog/')) return 'Catalog';
  return null;
}

class GoRouterRefreshStream extends ChangeNotifier {
  GoRouterRefreshStream(Stream<dynamic> stream) {
    _sub = stream.listen((_) => notifyListeners());
  }

  late final StreamSubscription<dynamic> _sub;

  @override
  void dispose() {
    _sub.cancel();
    super.dispose();
  }
}

class _FirebaseInitErrorPage extends StatelessWidget {
  const _FirebaseInitErrorPage({required this.error});

  final Object error;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 640),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.warning_amber_rounded, size: 48),
                  const SizedBox(height: 12),
                  Text(
                    'Firebase is not configured for Web.',
                    style: Theme.of(context).textTheme.titleLarge,
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 8),
                  Text(error.toString(), textAlign: TextAlign.center),
                  const SizedBox(height: 16),
                  const Text(
                    'Fix:\n'
                    '1) Run: flutterfire configure (recommended)\n'
                    'or\n'
                    '2) Provide Firebase web settings via --dart-define.',
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 16),
                  OutlinedButton(
                    onPressed: () => context.go('/'),
                    child: const Text('Back to Home'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
