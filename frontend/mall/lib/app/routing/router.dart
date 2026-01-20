// frontend\mall\lib\app\routing\router.dart
import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../shell/presentation/layout/app_shell.dart';
import '../shell/presentation/components/footer.dart';

// ✅ route defs
import 'routes.dart';

// ✅ redirect / stores
import 'navigation.dart';

// pages
import '../../features/list/presentation/page/list.dart';
import '../../features/list/presentation/page/catalog.dart';
import '../../features/avatar/presentation/page/avatar.dart';
import '../../features/avatar/presentation/page/avatar_edit.dart';
import '../../features/user/presentation/page/user_edit.dart';
import '../../features/cart/presentation/page/cart.dart';
import '../../features/preview/presentation/page/preview.dart';
import '../../features/payment/presentation/page/payment.dart';
import '../../features/wallet/presentation/page/contents.dart';

// ✅ MallListItem 型
import '../../features/list/infrastructure/list_repository_http.dart';

// ✅ cart repository (for header badge)
import '../../features/cart/infrastructure/repository_http.dart'
    show CartRepositoryHttp, CartDTO, CartHttpException;

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
/// ✅ Pattern B: URL の `from` 制御を廃止し、narratives.jp 内で navigation state を保持する
/// - 戻り先は `navigation.dart` 側の Store（例: NavStore）に保存する
/// - このファイルでは `from` query の decode/encode/sanitize を一切行わない
///
/// NOTE:
/// - ここでは `NavStore.I.setReturnTo(String location)` / `consumeReturnTo()` のような
///   API が `navigation.dart` に存在する想定で呼び出しています。
/// - まだ未実装の場合は `navigation.dart` に追加してください。
void _captureReturnToForInternalNav(String location) {
  try {
    // navigation.dart 側で実装する（Listenable でも可）
    NavStore.I.setReturnTo(location);
  } catch (_) {
    // fail-open（遷移自体は止めない）
  }
}

/// ------------------------------------------------------------
/// ✅ /:productId が “固定パス” と衝突した時の安全弁（念のため）
bool _isReservedTopSegment(String seg) {
  const reserved = <String>{
    'login',
    'create-account',
    'shipping-address',
    'billing-address',
    'avatar-create',
    'avatar-edit',
    'avatar',
    'user-edit',
    'cart',
    'preview',
    'payment',
    'catalog',

    // ✅ wallet contents
    'wallet',
  };
  return reserved.contains(seg);
}

GoRouter buildRouter({required bool firebaseReady, Object? initError}) {
  if (firebaseReady) return buildAppRouter();
  return buildPublicOnlyRouter(
    initError: initError ?? Exception('Firebase init failed'),
  );
}

GoRouter buildAppRouter() {
  final authRefresh = GoRouterRefreshStream(
    FirebaseAuth.instance.userChanges(),
  );

  return GoRouter(
    initialLocation: AppRoutePath.home,
    // ✅ NavStore を Listenable にする場合は merge に追加可能
    refreshListenable: Listenable.merge([
      authRefresh,
      AvatarIdStore.I,
      // NavStore.I, // <- navigation.dart 側で Listenable を実装したら有効化
    ]),
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

      // ✅ wallet contents（public-only時はエラーページ）
      GoRoute(
        path: AppRoutePath.walletContents,
        name: AppRouteName.walletContents,
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
        final intent = state.uri.queryParameters[AppQueryKey.intent];

        // ✅ Pattern B: login 後の戻り先は NavStore が保持する
        // - このページへ遷移する直前に header/button/redirect で setReturnTo する
        return NoTransitionPage(child: auth_login.LoginPage(intent: intent));
      },
    ),
    GoRoute(
      path: AppRoutePath.createAccount,
      name: AppRouteName.createAccount,
      pageBuilder: (context, state) {
        final intent = state.uri.queryParameters[AppQueryKey.intent];
        return NoTransitionPage(
          child: auth_create.CreateAccountPage(intent: intent),
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
            intent: qp[AppQueryKey.intent],
          ),
        );
      },
    ),
    GoRoute(
      path: AppRoutePath.billingAddress,
      name: AppRouteName.billingAddress,
      pageBuilder: (context, state) {
        return const NoTransitionPage(child: auth_bill.BillingAddressPage());
      },
    ),
    GoRoute(
      path: AppRoutePath.avatarCreate,
      name: AppRouteName.avatarCreate,
      pageBuilder: (context, state) {
        return const NoTransitionPage(child: auth_avatar.AvatarCreatePage());
      },
    ),
    GoRoute(
      path: AppRoutePath.avatarEdit,
      name: AppRouteName.avatarEdit,
      pageBuilder: (context, state) {
        return const NoTransitionPage(child: AvatarEditPage());
      },
    ),
    GoRoute(
      path: AppRoutePath.userEdit,
      name: AppRouteName.userEdit,
      pageBuilder: (context, state) {
        // ✅ URL から tab だけは取る（UI タブ制御は URL でも問題ない）
        // - ただし「戻り先」は NavStore に寄せる
        return NoTransitionPage(
          child: UserEditPage(tab: state.uri.queryParameters[AppQueryKey.tab]),
        );
      },
    ),

    // ✅ ShellRoute 配下に置くことで header/footer が必ず出る
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
            return const NoTransitionPage(child: AvatarPage());
          },
        ),

        // ✅ wallet token contents（Shell内）
        GoRoute(
          path: AppRoutePath.walletContents,
          name: AppRouteName.walletContents,
          pageBuilder: (context, state) {
            final extra = state.extra;
            final mint = (extra is String ? extra : '').trim();
            if (mint.isEmpty) {
              return const NoTransitionPage(
                child: Center(child: Text('mintAddress is required.')),
              );
            }

            return NoTransitionPage(
              key: ValueKey('wallet-contents-$mint'),
              child: WalletContentsPage(mintAddress: mint),
            );
          },
        ),

        GoRoute(
          path: AppRoutePath.cart,
          name: AppRouteName.cart,
          pageBuilder: (context, state) {
            return const NoTransitionPage(
              key: ValueKey('cart'),
              child: CartPage(),
            );
          },
        ),

        GoRoute(
          path: AppRoutePath.preview,
          name: AppRouteName.preview,
          pageBuilder: (context, state) {
            final qp = state.uri.queryParameters;
            final productId = (qp[AppQueryKey.productId] ?? '').trim();

            // ✅ avatarId は URL から取らない（store から使う）
            final avatarId = AvatarIdStore.I.avatarId;

            return NoTransitionPage(
              key: ValueKey('preview-$productId'),
              child: PreviewPage(
                avatarId: avatarId,
                productId: productId.isEmpty ? null : productId,
              ),
            );
          },
        ),

        GoRoute(
          path: AppRoutePath.payment,
          name: AppRouteName.payment,
          pageBuilder: (context, state) {
            // ✅ avatarId は URL から取らない（store から使う）
            final avatarId = AvatarIdStore.I.avatarId;

            return NoTransitionPage(
              key: const ValueKey('payment'),
              child: PaymentPage(avatarId: avatarId),
            );
          },
        ),

        GoRoute(
          path: AppRoutePath.qrProduct,
          name: AppRouteName.qrProduct,
          pageBuilder: (context, state) {
            final productId = (state.pathParameters['productId'] ?? '').trim();

            if (productId.isEmpty || _isReservedTopSegment(productId)) {
              return const NoTransitionPage(child: HomePage());
            }

            // ✅ avatarId は URL から取らない（store から使う）
            final avatarId = AvatarIdStore.I.avatarId;

            // ✅ Pattern B: from を URL に詰めず、戻り先は Store に任せる
            return NoTransitionPage(
              key: ValueKey('qr-preview-$productId'),
              child: PreviewPage(avatarId: avatarId, productId: productId),
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

  if (loc == AppRoutePath.avatar) return _avatarNameForHeader();
  if (loc == AppRoutePath.cart) return 'Cart';

  if (loc == AppRoutePath.preview) return 'Preview';
  if (state.uri.pathSegments.length == 1 &&
      state.uri.pathSegments.first.isNotEmpty &&
      !_isReservedTopSegment(state.uri.pathSegments.first)) {
    return 'Preview';
  }

  if (loc == AppRoutePath.payment) return 'Payment';
  if (loc == AppRoutePath.walletContents) return 'Token';

  if (loc == AppRoutePath.avatarEdit) return 'Edit Avatar';
  if (loc == AppRoutePath.userEdit) return 'Account';
  return null;
}

String _resolveAvatarIdForHeader(GoRouterState state) {
  // ✅ URL からは取らない（セキュリティ要件）
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
      path == AppRoutePath.payment ||
      path == AppRoutePath.walletContents) {
    return const [];
  }

  if (!allowLogin) return const [];

  final avatarId = _resolveAvatarIdForHeader(state);
  final returnTo = state.uri.toString();

  if (!firebaseReady) {
    if (isHome) {
      return [
        _HeaderCartButton(returnTo: returnTo, avatarId: avatarId),
        _HeaderSignInButton(to: AppRoutePath.login, returnTo: returnTo),
      ];
    }
    return [_HeaderSignInButton(to: AppRoutePath.login, returnTo: returnTo)];
  }

  final isLoggedIn = FirebaseAuth.instance.currentUser != null;

  if (!isLoggedIn) {
    if (isHome) {
      return [
        _HeaderCartButton(returnTo: returnTo, avatarId: avatarId),
        _HeaderSignInButton(to: AppRoutePath.login, returnTo: returnTo),
      ];
    }
    return [_HeaderSignInButton(to: AppRoutePath.login, returnTo: returnTo)];
  }

  if (isAvatar) {
    return [_HeaderHamburgerMenuButton(returnTo: returnTo)];
  }

  if (isHome) {
    return [_HeaderCartButton(returnTo: returnTo, avatarId: avatarId)];
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
  const _HeaderSignInButton({required this.to, required this.returnTo});
  final String to;
  final String returnTo;

  @override
  Widget build(BuildContext context) {
    return TextButton(
      onPressed: () {
        // ✅ Pattern B: login 遷移前に戻り先を Store に保存
        _captureReturnToForInternalNav(returnTo);
        context.go(to);
      },
      child: const Text('Sign in'),
    );
  }
}

class _HeaderCartButton extends StatefulWidget {
  const _HeaderCartButton({required this.returnTo, required this.avatarId});

  final String returnTo;

  // NOTE: 旧実装では avatarId は未使用だったが、
  // バッジ表示のために使用する
  final String avatarId;

  @override
  State<_HeaderCartButton> createState() => _HeaderCartButtonState();
}

class _HeaderCartButtonState extends State<_HeaderCartButton> {
  CartRepositoryHttp? _repo;
  Future<int>? _futureQty;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  @override
  void initState() {
    super.initState();
    _repo = CartRepositoryHttp();
    _futureQty = _loadTotalQty();
  }

  @override
  void didUpdateWidget(covariant _HeaderCartButton oldWidget) {
    super.didUpdateWidget(oldWidget);

    // avatarId が変わったら取り直す
    if (_s(oldWidget.avatarId) != _s(widget.avatarId)) {
      _futureQty = _loadTotalQty();
    }
  }

  @override
  void dispose() {
    _repo?.dispose();
    super.dispose();
  }

  Future<int> _loadTotalQty() async {
    final aid = _s(widget.avatarId);

    // ログイン前 / avatarId 未解決ならバッジなし（0扱い）
    final loggedIn = FirebaseAuth.instance.currentUser != null;
    if (!loggedIn || aid.isEmpty) return 0;

    try {
      final CartDTO c = await _repo!.fetchCart(avatarId: aid);

      // ✅ “アイテム数” は qty 合計で表示（行数ではなく点数）
      final totalQty = c.items.values.fold<int>(0, (sum, it) => sum + it.qty);
      return totalQty < 0 ? 0 : totalQty;
    } catch (e) {
      // 404 は空カート扱い
      if (e is CartHttpException && e.statusCode == 404) return 0;

      // それ以外は fail-open（バッジ無し）
      return 0;
    }
  }

  String _badgeText(int n) {
    if (n <= 0) return '';
    if (n > 99) return '99+';
    return n.toString();
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<int>(
      future: _futureQty,
      builder: (context, snap) {
        final qty = (snap.data ?? 0);
        final text = _badgeText(qty);
        final showBadge = text.isNotEmpty;

        return Stack(
          clipBehavior: Clip.none,
          children: [
            IconButton(
              tooltip: 'Cart',
              icon: const Icon(Icons.shopping_cart_outlined),
              onPressed: () {
                // ✅ Pattern B: cart 遷移前に戻り先を Store に保存
                _captureReturnToForInternalNav(widget.returnTo);
                context.goNamed(AppRouteName.cart);
              },
            ),

            if (showBadge)
              Positioned(
                right: 6,
                top: 6,
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 6,
                    vertical: 2,
                  ),
                  decoration: BoxDecoration(
                    color: Theme.of(context).colorScheme.error,
                    borderRadius: BorderRadius.circular(999),
                  ),
                  constraints: const BoxConstraints(minWidth: 18),
                  child: Text(
                    text,
                    textAlign: TextAlign.center,
                    style: const TextStyle(
                      fontSize: 11,
                      height: 1.1,
                      color: Colors.white,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
              ),
          ],
        );
      },
    );
  }
}

class _HeaderHamburgerMenuButton extends StatelessWidget {
  const _HeaderHamburgerMenuButton({required this.returnTo});
  final String returnTo;

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
          builder: (_) => _AccountMenuSheet(returnTo: returnTo),
        );
      },
    );
  }
}

class _AccountMenuSheet extends StatelessWidget {
  const _AccountMenuSheet({required this.returnTo});
  final String returnTo;

  void _go(BuildContext context, String path, {Map<String, String>? qp}) {
    Navigator.pop(context);

    // ✅ Pattern B: メニュー遷移前に戻り先を Store に保存
    _captureReturnToForInternalNav(returnTo);

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
                  onTap: () => _go(context, AppRoutePath.userEdit),
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
                  onTap: () => _go(context, AppRoutePath.billingAddress),
                ),
                _divider(context),
                ListTile(
                  leading: const Icon(Icons.email_outlined),
                  title: const Text('メールアドレス'),
                  subtitle: const Text('Email'),
                  onTap: () => _go(
                    context,
                    AppRoutePath.userEdit,
                    qp: {AppQueryKey.tab: 'email'},
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
                    qp: {AppQueryKey.tab: 'password'},
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
