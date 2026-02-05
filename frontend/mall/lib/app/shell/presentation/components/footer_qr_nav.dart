import 'package:flutter/material.dart';

import '../../../routing/routes.dart';

/// QRでスキャンした文字列を「アプリ内遷移先URI」に正規化する責務だけを持つ。
///
/// ✅ Pattern B:
/// - `from` / `avatarId` / `mintAddress` を query に付与しない
/// - 既に含まれていた場合も削除する
class FooterQrNav {
  FooterQrNav._();

  /// ------------------------------------------------------------
  /// ✅ /:productId が固定パスと衝突しないように除外（scan時の安全弁）
  static bool isReservedTopSegment(String seg) {
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

  /// ------------------------------------------------------------
  /// ✅ QRでスキャンした文字列を「アプリ内遷移先URI」に正規化して返す
  ///
  /// - 生の productId だけ渡されたら /{productId} にする
  /// - http(s) URL なら path/query だけ抽出してアプリ内遷移にする
  /// - fragment は捨てる（ルーティング破壊回避）
  /// - query から from/avatarId/mintAddress を除去（Pattern B）
  static Uri? normalizeScannedToAppUri(String raw) {
    final s = raw.trim();
    if (s.isEmpty) return null;

    Uri? u;

    // 1) まず Uri として解釈
    try {
      u = Uri.parse(s);
    } catch (_) {
      u = null;
    }

    // 2) scheme が無い & / でもない場合は「生 productId」とみなす
    if (u == null || (u.scheme.isEmpty && !s.startsWith('/'))) {
      final pid = s.trim();
      if (pid.isEmpty) return null;
      if (isReservedTopSegment(pid)) return null;
      return Uri(path: '/$pid');
    }

    // 3) http(s) の場合は path/query だけ抽出してアプリ内遷移にする
    final extracted = Uri(
      path: (u.path.trim().isEmpty
          ? '/'
          : (u.path.startsWith('/') ? u.path : '/${u.path}')),
      queryParameters: (u.queryParameters.isEmpty ? null : u.queryParameters),
      fragment: null, // ✅ fragment は捨てる（ルーティング破壊回避）
    );

    // 4) パスがトップ1階層（= /{something}）の場合、reserved は弾く
    final segs = extracted.pathSegments;
    if (segs.length == 1) {
      final top = segs.first.trim();
      if (top.isNotEmpty && isReservedTopSegment(top)) {
        return null;
      }
    }

    // 5) Pattern B: URL state を持ち込まない（from/avatarId/mintAddress 等を除去）
    final merged = <String, String>{...extracted.queryParameters};

    merged.remove(AppQueryKey.from);
    merged.remove(AppQueryKey.avatarId);
    merged.remove(AppQueryKey.mintAddress);

    return extracted.replace(queryParameters: merged.isEmpty ? null : merged);
  }
}

/// small helper (optional)
void showInvalidScanSnackBar(BuildContext context) {
  ScaffoldMessenger.of(
    context,
  ).showSnackBar(const SnackBar(content: Text('スキャン結果が無効です（遷移できません）')));
}
