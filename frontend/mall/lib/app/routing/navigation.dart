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
  ///
  /// - 内部パスのみ許容:
  ///   - "/cart", "/avatar", "/preview?x=y" のような相対パスはOK
  ///   - "https://evil.com" や "javascript:..." は拒否
  ///
  /// NOTE:
  /// - GoRouter の location には query を含む文字列が来ることがあるため、
  ///   ここでは "Uri.tryParse" で解析し、authority/scheme があるものは弾く。
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
///
/// 重要（セキュリティ要件）:
/// - avatarId を URL に出さない方針のため、redirect で query へ注入しない。
/// - 代わりに AvatarIdStore に保持し、必要な画面で store から参照する。
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

  /// ✅ /mall/me/avatar で「自分の avatarId(docId)」を解決する（uid を query に入れない）
  Future<String?> resolveMyAvatarId() {
    // 既に確定しているならそれを返す
    if (_avatarId.trim().isNotEmpty) return Future.value(_avatarId.trim());

    // in-flight があればそれを待つ
    final running = _inflight;
    if (running != null) return running;

    final f = _resolveMe();
    _inflight = f;
    return f;
  }

  Future<String?> _resolveMe() async {
    try {
      // ✅ single source of truth: app/config/api_base.dart
      final base = resolveApiBase().trim();
      if (base.isEmpty) return null;

      final b = Uri.tryParse(base);
      if (b == null || !b.hasScheme || !b.hasAuthority) return null;

      // ✅ base の path を壊さず join する（replace(path:'/...') で潰さない）
      final uri = b.replace(
        path: _joinPaths(b.path, '/mall/me/avatar'),
        queryParameters: null,
        fragment: null,
      );

      // ✅ Authorization を付ける（必須）
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

      // まず通常取得（軽量）
      String? token = await getToken(false);
      if (token == null) return null;

      headers['Authorization'] = 'Bearer $token';

      http.Response res = await http.get(uri, headers: headers);

      // ✅ 401/403 のときだけ強制更新して 1 回だけリトライ
      if (res.statusCode == 401 || res.statusCode == 403) {
        token = await getToken(true);
        if (token == null) return null;
        headers['Authorization'] = 'Bearer $token';
        res = await http.get(uri, headers: headers);
      }

      if (res.statusCode == 404) return null;
      if (res.statusCode < 200 || res.statusCode >= 300) return null;

      // ✅ wrapper 吸収: {data:{avatarId:"..."}} を許容
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
/// ✅ サインイン後に avatarId を確実に解決する
///
/// - URLの avatarId は「信用しない」（uid が入っている事故を防ぐ）
/// - store があればそれを使う
/// - 最終的に /mall/me/avatar で解決
///
/// NOTE:
/// - redirect から無闇に呼ぶと「未作成ユーザー」で 404 を量産するため、
///   基本は login 遷移時（/login）など限定的なタイミングでのみ呼ぶ。
Future<String> _ensureAvatarIdResolved(GoRouterState state) async {
  // 1) store があればそれを採用（最優先）
  final storeId = AvatarIdStore.I.avatarId.trim();
  if (storeId.isNotEmpty) return storeId;

  // 2) URL の avatarId は “候補” として一時的に見るが、確定はしない
  final all = state.uri.queryParametersAll;
  final list = all[AppQueryKey.avatarId] ?? const <String>[];
  final qpId = (list.isNotEmpty ? list.last : '').trim();

  // qpId が入っていても set しない（uid混入を防ぐ）
  // ただし、/mall/me/avatar が取れない時の “保険” として最後に使えるよう保持
  final qpCandidate = qpId;

  // 3) サーバで確定（uid->avatarId の正規解）
  final resolved = await AvatarIdStore.I.resolveMyAvatarId();
  final id = (resolved ?? '').trim();
  if (id.isNotEmpty) return id;

  // 4) どうしても取れない場合だけ URL 候補を使う
  return qpCandidate;
}

/// ------------------------------------------------------------
/// ✅ redirect 本体（router.dart から呼ぶ）
///
/// Pattern B:
/// - `from` query を使わず、NavStore に returnTo を保持する
/// - login 完了後は NavStore の returnTo に復帰（なければ home）
///
/// セキュリティ要件:
/// - avatarId を URL に注入しない
/// - 代わりに store へ保存のみ行う
///
/// IMPORTANT:
/// - redirect から /mall/me/avatar を “常時 best-effort” で叩くと、
///   未作成ユーザー（avatar未作成）で 404 を量産し、不要なノイズになる。
/// - avatarId 解決は「本当に必要な画面側（編集/マイページ等）」で行う。
Future<String?> appRedirect(BuildContext context, GoRouterState state) async {
  final user = FirebaseAuth.instance.currentUser;

  final path = state.uri.path;
  final qp = state.uri.queryParameters;

  // ------------------------------------------------------------
  // 未ログイン
  // - avatarId / nav state をクリア
  // - ✅ ただし、メール認証リンク等の「oobCode 付き /shipping-address」は通す
  if (user == null) {
    AvatarIdStore.I.clear();
    NavStore.I.clear();

    if (path == AppRoutePath.shippingAddress && qp['oobCode'] != null) {
      return null; // allow landing page even when signed-out
    }

    return null;
  }

  final isLoginRoute = path == AppRoutePath.login;
  final isCreateAccountRoute = path == AppRoutePath.createAccount;

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
  // - avatarId は解決して store に入れる（best-effort）
  // - Pattern B: returnTo があればそこへ復帰。無ければ home。
  //
  // NOTE:
  // - ここは「ログイン直後」という限定的タイミングなので resolve を許可する。
  // ============================================================
  if (isLoginRoute) {
    final resolved = await _ensureAvatarIdResolved(state);
    if (resolved.isNotEmpty) {
      AvatarIdStore.I.set(resolved);
    }

    // ✅ 1) returnTo があれば復帰（consume は String を返す）
    final to = NavStore.I.consumeReturnTo().trim();
    if (to.isNotEmpty) {
      return to;
    }

    // ✅ 2) なければ home
    return AppRoutePath.home;
  }

  // ✅ create_account(/create-account) は、ログイン状態になっても強制遷移しない
  //
  // IMPORTANT:
  // - ここで /mall/me/avatar を叩く必要はない（未作成ユーザーの 404 ノイズになる）。
  if (isCreateAccountRoute) {
    return null;
  }

  // ✅ exempt は “avatarId 解決を試さない”
  //
  // IMPORTANT:
  // - billing-address / shipping-address / avatar-create は
  //   「未作成ユーザーでも通る導線」なので、redirect から resolve しない。
  if (exemptForAvatarId.contains(path)) {
    return null;
  }

  // ✅ それ以外の画面でも、redirect では avatarId を解決しない
  //
  // - avatarId が必要な画面（例: avatar edit / wallet contents 等）で、
  //   store が空なら各 feature 側で resolveMyAvatarId() を呼ぶ。
  return null;
}

/// ------------------------------------------------------------
/// ✅ Pattern B: 画面遷移ヘルパ
///
/// - URL query に `from` を載せない
/// - 代わりに NavStore に "returnTo" を保存してから遷移する
///
/// これにより、feature 側（use_avatar.dart 等）は
/// `goToAvatarEdit(context)` のみ呼べばよくなる。
void goToAvatarEdit(BuildContext context) {
  // 現在地（query含む）を returnTo として保存（URLには付けない）
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
