// frontend/sns/lib/app/routing/router.dart
import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../shell/presentation/layout/app_shell.dart';

// ✅ NEW: signed-in footer
import '../shell/presentation/components/footer.dart';

// pages
import '../../features/home/presentation/page/home_page.dart';
import '../../features/home/presentation/page/catalog.dart';

// ✅ SnsListItem 型
import '../../features/list/infrastructure/list_repository_http.dart';

// auth pages
import '../../features/auth/presentation/page/login_page.dart';
import '../../features/auth/presentation/page/create_account.dart';
import '../../features/auth/presentation/page/shipping_address.dart';
import '../../features/auth/presentation/page/billing_address.dart';
import '../../features/auth/presentation/page/avatar_create.dart';

/// ✅ router の入口を 1 本化（bootstrap はこれだけ呼べばOK）
///
/// - firebaseReady=true  -> Auth 連動 router
/// - firebaseReady=false -> 公開閲覧 router（initError を表示）
GoRouter buildRouter({required bool firebaseReady, Object? initError}) {
  if (firebaseReady) return buildAppRouter();
  return buildPublicOnlyRouter(
    initError: initError ?? Exception('Firebase init failed'),
  );
}

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
      // ✅ Signed-in のときだけ表示（内部で authStateChanges を見て hidden になる）
      footer: const SignedInFooter(),
      child: Center(child: Text(state.error?.toString() ?? 'Not Found')),
    ),
  );
}

/// ✅ Firebase 初期化に失敗した場合の “閲覧専用” ルーター
/// - catalog 閲覧は可能
/// - login/create-account/shipping-address/billing-address/avatar-create は「Firebase未設定」画面にする（落とさない）
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

      GoRoute(
        path: '/create-account',
        name: 'createAccount',
        pageBuilder: (context, state) {
          return NoTransitionPage(
            child: _FirebaseInitErrorPage(error: initError),
          );
        },
      ),

      GoRoute(
        path: '/shipping-address',
        name: 'shippingAddress',
        pageBuilder: (context, state) {
          return NoTransitionPage(
            child: _FirebaseInitErrorPage(error: initError),
          );
        },
      ),

      GoRoute(
        path: '/billing-address',
        name: 'billingAddress',
        pageBuilder: (context, state) {
          return NoTransitionPage(
            child: _FirebaseInitErrorPage(error: initError),
          );
        },
      ),

      GoRoute(
        path: '/avatar-create',
        name: 'avatarCreate',
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
            actions: _headerActionsFor(
              state,
              allowLogin: true,
              firebaseReady: false,
            ),
            // ✅ Firebase 未設定時は footer なし（認証前提UIを出さない）
            footer: null,
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
      footer: null,
      child: Center(child: Text(state.error?.toString() ?? 'Not Found')),
    ),
  );
}

List<RouteBase> _routes({required bool firebaseReady}) {
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

    GoRoute(
      path: '/create-account',
      name: 'createAccount',
      pageBuilder: (context, state) {
        final from = state.uri.queryParameters['from'];
        final intent = state.uri.queryParameters['intent'];
        return NoTransitionPage(
          child: CreateAccountPage(from: from, intent: intent),
        );
      },
    ),

    GoRoute(
      path: '/shipping-address',
      name: 'shippingAddress',
      pageBuilder: (context, state) {
        final qp = state.uri.queryParameters;
        return NoTransitionPage(
          child: ShippingAddressPage(
            mode: qp['mode'],
            oobCode: qp['oobCode'],
            continueUrl: qp['continueUrl'],
            lang: qp['lang'],
            from: qp['from'],
            intent: qp['intent'],
          ),
        );
      },
    ),

    GoRoute(
      path: '/billing-address',
      name: 'billingAddress',
      pageBuilder: (context, state) {
        final from = state.uri.queryParameters['from'];
        return NoTransitionPage(child: BillingAddressPage(from: from));
      },
    ),

    GoRoute(
      path: '/avatar-create',
      name: 'avatarCreate',
      pageBuilder: (context, state) {
        final from = state.uri.queryParameters['from'];
        return NoTransitionPage(child: AvatarCreatePage(from: from));
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
          // ✅ signed-in のときだけ表示（中で auth 判定して hidden になる）
          footer: const SignedInFooter(),
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
  if (path == '/') return false;
  return true;
}

String? _titleFor(GoRouterState state) {
  final loc = state.uri.path;
  if (loc == '/') return null;
  if (loc.startsWith('/catalog/')) return 'Catalog';
  return null;
}

List<Widget> _headerActionsFor(
  GoRouterState state, {
  required bool allowLogin,
  required bool firebaseReady,
}) {
  final path = state.uri.path;

  if (path == '/login' ||
      path == '/create-account' ||
      path == '/shipping-address' ||
      path == '/billing-address' ||
      path == '/avatar-create') {
    return const [];
  }

  if (!allowLogin) return const [];

  final from = state.uri.toString();

  if (!firebaseReady) {
    final loginUri = Uri(path: '/login', queryParameters: {'from': from});
    return [_HeaderSignInButton(to: loginUri.toString())];
  }

  final isLoggedIn = FirebaseAuth.instance.currentUser != null;

  if (!isLoggedIn) {
    final loginUri = Uri(path: '/login', queryParameters: {'from': from});
    return [_HeaderSignInButton(to: loginUri.toString())];
  }

  return const [_HeaderSignedInButton()];
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
