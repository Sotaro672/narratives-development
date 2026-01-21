// frontend\mall\lib\features\avatar\infrastructure\avatar_api_client.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../presentation/model/me_avatar.dart';
import 'api.dart';

/// 404 以外の失敗を hasError に載せるための例外
class AvatarApiException implements Exception {
  AvatarApiException(
    this.message, {
    this.path,
    this.statusCode,
    this.body,
    this.cause,
  });

  final String message;
  final String? path;
  final int? statusCode;
  final String? body;
  final Object? cause;

  @override
  String toString() {
    final sc = statusCode == null ? '' : ' status=$statusCode';
    final p = path == null ? '' : ' path=$path';
    return 'AvatarApiException($message$p$sc)';
  }
}

class AvatarApiClient {
  AvatarApiClient({http.Client? client}) : _api = MallAuthedApi(client: client);

  final MallAuthedApi _api;

  static String _s(Object? v) => (v ?? '').toString().trim();

  static String? _opt(Map<String, dynamic> j, String key) {
    if (!j.containsKey(key)) return null;
    final v = _s(j[key]);
    return v.isEmpty ? null : v;
  }

  Map<String, dynamic> _decodeObject(String body) {
    final b = body.trim();
    if (b.isEmpty) {
      throw const FormatException('Empty response body (expected JSON object)');
    }

    final decoded = jsonDecode(b);
    if (decoded is Map<String, dynamic>) return decoded;
    if (decoded is Map) return Map<String, dynamic>.from(decoded);

    throw const FormatException('Invalid JSON shape (expected object)');
  }

  /// `{"data": ...}` を unwrap（許容）
  Map<String, dynamic> _unwrapData(Map<String, dynamic> decoded) {
    return _api.unwrapData(decoded);
  }

  /// 404 は「未作成」扱いで null を返す。
  /// それ以外の失敗は例外として投げ、useFuture().hasError に載せる。
  Future<Map<String, dynamic>?> _getAuthedJson(String path) async {
    final uri = _api.uri(path);

    http.Response res;
    try {
      res = await _api.sendAuthed('GET', uri, jsonBody: null);
    } catch (e) {
      throw AvatarApiException(
        'Network/auth request failed',
        path: path,
        cause: e,
      );
    }

    if (res.statusCode == 404) return null;

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw AvatarApiException(
        'Non-2xx response',
        path: path,
        statusCode: res.statusCode,
        body: res.body,
      );
    }

    try {
      final decoded = _decodeObject(res.body);
      return _unwrapData(decoded);
    } catch (e) {
      throw AvatarApiException(
        'Failed to decode/unwrap JSON',
        path: path,
        statusCode: res.statusCode,
        body: res.body,
        cause: e,
      );
    }
  }

  // ===========================================================================
  // ✅ Contract (NO legacy):
  // /mall/me/avatar must return TOP-LEVEL avatar patch object:
  //
  // Required:
  // - avatarId
  // - walletAddress
  //
  // Expected (shown on profile):
  // - avatarName
  // - avatarIcon (nullable)
  // - profile (nullable)
  // - externalLink (nullable)
  //
  // ❌ NOT allowed:
  // - { "avatar": { ... } } wrapper
  // ===========================================================================
  MeAvatar _parseMeAvatarStrict(
    Map<String, dynamic> j, {
    required String path,
  }) {
    // legacy wrapper is explicitly rejected
    if (j.containsKey('avatar') && j['avatar'] is Map) {
      throw AvatarApiException(
        'Legacy payload is not allowed: wrapper key "avatar" detected',
        path: path,
        body: jsonEncode(j),
      );
    }

    final avatarId = _s(j['avatarId']);
    final walletAddress = _s(j['walletAddress']);

    if (avatarId.isEmpty) {
      throw AvatarApiException(
        'Invalid payload: missing avatarId',
        path: path,
        body: jsonEncode(j),
      );
    }
    if (walletAddress.isEmpty) {
      throw AvatarApiException(
        'Invalid payload: missing walletAddress',
        path: path,
        body: jsonEncode(j),
      );
    }

    // avatarName は “表示要件” なので原則必須に寄せる（未返却なら backend を直す）
    final avatarName = _s(j['avatarName']);
    if (avatarName.isEmpty) {
      throw AvatarApiException(
        'Invalid payload: missing avatarName',
        path: path,
        body: jsonEncode(j),
      );
    }

    return MeAvatar(
      avatarId: avatarId,
      walletAddress: walletAddress,
      avatarName: avatarName,
      avatarIcon: _opt(j, 'avatarIcon'),
      profile: _opt(j, 'profile'),
      externalLink: _opt(j, 'externalLink'),
    );
  }

  /// ✅ me avatar（avatar patch 全体）
  Future<MeAvatar?> fetchMeAvatar() async {
    const p = '/mall/me/avatar';

    final unwrapped = await _getAuthedJson(p);
    if (unwrapped == null) return null;

    return _parseMeAvatarStrict(unwrapped, path: p);
  }

  /// ✅ Edit プレフィルも “同じ契約” を使う（別 DTO を廃止して揃える）
  Future<MeAvatar?> fetchMyAvatarProfile() async {
    return fetchMeAvatar();
  }

  void dispose() {
    _api.dispose();
  }
}
