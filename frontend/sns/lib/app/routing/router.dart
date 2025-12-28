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

    // ✅ Firebase OK のときだけ authStateChanges を使う
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

    routes: _routes(firebaseReady: true),

    errorBuilder: (context, state) => AppShell(
      title: 'Not Found',
      showBack: true,
      actions: _headerActionsFor(state, allowLogin: true, firebaseReady: true),
      child: Center(child: Text(state.error?.toString() ?? 'Not Found')),
    ),
  );
}

/// ✅ Firebase 初期化に失敗した場合の “閲覧専用” ルーター
/// - catalog 閲覧は可能
/// - login は「Firebase未設定」画面にする（落とさない）
///
/// ⚠️ 重要:
///   このルーターでは FirebaseAuth.instance に一切触れない（minified type error 回避）
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
            showBack: _showBackFor(state),
            // ✅ Firebase 未設定でも「Sign in」は出す（ただし Auth 判定はしない）
            actions: _headerActionsFor(
              state,
              allowLogin: true,
              firebaseReady: false,
            ),
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
      actions: _headerActionsFor(state, allowLogin: true, firebaseReady: false),
      child: Center(child: Text(state.error?.toString() ?? 'Not Found')),
    ),
  );
}

List<RouteBase> _routes({required bool firebaseReady}) {
  return [
    // ✅ login は ShellRoute の外（ヘッダー/フッター不要）
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
        return AppShell(
          title: _titleFor(state),
          showBack: _showBackFor(state),
          actions: _headerActionsFor(
            state,
            allowLogin: true,
            firebaseReady: firebaseReady,
          ),
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
  ];
}

bool _showBackFor(GoRouterState state) {
  final path = state.uri.path;

  // ✅ Homeは戻るボタンを出さない
  if (path == '/') return false;

  // ✅ それ以外は表示（例: /catalog/:listId）
  return true;
}

String? _titleFor(GoRouterState state) {
  final loc = state.uri.path;
  if (loc == '/') return null;
  if (loc.startsWith('/catalog/')) return 'Catalog';
  return null;
}

/// ✅ ヘッダー右側 actions（Sign in / Sign out）
///
/// firebaseReady=false の場合は FirebaseAuth を触らない（Webのminified type error 回避）
List<Widget> _headerActionsFor(
  GoRouterState state, {
  required bool allowLogin,
  required bool firebaseReady,
}) {
  final path = state.uri.path;

  // login 画面では何も出さない（ループ/見た目の二重化防止）
  if (path == '/login') return const [];

  if (!allowLogin) return const [];

  // from は “今いる場所に戻す” ために、クエリ含めた URI を使う
  final from = state.uri.toString();

  // ✅ Firebase が未初期化/未設定の時は「常に Sign in」だけを出す（Auth判定しない）
  if (!firebaseReady) {
    final loginUri = Uri(path: '/login', queryParameters: {'from': from});
    return [_HeaderSignInButton(to: loginUri.toString())];
  }

  // ✅ Firebase OK の時だけ Auth 判定
  final isLoggedIn = FirebaseAuth.instance.currentUser != null;

  if (!isLoggedIn) {
    final loginUri = Uri(path: '/login', queryParameters: {'from': from});
    return [_HeaderSignInButton(to: loginUri.toString())];
  }

  return const [_HeaderSignedInButton()];
}

/// Stream を listen して GoRouter を refresh するための Listenable
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

class _HeaderSignInButton extends StatelessWidget {
  const _HeaderSignInButton({required this.to});

  final String to;

  @override
  Widget build(BuildContext context) {
    return TextButton(
      onPressed: () => context.go(to),
      child: const Text('Sign in'),
    );
  }
}

class _HeaderSignedInButton extends StatelessWidget {
  const _HeaderSignedInButton();

  @override
  Widget build(BuildContext context) {
    return TextButton(
      onPressed: () async {
        try {
          await FirebaseAuth.instance.signOut();
          if (context.mounted) context.go('/');
        } catch (_) {
          // ignore
        }
      },
      child: const Text('Sign out'),
    );
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
