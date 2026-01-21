// frontend/mall/lib/app/routing/app_scaffold_meta.dart
import 'package:firebase_auth/firebase_auth.dart';
import 'package:go_router/go_router.dart';

import 'routes.dart';

/// ------------------------------------------------------------
/// AppShell に渡す「タイトル」「戻るボタン表示」を router/app_routes から分離
/// ------------------------------------------------------------

bool resolveShowBackFor(GoRouterState state) {
  final path = state.uri.path;
  if (path == AppRoutePath.home) return false;
  return true;
}

String? resolveTitleFor(GoRouterState state) {
  final loc = state.uri.path;

  if (loc == AppRoutePath.home) return null;
  if (loc.startsWith('/catalog/')) return 'Catalog';

  if (loc == AppRoutePath.avatar) return _avatarNameForHeader();
  if (loc == AppRoutePath.cart) return 'Cart';

  if (loc == AppRoutePath.preview) return 'Preview';
  if (state.uri.pathSegments.length == 1 &&
      state.uri.pathSegments.first.isNotEmpty &&
      !_isReservedTopSegment(state.uri.pathSegments.first)) {
    // /:productId (QR入口) は Preview 扱い
    return 'Preview';
  }

  if (loc == AppRoutePath.payment) return 'Payment';
  if (loc == AppRoutePath.walletContents) return 'Token';

  if (loc == AppRoutePath.avatarEdit) return 'Edit Avatar';
  if (loc == AppRoutePath.userEdit) return 'Account';

  return null;
}

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
    'wallet',
  };
  return reserved.contains(seg);
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
