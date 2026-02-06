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

  bool _is2xx(int sc) => sc >= 200 && sc < 300;

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

    if (!_is2xx(res.statusCode)) {
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

  /// PATCH/PUT/POST のレスポンス JSON を読む（空ボディなら null）
  Future<Map<String, dynamic>?> _authedJson(
    String path, {
    required Map<String, dynamic> jsonBody,
    String method = 'PATCH',
  }) async {
    final uri = _api.uri(path);

    http.Response res;
    try {
      res = await _api.sendAuthed(method, uri, jsonBody: jsonBody);
    } catch (e) {
      throw AvatarApiException(
        'Network/auth request failed',
        path: path,
        cause: e,
      );
    }

    if (!_is2xx(res.statusCode)) {
      throw AvatarApiException(
        'Non-2xx response',
        path: path,
        statusCode: res.statusCode,
        body: res.body,
      );
    }

    if (res.body.trim().isEmpty) return null;

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
  // ✅ Contract:
  // - me 系: /mall/me/avatar
  // - create 系(正): POST /mall/avatars
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

  /// ✅ Edit プレフィルも “同じ契約” を使う
  Future<MeAvatar?> fetchMyAvatarProfile() async {
    return fetchMeAvatar();
  }

  // ===========================================================================
  // ✅ CREATE (新規作成)
  // POST /mall/avatars   <-- 正
  //
  // - 「未作成」の時はこっちを叩く
  // - avatarIcon はここでも禁止（画像操作は別API）
  //
  // NOTE:
  // backend AvatarHandler の body は userId/userUid を受け取る実装になっているため、
  // 呼び出し側で必要項目を含めてください。
  // ===========================================================================

  Future<MeAvatar> createAvatar(Map<String, dynamic> body) async {
    const p = '/mall/avatars';

    if (body.containsKey('avatarIcon')) {
      throw AvatarApiException(
        'avatarIcon is not allowed in POST /mall/avatars (icon ops are separate)',
        path: p,
        body: jsonEncode(body),
      );
    }

    // 送信前の軽い正規化（string は trim）
    final normalized = <String, dynamic>{};
    for (final e in body.entries) {
      final k = e.key.trim();
      if (k.isEmpty) continue;

      final v = e.value;
      if (v == null) continue;

      if (v is String) {
        normalized[k] = v.trim();
      } else {
        normalized[k] = v;
      }
    }

    if (normalized.isEmpty) {
      throw AvatarApiException('Empty body (nothing to create)', path: p);
    }

    final unwrapped = await _authedJson(
      p,
      method: 'POST',
      jsonBody: normalized,
    );

    // ✅ 空ボディでも、作成後は me が取れるはずなので取り直す
    if (unwrapped == null) {
      final latest = await fetchMeAvatar();
      if (latest == null) {
        throw AvatarApiException(
          'Create succeeded but fetchMeAvatar returned null',
          path: p,
        );
      }
      return latest;
    }

    return _parseMeAvatarStrict(unwrapped, path: p);
  }

  // ===========================================================================
  // ✅ UPDATE (編集)
  // PATCH /mall/me/avatar
  // ===========================================================================

  Future<MeAvatar> patchMeAvatar(Map<String, dynamic> patch) async {
    const p = '/mall/me/avatar';

    if (patch.containsKey('avatarIcon')) {
      throw AvatarApiException(
        'avatarIcon is not allowed in PATCH /mall/me/avatar (icon ops are separate)',
        path: p,
        body: jsonEncode(patch),
      );
    }

    final normalized = <String, dynamic>{};
    for (final e in patch.entries) {
      final k = e.key.trim();
      if (k.isEmpty) continue;

      final v = e.value;
      if (v == null) continue;

      if (v is String) {
        // "" はクリアとして意味があるので残す（server契約に従う）
        normalized[k] = v.trim();
      } else {
        normalized[k] = v;
      }
    }

    if (normalized.isEmpty) {
      throw AvatarApiException('Empty patch (nothing to update)', path: p);
    }

    final unwrapped = await _authedJson(p, jsonBody: normalized);

    if (unwrapped == null) {
      final latest = await fetchMeAvatar();
      if (latest == null) {
        throw AvatarApiException(
          'Patch succeeded but fetchMeAvatar returned null',
          path: p,
        );
      }
      return latest;
    }

    return _parseMeAvatarStrict(unwrapped, path: p);
  }

  // ===========================================================================
  // ✅ Icon ops (me-only)
  // ===========================================================================

  Future<Map<String, dynamic>> issueMeAvatarIconUploadUrl({
    String? fileName,
    String? mimeType,
    int? size,
  }) async {
    const p = '/mall/me/avatar/icon-upload-url';

    final body = <String, dynamic>{};
    final fn = _s(fileName);
    final mt = _s(mimeType);

    if (fn.isNotEmpty) body['fileName'] = fn;
    if (mt.isNotEmpty) body['mimeType'] = mt;
    if (size != null && size >= 0) body['size'] = size;

    final unwrapped = await _authedJson(p, method: 'POST', jsonBody: body);

    if (unwrapped == null) {
      throw AvatarApiException(
        'Empty response body (expected JSON object)',
        path: p,
      );
    }

    final uploadUrl = _s(unwrapped['uploadUrl']);
    final objectPath = _s(unwrapped['objectPath']);
    if (uploadUrl.isEmpty || objectPath.isEmpty) {
      throw AvatarApiException(
        'Invalid payload: missing uploadUrl/objectPath',
        path: p,
        body: jsonEncode(unwrapped),
      );
    }

    return unwrapped;
  }

  Future<void> deleteMeAvatarIconObject() async {
    const p = '/mall/me/avatar/icon-object';

    final uri = _api.uri(p);

    http.Response res;
    try {
      res = await _api.sendAuthed('DELETE', uri, jsonBody: null);
    } catch (e) {
      throw AvatarApiException(
        'Network/auth request failed',
        path: p,
        cause: e,
      );
    }

    if (!_is2xx(res.statusCode)) {
      throw AvatarApiException(
        'Non-2xx response',
        path: p,
        statusCode: res.statusCode,
        body: res.body,
      );
    }
  }

  void dispose() {
    _api.dispose();
  }
}
