// frontend\mall\lib\app\routing\navigation.dart
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../config/api_base.dart';
import 'routes.dart';

/// ------------------------------------------------------------
/// ✅ Pattern B: URL の `from` 制御を廃止し、narratives.jp 内で navigation state を保持する
///
/// - 戻り先は NavStore に保持（URL query には載せない）
/// - login 完了後は NavStore の returnTo があればそこへ復帰、無ければ home
/// - セキュリティ: returnTo は「内部パスのみ」許可（外部 URL / スキーム付きは拒否）
/// - ループ防止: login/create-account 等の auth ルートを returnTo として保存しない
class NavStore extends ChangeNotifier {
  NavStore._();
  static final NavStore I = NavStore._();

  String _returnTo = '';
  String get returnTo => _returnTo;

  /// ✅ 戻り先を保存（内部パスのみ / ループになるパスは禁止）
  void setReturnTo(String location) {
    final safe = _sanitizeInternalLocation(location);
    if (safe.isEmpty) return;

    // ループ防止: auth 系や導線系を戻り先にしない
    if (_isDisallowedReturnPath(safe)) return;

    if (safe == _returnTo) return;
    _returnTo = safe;
    notifyListeners();
  }

  /// ✅ 取り出したらクリア（one-shot）
  ///
  /// Pattern B では呼び出し側が `.trim()` したくなるため、
  /// null を返さず String を返す（空文字=無し）。
  String consumeReturnTo() {
    final v = _returnTo.trim();
    _returnTo = '';
    if (v.isNotEmpty) notifyListeners();
    return v;
  }

  void clear() {
    if (_returnTo.isEmpty) return;
    _returnTo = '';
    notifyListeners();
  }

  /// ----------------------------------------------------------
  /// Helpers
  static String _sanitizeInternalLocation(String raw) {
    final s = raw.trim();
    if (s.isEmpty) return '';

    final u = Uri.tryParse(s);

    // parse できない場合は「/ から始まる」ものだけ許容
    if (u == null) {
      return s.startsWith('/') ? s : '';
    }

    // 外部URL（scheme/authorityあり）は拒否
    if (u.hasScheme || u.hasAuthority) return '';

    // path が空の場合は拒否
    final path = u.path.trim();
    if (path.isEmpty || !path.startsWith('/')) return '';

    // fragment は不要なので落とす
    final normalized = u.replace(fragment: null);
    return normalized.toString();
  }

  static bool _isDisallowedReturnPath(String location) {
    // location は "/path?query" 形式を想定
    final u = Uri.tryParse(location);
    final path = (u?.path ?? location).trim();

    // auth 系・初期導線は戻り先にしない
    const disallowed = <String>{
      AppRoutePath.login,
      AppRoutePath.createAccount,
      AppRoutePath.shippingAddress,
      AppRoutePath.billingAddress,
      AppRoutePath.avatarCreate,
    };

    return disallowed.contains(path);
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

  /// ✅ /mall/me/avatars で「自分の avatarId(docId)」を解決する（uid を query に入れない）
  Future<String?> resolveMyAvatarId() {
    if (_avatarId.trim().isNotEmpty) return Future.value(_avatarId.trim());

    final running = _inflight;
    if (running != null) return running;

    final f = _resolveMe();
    _inflight = f;
    return f;
  }

  Future<String?> _resolveMe() async {
    try {
      final base = resolveApiBase().trim();
      if (base.isEmpty) return null;

      final b = Uri.tryParse(base);
      if (b == null || !b.hasScheme || !b.hasAuthority) return null;

      final uri = b.replace(
        path: _joinPaths(b.path, '/mall/me/avatars'),
        queryParameters: null,
        fragment: null,
      );

      final headers = <String, String>{
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      };

      final u = FirebaseAuth.instance.currentUser;
      if (u == null) return null;

      Future<String?> getToken(bool forceRefresh) async {
        final String? raw = await u.getIdToken(forceRefresh);
        final token = (raw ?? '').trim();
        return token.isEmpty ? null : token;
      }

      String? token = await getToken(false);
      if (token == null) return null;

      headers['Authorization'] = 'Bearer $token';

      http.Response res = await http.get(uri, headers: headers);

      if (res.statusCode == 401 || res.statusCode == 403) {
        token = await getToken(true);
        if (token == null) return null;
        headers['Authorization'] = 'Bearer $token';
        res = await http.get(uri, headers: headers);
      }

      if (res.statusCode == 404) return null;
      if (res.statusCode < 200 || res.statusCode >= 300) return null;

      final body = res.body.trim();
      if (body.isEmpty) return null;

      final jsonBody = jsonDecode(body);
      Map<String, dynamic>? m;
      if (jsonBody is Map<String, dynamic>) {
        m = jsonBody;
      } else if (jsonBody is Map) {
        m = jsonBody.cast<String, dynamic>();
      }
      if (m == null) return null;

      final data = (m['data'] is Map)
          ? (m['data'] as Map).cast<String, dynamic>()
          : m;

      final id = (data['avatarId'] ?? '').toString().trim();
      if (id.isNotEmpty) {
        set(id);
        return id;
      }
      return null;
    } catch (_) {
      return null;
    } finally {
      _inflight = null;
    }
  }
}

/// ------------------------------------------------------------
/// ✅ Header title 用の Store
/// - use_avatar.dart が /mall/me/avatars の avatarName をセットする
/// - app_scaffold_meta.dart がここから title を読む
class AvatarHeaderTitleStore extends ChangeNotifier {
  AvatarHeaderTitleStore._();
  static final AvatarHeaderTitleStore I = AvatarHeaderTitleStore._();

  String _title = 'Profile';
  String get title => _title;

  void setTitle(String? v) {
    final next = (v ?? '').trim();
    final normalized = next.isEmpty ? 'Profile' : next;
    if (_title == normalized) return;
    _title = normalized;
    notifyListeners();
  }

  void reset() => setTitle('Profile');
}

/// ------------------------------------------------------------
/// ✅ サインイン後に avatarId を確実に解決する
Future<String> _ensureAvatarIdResolved(GoRouterState state) async {
  final storeId = AvatarIdStore.I.avatarId.trim();
  if (storeId.isNotEmpty) return storeId;

  final all = state.uri.queryParametersAll;
  final list = all[AppQueryKey.avatarId] ?? const <String>[];
  final qpId = (list.isNotEmpty ? list.last : '').trim();
  final qpCandidate = qpId;

  final resolved = await AvatarIdStore.I.resolveMyAvatarId();
  final id = (resolved ?? '').trim();
  if (id.isNotEmpty) return id;

  return qpCandidate;
}

/// ------------------------------------------------------------
/// ✅ /:productId が “固定パス” と衝突した時の安全弁（navigation 側にも持つ）
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

/// ------------------------------------------------------------
/// ✅ redirect 本体（router.dart から呼ぶ）
Future<String?> appRedirect(BuildContext context, GoRouterState state) async {
  final user = FirebaseAuth.instance.currentUser;
  final path = state.uri.path;

  // ✅ DEBUG: redirect の判定をログで追えるようにする
  debugPrint(
    '[redirect] user=${user == null ? "null" : "in"} '
    'name=${state.name} path=${state.uri.path} loc=${state.uri}',
  );

  final isLoginRoute = path == AppRoutePath.login;
  final isCreateAccountRoute = path == AppRoutePath.createAccount;

  // ------------------------------------------------------------
  // ✅ 未ログイン
  // - AvatarId は確実にクリア
  // - Header title もリセット（メール/表示名依存を廃止したため）
  // - NavStore は “auth 復元の一瞬null” で消えると困るのでクリアしない
  // - 保護対象ページなら login に誘導し、returnTo を保存する（Pattern B）
  if (user == null) {
    AvatarIdStore.I.clear();
    AvatarHeaderTitleStore.I.reset();

    // ✅ 未ログインでも入れる route（動的パス対応のため name で判定）
    final allowWhenSignedOutByName = <String>{
      AppRouteName.home,
      AppRouteName.login,
      AppRouteName.createAccount,

      // ✅ expectation: public
      AppRouteName.preview,
      AppRouteName.catalog,
      AppRouteName.qrProduct, // ✅ /:productId (QR入口) も public
    };

    final name = state.name;

    // ✅ name が取れないケースの保険（home/login/create-account/preview は path でもOK）
    final allowWhenSignedOutByPath = <String>{
      AppRoutePath.home,
      AppRoutePath.login,
      AppRoutePath.createAccount,
      AppRoutePath.preview,
    };

    // ✅ IMPORTANT:
    // - /catalog/:listId は state.name が null になるケースがあるため prefix でも許可
    // - /:productId (QR入口) も name null 保険で許可（reserved は除外）
    final segs = state.uri.pathSegments;

    final isAllowed =
        (name != null && allowWhenSignedOutByName.contains(name)) ||
        allowWhenSignedOutByPath.contains(path) ||
        path.startsWith('/catalog/') ||
        path == '/catalog' ||
        (segs.length == 1 &&
            segs.first.isNotEmpty &&
            !_isReservedTopSegment(segs.first));

    // public に許可されていないページへ行こうとしているなら login へ
    if (!isAllowed) {
      final current = state.uri.toString();
      NavStore.I.setReturnTo(current);
      return AppRoutePath.login;
    }

    // 公開ページはそのまま
    return null;
  }

  // ✅ サインイン中でも avatarId を要求しないページ
  final exemptForAvatarId = <String>{
    AppRoutePath.login,
    AppRoutePath.createAccount,
    AppRoutePath.shippingAddress,
    AppRoutePath.billingAddress,
    AppRoutePath.avatarCreate,
  };

  // ============================================================
  // ✅ ログイン直後（/login に居る状態でログイン状態になった瞬間）
  // ============================================================
  if (isLoginRoute) {
    final resolved = await _ensureAvatarIdResolved(state);
    if (resolved.isNotEmpty) {
      AvatarIdStore.I.set(resolved);
    }

    final to = NavStore.I.consumeReturnTo().trim();
    if (to.isNotEmpty) {
      return to;
    }

    return AppRoutePath.home;
  }

  // ✅ create-account はログイン状態になっても強制遷移しない
  if (isCreateAccountRoute) {
    final resolved = await _ensureAvatarIdResolved(state); // best-effort
    if (resolved.isNotEmpty) {
      AvatarIdStore.I.set(resolved);
    }
    return null;
  }

  // ✅ exempt は best-effort で解決だけ試す（あれば store に入る）
  if (exemptForAvatarId.contains(path)) {
    final resolved = await _ensureAvatarIdResolved(state);
    if (resolved.isNotEmpty) {
      AvatarIdStore.I.set(resolved);
    }
    return null;
  }

  // ✅ サインイン後：avatarId は store に確保するが、URLは改変しない
  final resolved = await _ensureAvatarIdResolved(state);
  if (resolved.isNotEmpty) {
    AvatarIdStore.I.set(resolved);
  }

  return null;
}

/// ------------------------------------------------------------
/// ✅ Pattern B: 画面遷移ヘルパ
void goToAvatarEdit(BuildContext context) {
  final current = GoRouterState.of(context).uri.toString();
  NavStore.I.setReturnTo(current);

  context.go(AppRoutePath.avatarEdit);
}

/// create へ遷移（必要なら使う）
void goToAvatarCreate(BuildContext context) {
  final current = GoRouterState.of(context).uri.toString();
  NavStore.I.setReturnTo(current);

  context.go(AppRoutePath.avatarCreate);
}

/// ------------------------------------------------------------
/// Helpers
String _joinPaths(String a, String b) {
  final aa = a.trim();
  final bb = b.trim();
  if (aa.isEmpty || aa == '/') return bb.startsWith('/') ? bb : '/$bb';
  if (bb.isEmpty || bb == '/') return aa;
  if (aa.endsWith('/') && bb.startsWith('/')) return aa + bb.substring(1);
  if (!aa.endsWith('/') && !bb.startsWith('/')) return '$aa/$bb';
  return aa + bb;
}
