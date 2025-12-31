// frontend/sns/lib/app/routing/router.dart
import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../shell/presentation/layout/app_shell.dart';

// ✅ signed-in footer
import '../shell/presentation/components/footer.dart';

// pages
import '../../features/list/presentation/page/list.dart';
import '../../features/list/presentation/page/catalog.dart';

// ✅ Profile page
import '../../features/avatar/presentation/page/avatar.dart';

// ✅ Profile (edit)
import '../../features/avatar/presentation/page/avatar_edit.dart';

// ✅ User edit page
import '../../features/user/presentation/page/user_edit.dart';

// ✅ Cart page
import '../../features/cart/presentation/page/cart.dart';

// ✅ SnsListItem 型
import '../../features/list/infrastructure/list_repository_http.dart';

// auth pages
import '../../features/auth/presentation/page/login_page.dart';
import '../../features/auth/presentation/page/create_account.dart';
import '../../features/auth/presentation/page/shipping_address.dart';
import '../../features/auth/presentation/page/billing_address.dart';
import '../../features/auth/presentation/page/avatar_create.dart';

GoRouter buildRouter({required bool firebaseReady, Object? initError}) {
  if (firebaseReady) return buildAppRouter();
  return buildPublicOnlyRouter(
    initError: initError ?? Exception('Firebase init failed'),
  );
}

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
      footer: const SignedInFooter(),
      child: Center(child: Text(state.error?.toString() ?? 'Not Found')),
    ),
  );
}

GoRouter buildPublicOnlyRouter({required Object initError}) {
  return GoRouter(
    initialLocation: '/',
    routes: [
      GoRoute(
        path: '/login',
        name: 'login',
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: '/create-account',
        name: 'createAccount',
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: '/shipping-address',
        name: 'shippingAddress',
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: '/billing-address',
        name: 'billingAddress',
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: '/avatar-create',
        name: 'avatarCreate',
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: '/avatar-edit',
        name: 'avatarEdit',
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),

      // ✅ NEW: /avatar も init error ページへ（firebase 未初期化時の 404 回避）
      GoRoute(
        path: '/avatar',
        name: 'avatar',
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),

      // ✅ NEW: /cart も init error ページへ（firebase 未初期化時の 404 回避）
      GoRoute(
        path: '/cart',
        name: 'cart',
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),

      GoRoute(
        path: '/user-edit',
        name: 'userEdit',
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
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

    // ✅ Profile edit は単独ルートのまま（AvatarEditPage 自前 header を使っているため）
    GoRoute(
      path: '/avatar-edit',
      name: 'avatarEdit',
      pageBuilder: (context, state) {
        final from = state.uri.queryParameters['from'];
        return NoTransitionPage(child: AvatarEditPage(from: from));
      },
    ),

    // ✅ User edit（from を渡す）
    GoRoute(
      path: '/user-edit',
      name: 'userEdit',
      pageBuilder: (context, state) {
        final from = state.uri.queryParameters['from'];
        return NoTransitionPage(child: UserEditPage(from: from));
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

        // ✅ 重要: /avatar を ShellRoute 内へ移動 -> AppShell(header/footer) が必ず出る
        GoRoute(
          path: '/avatar',
          name: 'avatar',
          pageBuilder: (context, state) {
            final from = state.uri.queryParameters['from'];
            return NoTransitionPage(child: AvatarPage(from: from));
          },
        ),

        // ✅ NEW: /cart（ShellRoute 内なので header/footer が出る）
        GoRoute(
          path: '/cart',
          name: 'cart',
          pageBuilder: (context, state) {
            final qp = state.uri.queryParameters;
            final avatarId = (qp['avatarId'] ?? '').trim();
            final from = qp['from'];
            return NoTransitionPage(
              child: CartPage(avatarId: avatarId, from: from),
            );
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
  if (loc == '/avatar') return 'Profile';
  if (loc == '/cart') return 'Cart';
  if (loc == '/avatar-edit') return 'Edit Avatar';
  if (loc == '/user-edit') return 'Account';
  return null;
}

List<Widget> _headerActionsFor(
  GoRouterState state, {
  required bool allowLogin,
  required bool firebaseReady,
}) {
  final path = state.uri.path;
  final isHome = path == '/'; // ✅ list.dart (= HomePage)
  final isAvatar = path == '/avatar';

  // これらは常にヘッダー右側アクション無し
  if (path == '/login' ||
      path == '/create-account' ||
      path == '/shipping-address' ||
      path == '/billing-address' ||
      path == '/avatar-create' ||
      path == '/avatar-edit' ||
      path == '/user-edit') {
    return const [];
  }

  if (!allowLogin) return const [];

  final from = state.uri.toString();
  final avatarId = (state.uri.queryParameters['avatarId'] ?? '').trim();

  // firebase 未初期化時は Sign in（+ Home のときだけ Cart）
  if (!firebaseReady) {
    final loginUri = Uri(path: '/login', queryParameters: {'from': from});
    if (isHome) {
      return [
        _HeaderCartButton(from: from, avatarId: avatarId),
        _HeaderSignInButton(to: loginUri.toString()),
      ];
    }
    return [_HeaderSignInButton(to: loginUri.toString())];
  }

  final isLoggedIn = FirebaseAuth.instance.currentUser != null;

  // 未ログイン時は Sign in（+ Home のときだけ Cart）
  if (!isLoggedIn) {
    final loginUri = Uri(path: '/login', queryParameters: {'from': from});
    if (isHome) {
      return [
        _HeaderCartButton(from: from, avatarId: avatarId),
        _HeaderSignInButton(to: loginUri.toString()),
      ];
    }
    return [_HeaderSignInButton(to: loginUri.toString())];
  }

  // ✅ ログイン済み:
  // - /avatar はハンバーガーのみ
  if (isAvatar) {
    return [_HeaderHamburgerMenuButton(from: from)];
  }

  // - list.dart（/）はカートのみ
  if (isHome) {
    return [_HeaderCartButton(from: from, avatarId: avatarId)];
  }

  return const [];
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

/// ✅ list.dart（/）用のカートボタン（header の右側に出す）
class _HeaderCartButton extends StatelessWidget {
  const _HeaderCartButton({required this.from, required this.avatarId});
  final String from;
  final String avatarId;

  @override
  Widget build(BuildContext context) {
    return IconButton(
      tooltip: 'Cart',
      icon: const Icon(Icons.shopping_cart_outlined),
      onPressed: () {
        final qp = <String, String>{'from': from};
        if (avatarId.trim().isNotEmpty) qp['avatarId'] = avatarId.trim();
        final uri = Uri(path: '/cart', queryParameters: qp);
        context.go(uri.toString());
      },
    );
  }
}

class _HeaderHamburgerMenuButton extends StatelessWidget {
  const _HeaderHamburgerMenuButton({required this.from});
  final String from;

  @override
  Widget build(BuildContext context) {
    return IconButton(
      tooltip: 'Menu',
      icon: const Icon(Icons.menu),
      onPressed: () async {
        await showModalBottomSheet<void>(
          context: context,
          isScrollControlled: true,
          useSafeArea: true,
          backgroundColor: Theme.of(context).colorScheme.surface,
          shape: const RoundedRectangleBorder(
            borderRadius: BorderRadius.vertical(top: Radius.circular(18)),
          ),
          builder: (_) => _AccountMenuSheet(from: from),
        );
      },
    );
  }
}

class _AccountMenuSheet extends StatelessWidget {
  const _AccountMenuSheet({required this.from});
  final String from;

  void _go(BuildContext context, String path, {Map<String, String>? qp}) {
    Navigator.pop(context);
    final uri = Uri(path: path, queryParameters: qp);
    context.go(uri.toString());
  }

  Widget _divider(BuildContext context) {
    return Divider(
      height: 1,
      thickness: 1,
      color: Theme.of(context).dividerColor.withValues(alpha: 0.35),
    );
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 16),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 44,
            height: 4,
            margin: const EdgeInsets.only(bottom: 10),
            decoration: BoxDecoration(
              color: scheme.outlineVariant.withValues(alpha: 0.7),
              borderRadius: BorderRadius.circular(999),
            ),
          ),
          Align(
            alignment: Alignment.centerLeft,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 6),
              child: Text(
                'Menu',
                style: Theme.of(context).textTheme.titleMedium,
              ),
            ),
          ),
          const SizedBox(height: 6),
          Expanded(
            child: ListView(
              children: [
                ListTile(
                  leading: const Icon(Icons.account_circle_outlined),
                  title: const Text('アバター情報'),
                  subtitle: const Text('プロフィール編集'),
                  onTap: () => _go(context, '/avatar-edit', qp: {'from': from}),
                ),
                _divider(context),
                ListTile(
                  leading: const Icon(Icons.manage_accounts_outlined),
                  title: const Text('ユーザー情報'),
                  subtitle: const Text('ユーザー編集'),
                  onTap: () => _go(context, '/user-edit', qp: {'from': from}),
                ),
                _divider(context),
                ListTile(
                  leading: const Icon(Icons.local_shipping_outlined),
                  title: const Text('配送先住所'),
                  subtitle: const Text('Shipping address'),
                  onTap: () => _go(
                    context,
                    '/shipping-address',
                    qp: {'from': from, 'intent': 'settings', 'mode': 'edit'},
                  ),
                ),
                _divider(context),
                ListTile(
                  leading: const Icon(Icons.receipt_long_outlined),
                  title: const Text('支払情報'),
                  subtitle: const Text('Billing address'),
                  onTap: () =>
                      _go(context, '/billing-address', qp: {'from': from}),
                ),
                _divider(context),
                ListTile(
                  leading: const Icon(Icons.email_outlined),
                  title: const Text('メールアドレス'),
                  subtitle: const Text('Email'),
                  onTap: () => _go(
                    context,
                    '/user-edit',
                    qp: {'from': from, 'tab': 'email'},
                  ),
                ),
                _divider(context),
                ListTile(
                  leading: const Icon(Icons.lock_outline),
                  title: const Text('パスワード'),
                  subtitle: const Text('Password'),
                  onTap: () => _go(
                    context,
                    '/user-edit',
                    qp: {'from': from, 'tab': 'password'},
                  ),
                ),
                _divider(context),
                ListTile(
                  leading: const Icon(Icons.logout),
                  title: const Text('サインアウト'),
                  onTap: () async {
                    Navigator.pop(context);
                    try {
                      await FirebaseAuth.instance.signOut();
                      if (context.mounted) context.go('/');
                    } catch (_) {}
                  },
                ),
              ],
            ),
          ),
          const SizedBox(height: 8),
        ],
      ),
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
