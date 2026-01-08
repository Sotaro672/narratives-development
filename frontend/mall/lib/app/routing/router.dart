// frontend\mall\lib\app\routing\router.dart
import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../shell/presentation/layout/app_shell.dart';
import '../shell/presentation/components/footer.dart';

// ✅ route defs
import 'routes.dart';

// ✅ redirect / AvatarIdStore
import 'navigation.dart';

// pages
import '../../features/list/presentation/page/list.dart';
import '../../features/list/presentation/page/catalog.dart';
import '../../features/avatar/presentation/page/avatar.dart';
import '../../features/avatar/presentation/page/avatar_edit.dart';
import '../../features/user/presentation/page/user_edit.dart';
import '../../features/cart/presentation/page/cart.dart';

// ✅ NEW: preview page
import '../../features/preview/presentation/preview.dart';

// ✅ NEW: payment page
import '../../features/payment/presentation/page/payment.dart';

// ✅ MallListItem 型
import '../../features/list/infrastructure/list_repository_http.dart';

// auth pages（✅ alias で確実に解決させる）
import '../../features/auth/presentation/page/login_page.dart' as auth_login;
import '../../features/auth/presentation/page/create_account.dart'
    as auth_create;
import '../../features/auth/presentation/page/shipping_address.dart'
    as auth_ship;
import '../../features/auth/presentation/page/billing_address.dart'
    as auth_bill;
import '../../features/auth/presentation/page/avatar_create.dart'
    as auth_avatar;

/// ------------------------------------------------------------
/// ✅ `from` は URL で壊れやすい（Hash + `?` `&` 混在）ので base64url で安全に運ぶ
String _encodeFrom(String raw) {
  final s = raw.trim();
  if (s.isEmpty) return '';
  return base64UrlEncode(utf8.encode(s));
}

String _decodeFrom(String? v) {
  final s = (v ?? '').trim();
  if (s.isEmpty) return '';
  // 既存の「生 from」も混在するので、失敗したらそのまま返す
  try {
    return utf8.decode(base64Url.decode(s));
  } catch (_) {
    return s;
  }
}

GoRouter buildRouter({required bool firebaseReady, Object? initError}) {
  if (firebaseReady) return buildAppRouter();
  return buildPublicOnlyRouter(
    initError: initError ?? Exception('Firebase init failed'),
  );
}

GoRouter buildAppRouter() {
  // ✅ Web の “初期復元” も拾いやすい
  final authRefresh = GoRouterRefreshStream(
    FirebaseAuth.instance.userChanges(),
  );

  return GoRouter(
    initialLocation: AppRoutePath.home,
    refreshListenable: Listenable.merge([authRefresh, AvatarIdStore.I]),

    // ✅ async redirect
    redirect: (context, state) async => appRedirect(context, state),

    debugLogDiagnostics: true,
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
    debugLogDiagnostics: true,
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
        path: AppRoutePath.preview,
        name: AppRouteName.preview,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.payment,
        name: AppRouteName.payment,
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
              final initialItem = extra is MallListItem ? extra : null;
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
        final from = _decodeFrom(state.uri.queryParameters[AppQueryKey.from]);
        final intent = state.uri.queryParameters[AppQueryKey.intent];
        return NoTransitionPage(
          child: auth_login.LoginPage(
            from: from.isEmpty ? null : from,
            intent: intent,
          ),
        );
      },
    ),
    GoRoute(
      path: AppRoutePath.createAccount,
      name: AppRouteName.createAccount,
      pageBuilder: (context, state) {
        final from = _decodeFrom(state.uri.queryParameters[AppQueryKey.from]);
        final intent = state.uri.queryParameters[AppQueryKey.intent];
        return NoTransitionPage(
          child: auth_create.CreateAccountPage(
            from: from.isEmpty ? null : from,
            intent: intent,
          ),
        );
      },
    ),
    GoRoute(
      path: AppRoutePath.shippingAddress,
      name: AppRouteName.shippingAddress,
      pageBuilder: (context, state) {
        final qp = state.uri.queryParameters;
        return NoTransitionPage(
          child: auth_ship.ShippingAddressPage(
            mode: qp[AppQueryKey.mode],
            oobCode: qp[AppQueryKey.oobCode],
            continueUrl: qp[AppQueryKey.continueUrl],
            lang: qp[AppQueryKey.lang],
            from: _decodeFrom(qp[AppQueryKey.from]),
            intent: qp[AppQueryKey.intent],
          ),
        );
      },
    ),
    GoRoute(
      path: AppRoutePath.billingAddress,
      name: AppRouteName.billingAddress,
      pageBuilder: (context, state) {
        final from = _decodeFrom(state.uri.queryParameters[AppQueryKey.from]);
        return NoTransitionPage(
          child: auth_bill.BillingAddressPage(from: from.isEmpty ? null : from),
        );
      },
    ),
    GoRoute(
      path: AppRoutePath.avatarCreate,
      name: AppRouteName.avatarCreate,
      pageBuilder: (context, state) {
        final from = _decodeFrom(state.uri.queryParameters[AppQueryKey.from]);
        return NoTransitionPage(
          child: auth_avatar.AvatarCreatePage(from: from.isEmpty ? null : from),
        );
      },
    ),
    GoRoute(
      path: AppRoutePath.avatarEdit,
      name: AppRouteName.avatarEdit,
      pageBuilder: (context, state) {
        final from = _decodeFrom(state.uri.queryParameters[AppQueryKey.from]);
        return NoTransitionPage(
          child: AvatarEditPage(from: from.isEmpty ? null : from),
        );
      },
    ),
    GoRoute(
      path: AppRoutePath.userEdit,
      name: AppRouteName.userEdit,
      pageBuilder: (context, state) {
        final from = _decodeFrom(state.uri.queryParameters[AppQueryKey.from]);
        return NoTransitionPage(
          child: UserEditPage(from: from.isEmpty ? null : from),
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
            final initialItem = extra is MallListItem ? extra : null;
            return CatalogPage(listId: listId, initialItem: initialItem);
          },
        ),
        GoRoute(
          path: AppRoutePath.avatar,
          name: AppRouteName.avatar,
          pageBuilder: (context, state) {
            final from = _decodeFrom(
              state.uri.queryParameters[AppQueryKey.from],
            );
            return NoTransitionPage(
              child: AvatarPage(from: from.isEmpty ? null : from),
            );
          },
        ),

        // ✅ CartPage は URL から読む前提（引数注入しない）
        GoRoute(
          path: AppRoutePath.cart,
          name: AppRouteName.cart,
          pageBuilder: (context, state) {
            final qp = state.uri.queryParameters;
            final qpAvatarId = (qp[AppQueryKey.avatarId] ?? '').trim();
            final avatarId = qpAvatarId.isNotEmpty
                ? qpAvatarId
                : AvatarIdStore.I.avatarId;

            return NoTransitionPage(
              key: ValueKey('cart-$avatarId'),
              child: const CartPage(),
            );
          },
        ),

        GoRoute(
          path: AppRoutePath.preview,
          name: AppRouteName.preview,
          pageBuilder: (context, state) {
            final qp = state.uri.queryParameters;
            final qpAvatarId = (qp[AppQueryKey.avatarId] ?? '').trim();
            final avatarId = qpAvatarId.isNotEmpty
                ? qpAvatarId
                : AvatarIdStore.I.avatarId;
            final from = _decodeFrom(qp[AppQueryKey.from]);

            return NoTransitionPage(
              key: ValueKey('preview-$avatarId'),
              child: PreviewPage(
                avatarId: avatarId,
                from: from.isEmpty ? null : from,
              ),
            );
          },
        ),

        GoRoute(
          path: AppRoutePath.payment,
          name: AppRouteName.payment,
          pageBuilder: (context, state) {
            final qp = state.uri.queryParameters;
            final qpAvatarId = (qp[AppQueryKey.avatarId] ?? '').trim();
            final avatarId = qpAvatarId.isNotEmpty
                ? qpAvatarId
                : AvatarIdStore.I.avatarId;
            final from = _decodeFrom(qp[AppQueryKey.from]);

            return NoTransitionPage(
              key: ValueKey('payment-$avatarId'),
              child: PaymentPage(
                avatarId: avatarId,
                from: from.isEmpty ? null : from,
              ),
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

/// ✅ Headerに表示する「アバター名」
/// - displayName があればそれ
/// - 無ければ email の @ より前
/// - それも無ければ 'My Profile'
String _avatarNameForHeader() {
  final u = FirebaseAuth.instance.currentUser;
  if (u == null) return 'Profile';

  final dn = (u.displayName ?? '').trim();
  if (dn.isNotEmpty) return dn;

  final email = (u.email ?? '').trim();
  if (email.isNotEmpty) {
    final i = email.indexOf('@');
    if (i > 0) return email.substring(0, i);
    return email;
  }

  return 'My Profile';
}

String? _titleFor(GoRouterState state) {
  final loc = state.uri.path;
  if (loc == AppRoutePath.home) return null;
  if (loc.startsWith('/catalog/')) return 'Catalog';

  // ✅ AvatarPage表示中は 'Profile' ではなく avatarName を表示
  if (loc == AppRoutePath.avatar) return _avatarNameForHeader();

  if (loc == AppRoutePath.cart) return 'Cart';
  if (loc == AppRoutePath.preview) return 'Preview';
  if (loc == AppRoutePath.payment) return 'Payment';
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
      path == AppRoutePath.userEdit ||
      path == AppRoutePath.preview ||
      path == AppRoutePath.payment) {
    return const [];
  }

  if (!allowLogin) return const [];

  final from = state.uri.toString();
  final avatarId = _resolveAvatarIdForHeader(state);

  if (!firebaseReady) {
    final loginUri = Uri(
      path: AppRoutePath.login,
      queryParameters: {AppQueryKey.from: _encodeFrom(from)},
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
      queryParameters: {AppQueryKey.from: _encodeFrom(from)},
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

        final qp = <String, String>{AppQueryKey.from: _encodeFrom(from)};
        if (id.trim().isNotEmpty) {
          qp[AppQueryKey.avatarId] = id;
        }

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
                  leading: const Icon(Icons.manage_accounts_outlined),
                  title: const Text('ユーザー情報'),
                  subtitle: const Text('ユーザー編集'),
                  onTap: () => _go(
                    context,
                    AppRoutePath.userEdit,
                    qp: {AppQueryKey.from: _encodeFrom(from)},
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
                      AppQueryKey.from: _encodeFrom(from),
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
                    qp: {AppQueryKey.from: _encodeFrom(from)},
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
                    qp: {
                      AppQueryKey.from: _encodeFrom(from),
                      AppQueryKey.tab: 'email',
                    },
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
                    qp: {
                      AppQueryKey.from: _encodeFrom(from),
                      AppQueryKey.tab: 'password',
                    },
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
                      AvatarIdStore.I.clear();
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
