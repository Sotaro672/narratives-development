// frontend\sns\lib\app\routing\navigation.dart
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import 'routes.dart';

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

      // ✅ 可能なら Authorization を付ける
      final headers = <String, String>{
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      };

      try {
        final u = FirebaseAuth.instance.currentUser;
        if (u != null) {
          final String? raw = await u.getIdToken(false);
          final token = (raw ?? '').trim();
          if (token.isNotEmpty) {
            headers['Authorization'] = 'Bearer $token';
          }
        }
      } catch (_) {}

      final res = await http.get(uri, headers: headers);

      if (res.statusCode == 404) return null;
      if (res.statusCode < 200 || res.statusCode >= 300) return null;

      final jsonBody = jsonDecode(res.body);
      if (jsonBody is Map<String, dynamic>) {
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
      _inflight = null;
    }
  }
}

/// ✅ API_BASE を読む（既存設計に合わせる）
String _apiBase() {
  const v = String.fromEnvironment('API_BASE');
  return v.trim();
}

/// ✅ URLの query を「必ず avatarId を1つだけ」に正規化して返す
Map<String, String> _normalizedQueryWithSingleAvatarId(
  GoRouterState state,
  String resolvedAvatarId,
) {
  final all = state.uri.queryParametersAll; // ここが重要（複数keyに対応）
  final out = <String, String>{};

  all.forEach((k, vals) {
    if (vals.isEmpty) return;

    // avatarId は一旦捨てる（最後に必ず1個だけ入れる）
    if (k == AppQueryKey.avatarId) return;

    // 同名キーが複数ある場合は「最後」を採用（安定させる）
    out[k] = vals.last;
  });

  out[AppQueryKey.avatarId] = resolvedAvatarId.trim();
  return out;
}

Future<String> _ensureAvatarIdResolved(GoRouterState state, String uid) async {
  // ✅ まず URL から拾う（重複してても最後を採用）
  final all = state.uri.queryParametersAll;
  final list = all[AppQueryKey.avatarId] ?? const <String>[];
  final qpId = (list.isNotEmpty ? list.last : '').trim();

  if (qpId.isNotEmpty) {
    AvatarIdStore.I.set(qpId);
    return qpId;
  }

  final storeId = AvatarIdStore.I.avatarId.trim();
  if (storeId.isNotEmpty) return storeId;

  final resolved = await AvatarIdStore.I.resolveAvatarIdByUserId(uid);
  return (resolved ?? '').trim();
}

/// ------------------------------------------------------------
/// ✅ redirect 本体（router.dart から呼ぶ）
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

  final uid = user.uid.trim();

  // ============================================================
  // ✅ ログイン直後（/login に居る状態でログイン状態になった瞬間）は
  // 必ず avatarId を解決して home(list.dart) へ `?avatarId=...` を付けて遷移
  // ============================================================
  if (isLoginRoute) {
    final resolved = await _ensureAvatarIdResolved(state, uid);

    if (resolved.isNotEmpty) {
      return Uri(
        path: AppRoutePath.home,
        queryParameters: {AppQueryKey.avatarId: resolved},
      ).toString();
    }

    // 取れない場合は従来通り home（ただし強制 avatar_create はしない）
    return AppRoutePath.home;
  }

  // ✅ create_account(/create-account) は、ログイン状態になっても強制遷移しない
  if (isCreateAccountRoute) {
    await _ensureAvatarIdResolved(state, uid); // best-effort
    return null;
  }

  // ✅ exempt は best-effort で解決だけ試す（あれば store に入る）
  if (exemptForAvatarId.contains(path)) {
    await _ensureAvatarIdResolved(state, uid);
    return null;
  }

  // ✅ サインイン後：原則「全ページ URL に avatarId を必ず持たせる」
  // ❌ avatarId 未解決でも avatar_create には飛ばさない（既存要件維持）
  final resolved = await _ensureAvatarIdResolved(state, uid);
  if (resolved.isEmpty) return null;

  // ✅ ここで「avatarIdが複数付いていても」必ず1つに正規化する
  final normalized = _normalizedQueryWithSingleAvatarId(state, resolved);
  final next = state.uri.replace(queryParameters: normalized).toString();

  if (next != state.uri.toString()) return next;
  return null;
}
