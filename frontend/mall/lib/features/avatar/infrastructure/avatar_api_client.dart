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

    // Cloud Run などで body が空の可能性があるが、GET は原則 JSON を返す想定
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
  Future<Map<String, dynamic>?> _patchAuthedJson(
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

    // ✅ 200 でも content-length:0 があり得る
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

  // ===========================================================================
  // ✅ Update ("PATCH") - me 系で完結
  //
  // IMPORTANT (推奨B / 現行運用):
  // - avatarIcon は me PATCH では更新しない（送らない）
  // - 画像実体の更新/削除は別エンドポイントで GCS object overwrite / delete を行う
  //
  // This client therefore:
  // - rejects "avatarIcon" in patchMeAvatar()
  // ===========================================================================

  /// PATCH /mall/me/avatar
  ///
  /// - patch は partial update（キーが無いものは更新しない）
  /// - 値が "" の場合は「クリア」を表現したいので、そのまま送る（backend 契約に従う）
  /// - 成功時のレスポンスが空の場合は GET で取り直す
  ///
  /// ❌ avatarIcon は禁止（推奨B：フロントが送らない運用で担保）
  Future<MeAvatar> patchMeAvatar(Map<String, dynamic> patch) async {
    const p = '/mall/me/avatar';

    // 明示的に禁止（事故防止）
    if (patch.containsKey('avatarIcon')) {
      throw AvatarApiException(
        'avatarIcon is not allowed in PATCH /mall/me/avatar (icon ops are separate)',
        path: p,
        body: jsonEncode(patch),
      );
    }

    // 送信前の軽い正規化（string は trim）
    final normalized = <String, dynamic>{};
    for (final e in patch.entries) {
      final k = e.key.trim();
      if (k.isEmpty) continue;

      final v = e.value;
      if (v == null) continue; // nil は「送らない」

      if (v is String) {
        // "" はクリアとして意味があるので残す
        normalized[k] = v.trim();
      } else {
        normalized[k] = v;
      }
    }

    if (normalized.isEmpty) {
      throw AvatarApiException('Empty patch (nothing to update)', path: p);
    }

    final unwrapped = await _patchAuthedJson(p, jsonBody: normalized);

    // ✅ 空ボディなら GET で取り直す（Cloud Run の 200 + 0 bytes 対策）
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
  // - POST   /mall/me/avatar/icon-upload-url  -> signed PUT url for fixed objectPath
  // - DELETE /mall/me/avatar/icon-object      -> delete fixed object
  //
  // NOTE:
  // - These endpoints do NOT change avatarIcon string in DB.
  // - The UI keeps using the existing avatarIcon URL.
  // ===========================================================================

  /// POST /mall/me/avatar/icon-upload-url
  ///
  /// Returns (best-effort; server contract):
  /// - uploadUrl, bucket, objectPath, gsUrl, publicUrl, expiresAt, contentType
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

    final unwrapped = await _patchAuthedJson(p, method: 'POST', jsonBody: body);

    if (unwrapped == null) {
      throw AvatarApiException(
        'Empty response body (expected JSON object)',
        path: p,
      );
    }

    // 최소한 uploadUrl / objectPath 정도는 있어야 함
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

  /// DELETE /mall/me/avatar/icon-object
  ///
  /// Deletes only the GCS object for the fixed path.
  /// (avatarIcon string remains unchanged)
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

    // 204 expected, but accept any 2xx as success
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
