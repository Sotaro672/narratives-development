// frontend/sns/lib/app/routing/router.dart
import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

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

// ✅ NEW: payment page
import '../../features/payment/presentation/page/payment.dart';

// ✅ SnsListItem 型
import '../../features/list/infrastructure/list_repository_http.dart';

// auth pages
import '../../features/auth/presentation/page/login_page.dart';
import '../../features/auth/presentation/page/create_account.dart';
import '../../features/auth/presentation/page/shipping_address.dart';
import '../../features/auth/presentation/page/billing_address.dart';
import '../../features/auth/presentation/page/avatar_create.dart';

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

/// ------------------------------------------------------------
/// ✅ avatarId の “現在値” をアプリ側で保持（URLに無い時の補完に使う）
class AvatarIdStore extends ChangeNotifier {
  AvatarIdStore._();
  static final AvatarIdStore I = AvatarIdStore._();

  String _avatarId = '';
  String get avatarId => _avatarId;

  // ✅ 1ユーザーにつき1つの in-flight 解決（redirect 連打で多重に叩かない）
  Future<String?>? _inflight;

  void set(String v) {
    final next = v.trim();
    if (next.isEmpty) return;
    if (next == _avatarId) return;
    _avatarId = next;
    notifyListeners();
  }

  void clear() {
    if (_avatarId.isEmpty) return;
    _avatarId = '';
    _inflight = null;
    notifyListeners();
  }

  /// ✅ uid -> avatarId をバックエンドで解決
  Future<String?> resolveAvatarIdByUserId(String userId) {
    final uid = userId.trim();
    if (uid.isEmpty) return Future.value(null);

    // 既に確定しているならそれを返す
    if (_avatarId.trim().isNotEmpty) {
      return Future.value(_avatarId.trim());
    }

    // in-flight があればそれを待つ
    final running = _inflight;
    if (running != null) return running;

    final f = _resolve(uid);
    _inflight = f;
    return f;
  }

  Future<String?> _resolve(String userId) async {
    try {
      final base = _apiBase();
      if (base.isEmpty) return null;

      final uri = Uri.parse(
        base,
      ).replace(path: '/sns/avatars', queryParameters: {'userId': userId});

      // ✅ 可能なら Authorization を付ける（権限必須の環境で 401 になって redirect が暴れるのを防ぐ）
      final headers = <String, String>{
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      };
      try {
        final u = FirebaseAuth.instance.currentUser;
        if (u != null) {
          final raw = await u.getIdToken(false);
          final token = (raw ?? '').trim();
          if (token.isNotEmpty) {
            headers['Authorization'] = 'Bearer $token';
          }
        }
      } catch (_) {
        // best-effort
      }

      final res = await http.get(uri, headers: headers);

      if (res.statusCode == 404) return null;
      if (res.statusCode < 200 || res.statusCode >= 300) {
        return null;
      }

      final jsonBody = jsonDecode(res.body);
      if (jsonBody is Map<String, dynamic>) {
        // backend が返すキー揺れを吸収
        final id = (jsonBody['id'] ?? jsonBody['avatarId'] ?? '')
            .toString()
            .trim();
        if (id.isNotEmpty) {
          set(id);
          return id;
        }
      }
      return null;
    } catch (_) {
      return null;
    } finally {
      // 次の解決要求に備える
      _inflight = null;
    }
  }
}

/// ✅ API_BASE を読む（既存設計に合わせる）
String _apiBase() {
  // 既存ログから API_BASE を想定
  const v = String.fromEnvironment('API_BASE');
  return v.trim();
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
    redirect: (context, state) async {
      final user = FirebaseAuth.instance.currentUser;
      final isLoggedIn = user != null;

      final path = state.uri.path;
      final qp = state.uri.queryParameters;

      final isLoginRoute = path == AppRoutePath.login;
      final isCreateAccountRoute = path == AppRoutePath.createAccount;

      // ✅ サインイン中でも avatarId を要求しないページ（作成/住所登録など）
      final exemptForAvatarId = <String>{
        AppRoutePath.login,
        AppRoutePath.createAccount,
        AppRoutePath.shippingAddress,
        AppRoutePath.billingAddress,
        AppRoutePath.avatarCreate,
      };

      // ------------------------------------------------------------
      // ✅ 0) 未ログイン
      // ------------------------------------------------------------
      if (!isLoggedIn) {
        // store は残しても良いが、混線防止でクリア
        AvatarIdStore.I.clear();
        return null;
      }

      // ------------------------------------------------------------
      // ✅ 1) login/createAccount -> サインイン後は from or home に戻す
      //    ※戻り先にも avatarId を必ず付与する（＝ここで解決する）
      // ------------------------------------------------------------
      if (isLoggedIn && (isLoginRoute || isCreateAccountRoute)) {
        final rawFromEncoded = (qp[AppQueryKey.from] ?? '').trim();
        final rawFrom = _decodeFrom(rawFromEncoded);
        final uid = user.uid.trim();

        // URL/store に avatarId が無ければ uid で解決
        final resolved = await _ensureAvatarIdResolved(state, uid);
        if (resolved.isEmpty) {
          // avatar が無い（= 初回）なら avatarCreate へ
          final from = rawFrom.isNotEmpty ? rawFrom : AppRoutePath.home;
          return Uri(
            path: AppRoutePath.avatarCreate,
            queryParameters: {
              AppQueryKey.from: _encodeFrom(from),
              AppQueryKey.intent: 'bootstrap',
            },
          ).toString();
        }

        if (rawFrom.isNotEmpty) {
          final fixed = _withAvatarId(rawFrom, resolved);
          if (Uri.tryParse(fixed)?.path == AppRoutePath.login) {
            return Uri(
              path: AppRoutePath.home,
              queryParameters: {AppQueryKey.avatarId: resolved},
            ).toString();
          }
          return fixed;
        }

        return Uri(
          path: AppRoutePath.home,
          queryParameters: {AppQueryKey.avatarId: resolved},
        ).toString();
      }

      // ------------------------------------------------------------
      // ✅ 2) サインイン後：原則「全ページ URL に avatarId を必ず持たせる」
      //    - ただし exempt ページは除外
      // ------------------------------------------------------------
      final uid = user.uid.trim();

      // exempt ページは avatarId 無くても通す（ただし store は解決しておく）
      if (exemptForAvatarId.contains(path)) {
        await _ensureAvatarIdResolved(state, uid);
        return null;
      }

      // 通常ページ：avatarId を確定させる
      final resolved = await _ensureAvatarIdResolved(state, uid);
      if (resolved.isEmpty) {
        // avatar が無いなら avatarCreate へ
        final from = state.uri.toString();
        return Uri(
          path: AppRoutePath.avatarCreate,
          queryParameters: {
            AppQueryKey.from: _encodeFrom(from),
            AppQueryKey.intent: 'bootstrap',
          },
        ).toString();
      }

      // URLに avatarId が無い/違うなら正規化
      final qpId = (qp[AppQueryKey.avatarId] ?? '').trim();
      if (qpId != resolved) {
        final fixed = Map<String, String>.from(qp);
        fixed[AppQueryKey.avatarId] = resolved;

        final next = state.uri.replace(queryParameters: fixed).toString();
        if (next != state.uri.toString()) return next;
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

/// ✅ URL/store から avatarId を拾う。無ければ uid でバックエンド解決。
Future<String> _ensureAvatarIdResolved(GoRouterState state, String uid) async {
  final qp = state.uri.queryParameters;

  // 1) URL に avatarId があれば store に同期して採用
  final qpId = (qp[AppQueryKey.avatarId] ?? '').trim();
  if (qpId.isNotEmpty) {
    AvatarIdStore.I.set(qpId);
    return qpId;
  }

  // 2) store にあれば採用
  final storeId = AvatarIdStore.I.avatarId.trim();
  if (storeId.isNotEmpty) return storeId;

  // 3) uid で解決（= 期待値：avatars テーブルから userId で avatar を引く）
  final resolved = await AvatarIdStore.I.resolveAvatarIdByUserId(uid);
  return (resolved ?? '').trim();
}

/// ✅ 任意のURL文字列に avatarId を付与（既にあれば上書きしない）
String _withAvatarId(String raw, String avatarId) {
  final a = avatarId.trim();
  if (a.isEmpty) return raw;

  final u = Uri.tryParse(raw);
  if (u == null) return raw;

  final qp = <String, String>{...u.queryParameters};
  qp[AppQueryKey.avatarId] = a;
  return u.replace(queryParameters: qp).toString();
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
        final from = _decodeFrom(state.uri.queryParameters[AppQueryKey.from]);
        final intent = state.uri.queryParameters[AppQueryKey.intent];
        return NoTransitionPage(
          child: LoginPage(from: from.isEmpty ? null : from, intent: intent),
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
          child: CreateAccountPage(
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
          child: ShippingAddressPage(
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
          child: BillingAddressPage(from: from.isEmpty ? null : from),
        );
      },
    ),
    GoRoute(
      path: AppRoutePath.avatarCreate,
      name: AppRouteName.avatarCreate,
      pageBuilder: (context, state) {
        final from = _decodeFrom(state.uri.queryParameters[AppQueryKey.from]);
        return NoTransitionPage(
          child: AvatarCreatePage(from: from.isEmpty ? null : from),
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
            final initialItem = extra is SnsListItem ? extra : null;
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
        GoRoute(
          path: AppRoutePath.cart,
          name: AppRouteName.cart,
          pageBuilder: (context, state) {
            final qp = state.uri.queryParameters;
            final avatarId = (qp[AppQueryKey.avatarId] ?? '').trim();
            final from = _decodeFrom(qp[AppQueryKey.from]);

            return NoTransitionPage(
              key: ValueKey('cart-$avatarId'),
              child: CartPage(
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
            final avatarId = (qp[AppQueryKey.avatarId] ?? '').trim();
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

String? _titleFor(GoRouterState state) {
  final loc = state.uri.path;
  if (loc == AppRoutePath.home) return null;
  if (loc.startsWith('/catalog/')) return 'Catalog';
  if (loc == AppRoutePath.avatar) return 'Profile';
  if (loc == AppRoutePath.cart) return 'Cart';
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
                  leading: const Icon(Icons.account_circle_outlined),
                  title: const Text('アバター情報'),
                  subtitle: const Text('プロフィール編集'),
                  onTap: () => _go(
                    context,
                    AppRoutePath.avatarEdit,
                    qp: {AppQueryKey.from: _encodeFrom(from)},
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
