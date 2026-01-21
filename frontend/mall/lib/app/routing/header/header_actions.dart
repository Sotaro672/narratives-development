// frontend/mall/lib/app/routing/header/header_actions.dart
import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../routes.dart';
import '../navigation.dart';

// ✅ cart repository (for header badge)
import '../../../features/cart/infrastructure/repository_http.dart'
    show CartRepositoryHttp, CartDTO, CartHttpException;

/// ------------------------------------------------------------
/// AppShell header の actions を router/app_routes から分離
/// ------------------------------------------------------------

List<Widget> buildHeaderActionsFor(
  GoRouterState state, {
  required bool allowLogin,
  required bool firebaseReady,
}) {
  final path = state.uri.path;
  final isHome = path == AppRoutePath.home;
  final isAvatar = path == AppRoutePath.avatar;

  // actions を出したくない画面
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

  final avatarId = AvatarIdStore.I.avatarId;
  final returnTo = state.uri.toString();

  // Firebase 未初期化 or Web未設定時（public-only 相当）
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

/// ✅ Pattern B: 戻り先は URL ではなく Store へ保持
void _captureReturnToForInternalNav(String location) {
  try {
    NavStore.I.setReturnTo(location);
  } catch (_) {
    // fail-open
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

  // バッジ表示に使用
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

    final loggedIn = FirebaseAuth.instance.currentUser != null;
    if (!loggedIn || aid.isEmpty) return 0;

    try {
      final CartDTO c = await _repo!.fetchCart(avatarId: aid);
      final totalQty = c.items.values.fold<int>(0, (sum, it) => sum + it.qty);
      return totalQty < 0 ? 0 : totalQty;
    } catch (e) {
      if (e is CartHttpException && e.statusCode == 404) return 0;
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
                  onTap: () => _go(context, AppRoutePath.userEdit),
                ),
                _divider(context),
                ListTile(
                  leading: const Icon(Icons.local_shipping_outlined),
                  title: const Text('配送先住所'),
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
                  onTap: () => _go(context, AppRoutePath.billingAddress),
                ),
                _divider(context),
                ListTile(
                  leading: const Icon(Icons.email_outlined),
                  title: const Text('メールアドレス'),
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
                      if (context.mounted) {
                        context.go(AppRoutePath.home);
                      }
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
