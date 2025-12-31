// frontend/sns/lib/app/routing/router.dart
import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../shell/presentation/layout/app_shell.dart';
import '../shell/presentation/components/footer.dart';

// ✅ route defs
import 'routes.dart';

// pages
import '../../features/list/presentation/page/list.dart';
import '../../features/list/presentation/page/catalog.dart';
import '../../features/avatar/presentation/page/avatar.dart';
import '../../features/avatar/presentation/page/avatar_edit.dart';
import '../../features/user/presentation/page/user_edit.dart';
import '../../features/cart/presentation/page/cart.dart';

// ✅ SnsListItem 型
import '../../features/list/infrastructure/list_repository_http.dart';

// auth pages
import '../../features/auth/presentation/page/login_page.dart';
import '../../features/auth/presentation/page/create_account.dart';
import '../../features/auth/presentation/page/shipping_address.dart';
import '../../features/auth/presentation/page/billing_address.dart';
import '../../features/auth/presentation/page/avatar_create.dart';

/// ------------------------------------------------------------
/// ✅ avatarId の “現在値” をアプリ側で保持（URLに無い時の補完に使う）
class AvatarIdStore extends ChangeNotifier {
  AvatarIdStore._();
  static final AvatarIdStore I = AvatarIdStore._();

  String _avatarId = '';

  String get avatarId => _avatarId;

  void set(String v) {
    final next = v.trim();
    if (next.isEmpty) return;
    if (next == _avatarId) return;
    _avatarId = next;
    notifyListeners();
  }
}

GoRouter buildRouter({required bool firebaseReady, Object? initError}) {
  if (firebaseReady) return buildAppRouter();
  return buildPublicOnlyRouter(
    initError: initError ?? Exception('Firebase init failed'),
  );
}

GoRouter buildAppRouter() {
  final authRefresh = GoRouterRefreshStream(
    FirebaseAuth.instance.authStateChanges(),
  );

  return GoRouter(
    initialLocation: AppRoutePath.home,
    refreshListenable: Listenable.merge([authRefresh, AvatarIdStore.I]),
    redirect: (context, state) {
      final isLoggedIn = FirebaseAuth.instance.currentUser != null;
      final path = state.uri.path;
      final isLoginRoute = path == AppRoutePath.login;

      // login -> from or /
      if (isLoggedIn && isLoginRoute) {
        final from = state.uri.queryParameters[AppQueryKey.from];
        if (from != null && from.trim().isNotEmpty) return from;
        return AppRoutePath.home;
      }

      // ✅ /cart は avatarId が無ければ store から補完して URL を正規化（Web直打ち対策）
      if (path == AppRoutePath.cart) {
        final qp = state.uri.queryParameters;
        final avatarId = (qp[AppQueryKey.avatarId] ?? '').trim();
        if (avatarId.isEmpty && AvatarIdStore.I.avatarId.isNotEmpty) {
          final fixed = Map<String, String>.from(qp);
          fixed[AppQueryKey.avatarId] = AvatarIdStore.I.avatarId;
          final uri = Uri(path: AppRoutePath.cart, queryParameters: fixed);
          return uri.toString();
        }
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
    initialLocation: AppRoutePath.home,
    routes: [
      GoRoute(
        path: AppRoutePath.login,
        name: AppRouteName.login,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.createAccount,
        name: AppRouteName.createAccount,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.shippingAddress,
        name: AppRouteName.shippingAddress,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.billingAddress,
        name: AppRouteName.billingAddress,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.avatarCreate,
        name: AppRouteName.avatarCreate,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.avatarEdit,
        name: AppRouteName.avatarEdit,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.avatar,
        name: AppRouteName.avatar,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.cart,
        name: AppRouteName.cart,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.userEdit,
        name: AppRouteName.userEdit,
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
            path: AppRoutePath.home,
            name: AppRouteName.home,
            pageBuilder: (context, state) =>
                const NoTransitionPage(child: HomePage()),
          ),
          GoRoute(
            path: AppRoutePath.catalog,
            name: AppRouteName.catalog,
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
      path: AppRoutePath.login,
      name: AppRouteName.login,
      pageBuilder: (context, state) {
        final from = state.uri.queryParameters[AppQueryKey.from];
        final intent = state.uri.queryParameters[AppQueryKey.intent];
        return NoTransitionPage(
          child: LoginPage(from: from, intent: intent),
        );
      },
    ),
    GoRoute(
      path: AppRoutePath.createAccount,
      name: AppRouteName.createAccount,
      pageBuilder: (context, state) {
        final from = state.uri.queryParameters[AppQueryKey.from];
        final intent = state.uri.queryParameters[AppQueryKey.intent];
        return NoTransitionPage(
          child: CreateAccountPage(from: from, intent: intent),
        );
      },
    ),
    GoRoute(
      path: AppRoutePath.shippingAddress,
      name: AppRouteName.shippingAddress,
      pageBuilder: (context, state) {
        final qp = state.uri.queryParameters;
        return NoTransitionPage(
          child: ShippingAddressPage(
            mode: qp[AppQueryKey.mode],
            oobCode: qp[AppQueryKey.oobCode],
            continueUrl: qp[AppQueryKey.continueUrl],
            lang: qp[AppQueryKey.lang],
            from: qp[AppQueryKey.from],
            intent: qp[AppQueryKey.intent],
          ),
        );
      },
    ),
    GoRoute(
      path: AppRoutePath.billingAddress,
      name: AppRouteName.billingAddress,
      pageBuilder: (context, state) {
        final from = state.uri.queryParameters[AppQueryKey.from];
        return NoTransitionPage(child: BillingAddressPage(from: from));
      },
    ),
    GoRoute(
      path: AppRoutePath.avatarCreate,
      name: AppRouteName.avatarCreate,
      pageBuilder: (context, state) {
        final from = state.uri.queryParameters[AppQueryKey.from];

        final qpId = (state.uri.queryParameters[AppQueryKey.avatarId] ?? '')
            .trim();
        final exId = (state.extra is String)
            ? (state.extra as String).trim()
            : '';
        final id = qpId.isNotEmpty ? qpId : exId;
        if (id.isNotEmpty) AvatarIdStore.I.set(id);

        return NoTransitionPage(child: AvatarCreatePage(from: from));
      },
    ),
    GoRoute(
      path: AppRoutePath.avatarEdit,
      name: AppRouteName.avatarEdit,
      pageBuilder: (context, state) {
        final from = state.uri.queryParameters[AppQueryKey.from];

        final qpId = (state.uri.queryParameters[AppQueryKey.avatarId] ?? '')
            .trim();
        final exId = (state.extra is String)
            ? (state.extra as String).trim()
            : '';
        final id = qpId.isNotEmpty ? qpId : exId;
        if (id.isNotEmpty) AvatarIdStore.I.set(id);

        return NoTransitionPage(child: AvatarEditPage(from: from));
      },
    ),
    GoRoute(
      path: AppRoutePath.userEdit,
      name: AppRouteName.userEdit,
      pageBuilder: (context, state) {
        final from = state.uri.queryParameters[AppQueryKey.from];
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
          path: AppRoutePath.home,
          name: AppRouteName.home,
          pageBuilder: (context, state) =>
              const NoTransitionPage(child: HomePage()),
        ),
        GoRoute(
          path: AppRoutePath.catalog,
          name: AppRouteName.catalog,
          builder: (context, state) {
            final listId = state.pathParameters['listId'] ?? '';
            final extra = state.extra;
            final initialItem = extra is SnsListItem ? extra : null;
            return CatalogPage(listId: listId, initialItem: initialItem);
          },
        ),
        GoRoute(
          path: AppRoutePath.avatar,
          name: AppRouteName.avatar,
          pageBuilder: (context, state) {
            final from = state.uri.queryParameters[AppQueryKey.from];

            final qpId = (state.uri.queryParameters[AppQueryKey.avatarId] ?? '')
                .trim();
            final exId = (state.extra is String)
                ? (state.extra as String).trim()
                : '';
            final id = qpId.isNotEmpty ? qpId : exId;
            if (id.isNotEmpty) AvatarIdStore.I.set(id);

            return NoTransitionPage(child: AvatarPage(from: from));
          },
        ),
        GoRoute(
          path: AppRoutePath.cart,
          name: AppRouteName.cart,
          pageBuilder: (context, state) {
            final qp = state.uri.queryParameters;

            final qpId = (qp[AppQueryKey.avatarId] ?? '').trim();
            final avatarId = qpId.isNotEmpty ? qpId : AvatarIdStore.I.avatarId;

            if (qpId.isNotEmpty) {
              AvatarIdStore.I.set(qpId);
            }

            final from = qp[AppQueryKey.from];

            return NoTransitionPage(
              key: ValueKey('cart-$avatarId'),
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
  if (path == AppRoutePath.home) return false;
  return true;
}

String? _titleFor(GoRouterState state) {
  final loc = state.uri.path;
  if (loc == AppRoutePath.home) return null;
  if (loc.startsWith('/catalog/')) return 'Catalog';
  if (loc == AppRoutePath.avatar) return 'Profile';
  if (loc == AppRoutePath.cart) return 'Cart';
  if (loc == AppRoutePath.avatarEdit) return 'Edit Avatar';
  if (loc == AppRoutePath.userEdit) return 'Account';
  return null;
}

String _resolveAvatarIdForHeader(GoRouterState state) {
  final qpId = (state.uri.queryParameters[AppQueryKey.avatarId] ?? '').trim();
  if (qpId.isNotEmpty) return qpId;
  return AvatarIdStore.I.avatarId;
}

List<Widget> _headerActionsFor(
  GoRouterState state, {
  required bool allowLogin,
  required bool firebaseReady,
}) {
  final path = state.uri.path;
  final isHome = path == AppRoutePath.home;
  final isAvatar = path == AppRoutePath.avatar;

  if (path == AppRoutePath.login ||
      path == AppRoutePath.createAccount ||
      path == AppRoutePath.shippingAddress ||
      path == AppRoutePath.billingAddress ||
      path == AppRoutePath.avatarCreate ||
      path == AppRoutePath.avatarEdit ||
      path == AppRoutePath.userEdit) {
    return const [];
  }

  if (!allowLogin) return const [];

  final from = state.uri.toString();
  final avatarId = _resolveAvatarIdForHeader(state);

  if (!firebaseReady) {
    final loginUri = Uri(
      path: AppRoutePath.login,
      queryParameters: {AppQueryKey.from: from},
    );
    if (isHome) {
      return [
        _HeaderCartButton(from: from, avatarId: avatarId),
        _HeaderSignInButton(to: loginUri.toString()),
      ];
    }
    return [_HeaderSignInButton(to: loginUri.toString())];
  }

  final isLoggedIn = FirebaseAuth.instance.currentUser != null;

  if (!isLoggedIn) {
    final loginUri = Uri(
      path: AppRoutePath.login,
      queryParameters: {AppQueryKey.from: from},
    );
    if (isHome) {
      return [
        _HeaderCartButton(from: from, avatarId: avatarId),
        _HeaderSignInButton(to: loginUri.toString()),
      ];
    }
    return [_HeaderSignInButton(to: loginUri.toString())];
  }

  if (isAvatar) {
    return [_HeaderHamburgerMenuButton(from: from)];
  }

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
        final id = avatarId.trim().isNotEmpty
            ? avatarId.trim()
            : AvatarIdStore.I.avatarId;

        final qp = <String, String>{AppQueryKey.from: from};
        if (id.trim().isNotEmpty) {
          qp[AppQueryKey.avatarId] = id;
        }

        // ✅ ここは “cart.dart へ遷移” に戻す（goNamedでOK）
        context.goNamed(AppRouteName.cart, queryParameters: qp);
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
          // ✅ 右上キャンセル（×）で閉じられる
          Row(
            children: [
              Expanded(
                child: Center(
                  child: Container(
                    width: 44,
                    height: 4,
                    margin: const EdgeInsets.only(bottom: 10, left: 44),
                    decoration: BoxDecoration(
                      color: scheme.outlineVariant.withValues(alpha: 0.7),
                      borderRadius: BorderRadius.circular(999),
                    ),
                  ),
                ),
              ),
              IconButton(
                tooltip: 'Close',
                icon: const Icon(Icons.close),
                onPressed: () => Navigator.pop(context),
              ),
            ],
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 4),
            child: Align(
              alignment: Alignment.centerLeft,
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
                  onTap: () => _go(
                    context,
                    AppRoutePath.avatarEdit,
                    qp: {AppQueryKey.from: from},
                  ),
                ),
                _divider(context),
                ListTile(
                  leading: const Icon(Icons.manage_accounts_outlined),
                  title: const Text('ユーザー情報'),
                  subtitle: const Text('ユーザー編集'),
                  onTap: () => _go(
                    context,
                    AppRoutePath.userEdit,
                    qp: {AppQueryKey.from: from},
                  ),
                ),
                _divider(context),
                ListTile(
                  leading: const Icon(Icons.local_shipping_outlined),
                  title: const Text('配送先住所'),
                  subtitle: const Text('Shipping address'),
                  onTap: () => _go(
                    context,
                    AppRoutePath.shippingAddress,
                    qp: {
                      AppQueryKey.from: from,
                      AppQueryKey.intent: 'settings',
                      AppQueryKey.mode: 'edit',
                    },
                  ),
                ),
                _divider(context),
                ListTile(
                  leading: const Icon(Icons.receipt_long_outlined),
                  title: const Text('支払情報'),
                  subtitle: const Text('Billing address'),
                  onTap: () => _go(
                    context,
                    AppRoutePath.billingAddress,
                    qp: {AppQueryKey.from: from},
                  ),
                ),
                _divider(context),
                ListTile(
                  leading: const Icon(Icons.email_outlined),
                  title: const Text('メールアドレス'),
                  subtitle: const Text('Email'),
                  onTap: () => _go(
                    context,
                    AppRoutePath.userEdit,
                    qp: {AppQueryKey.from: from, AppQueryKey.tab: 'email'},
                  ),
                ),
                _divider(context),
                ListTile(
                  leading: const Icon(Icons.lock_outline),
                  title: const Text('パスワード'),
                  subtitle: const Text('Password'),
                  onTap: () => _go(
                    context,
                    AppRoutePath.userEdit,
                    qp: {AppQueryKey.from: from, AppQueryKey.tab: 'password'},
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
                      if (context.mounted) context.go(AppRoutePath.home);
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
                    onPressed: () => context.go(AppRoutePath.home),
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
