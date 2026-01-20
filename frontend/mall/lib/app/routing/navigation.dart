// frontend\mall\lib\app\routing\navigation.dart
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../config/api_base.dart';
import 'routes.dart';

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
/// セキュリティ要件:
/// - avatarId を URL に注入しない
/// - 代わりに store へ保存のみ行う
Future<String?> appRedirect(BuildContext context, GoRouterState state) async {
  final user = FirebaseAuth.instance.currentUser;

  if (user == null) {
    AvatarIdStore.I.clear();
    return null;
  }

  final path = state.uri.path;

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
  // - URL へ avatarId を付けずに home へ遷移（URL伸長/露出を防ぐ）
  // ============================================================
  if (isLoginRoute) {
    final resolved = await _ensureAvatarIdResolved(state);
    if (resolved.isNotEmpty) {
      AvatarIdStore.I.set(resolved);
    }
    return AppRoutePath.home;
  }

  // ✅ create_account(/create-account) は、ログイン状態になっても強制遷移しない
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
  // ❌ avatarId 未解決でも avatar_create には飛ばさない（既存要件維持）
  final resolved = await _ensureAvatarIdResolved(state);
  if (resolved.isNotEmpty) {
    AvatarIdStore.I.set(resolved);
  }

  // ✅ URL は触らない（avatarId を query に入れない / 正規化しない）
  return null;
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
