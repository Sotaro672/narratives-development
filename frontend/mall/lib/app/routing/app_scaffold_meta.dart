// frontend/mall/lib/app/routing/app_scaffold_meta.dart
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/foundation.dart';
import 'package:go_router/go_router.dart';

// ✅ import は package: に統一（相対 import 混在で store が二重化するのを防ぐ）
import 'package:mall/app/routing/routes.dart';
import 'package:mall/app/routing/avatar_name_store.dart';

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

  // ✅ DEBUG: resolveTitleFor が呼ばれているか確認
  final storeName = AvatarNameStore.I.avatarName.trim();
  debugPrint(
    '[meta] resolveTitleFor() loc="$loc" store.avatarName="$storeName"',
  );
  // ignore: avoid_print
  print('[meta] resolveTitleFor() loc="$loc" store.avatarName="$storeName"');

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
  // ✅ DEBUG: _avatarNameForHeader が呼ばれているか確認
  final storeName = AvatarNameStore.I.avatarName.trim();
  debugPrint('[meta] _avatarNameForHeader() store.avatarName="$storeName"');
  // ignore: avoid_print
  print('[meta] _avatarNameForHeader() store.avatarName="$storeName"');

  // ✅ 1) Backend (/mall/me/avatars) resolved name (highest priority)
  final bn = storeName;
  if (bn.isNotEmpty) {
    debugPrint('[meta] _avatarNameForHeader() -> use store "$bn"');
    // ignore: avoid_print
    print('[meta] _avatarNameForHeader() -> use store "$bn"');
    return bn;
  }

  // ✅ 2) Fallback: FirebaseAuth (legacy behavior)
  final u = FirebaseAuth.instance.currentUser;
  if (u == null) {
    debugPrint(
      '[meta] _avatarNameForHeader() -> fallback "Profile" (user=null)',
    );
    // ignore: avoid_print
    print('[meta] _avatarNameForHeader() -> fallback "Profile" (user=null)');
    return 'Profile';
  }

  final dn = (u.displayName ?? '').trim();
  if (dn.isNotEmpty) {
    debugPrint('[meta] _avatarNameForHeader() -> fallback displayName="$dn"');
    // ignore: avoid_print
    print('[meta] _avatarNameForHeader() -> fallback displayName="$dn"');
    return dn;
  }

  final email = (u.email ?? '').trim();
  if (email.isNotEmpty) {
    final i = email.indexOf('@');
    final out = (i > 0) ? email.substring(0, i) : email;
    debugPrint(
      '[meta] _avatarNameForHeader() -> fallback email="$out" (raw="$email")',
    );
    // ignore: avoid_print
    print(
      '[meta] _avatarNameForHeader() -> fallback email="$out" (raw="$email")',
    );
    return out;
  }

  debugPrint(
    '[meta] _avatarNameForHeader() -> fallback "My Profile" (no displayName/email)',
  );
  // ignore: avoid_print
  print(
    '[meta] _avatarNameForHeader() -> fallback "My Profile" (no displayName/email)',
  );
  return 'My Profile';
}
