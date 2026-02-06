// frontend\mall\lib\features\avatar\infrastructure\avatar_api_client.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
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
  AvatarApiClient({
    http.Client? client,
    bool? enableLogging,
    void Function(String msg)? logger,
  }) : _api = MallAuthedApi(client: client),
       _enableLogging = enableLogging ?? (kDebugMode || kProfileMode),
       _logger = logger;

  final MallAuthedApi _api;

  final bool _enableLogging;
  final void Function(String msg)? _logger;

  // ----------------------------------------------------------------------------
  // ✅ logging helpers (確実に Chrome Console に出る)
  // ----------------------------------------------------------------------------

  void _log(String msg, {String? path}) {
    if (!_enableLogging) return;

    final p = (path ?? '').trim();
    final prefix = p.isEmpty ? '[AvatarApiClient] ' : '[AvatarApiClient][$p] ';
    final out = '$prefix$msg';

    // 呼び出し側で logger を渡したらそっちを優先
    if (_logger != null) {
      _logger(out);
      return;
    }

    // ✅ debugPrint は avoid_print に引っかからず、Web Console にも出る
    debugPrint(out);
  }

  String _short(String s, {int max = 280}) {
    final t = s.trim();
    if (t.length <= max) return t;
    return '${t.substring(0, max)}...';
  }

  // ----------------------------------------------------------------------------

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

    _log('GET start uri=$uri', path: path);

    http.Response res;
    try {
      res = await _api.sendAuthed('GET', uri, jsonBody: null);
    } catch (e) {
      _log('GET failed (network/auth) err=$e', path: path);
      throw AvatarApiException(
        'Network/auth request failed',
        path: path,
        cause: e,
      );
    }

    _log(
      'GET done status=${res.statusCode} bodyLen=${res.body.length} bodyHead="${_short(res.body)}"',
      path: path,
    );

    if (res.statusCode == 404) {
      _log('GET -> 404 (treat as null)', path: path);
      return null;
    }

    if (!_is2xx(res.statusCode)) {
      _log('GET -> non-2xx, throwing', path: path);
      throw AvatarApiException(
        'Non-2xx response',
        path: path,
        statusCode: res.statusCode,
        body: res.body,
      );
    }

    try {
      final decoded = _decodeObject(res.body);
      _log('decoded keys=${decoded.keys.toList()}', path: path);

      final unwrapped = _unwrapData(decoded);
      _log(
        'unwrapped keys=${unwrapped.keys.toList()} '
        'avatarId="${_s(unwrapped['avatarId'])}" '
        'avatarName="${_s(unwrapped['avatarName'])}" '
        'profile="${_s(unwrapped['profile'])}" '
        'walletAddress="${_s(unwrapped['walletAddress'])}"',
        path: path,
      );

      return unwrapped;
    } catch (e) {
      _log('decode/unwrap failed err=$e', path: path);
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

    _log(
      '$method start uri=$uri payloadKeys=${jsonBody.keys.toList()} payload="$jsonBody"',
      path: path,
    );

    http.Response res;
    try {
      res = await _api.sendAuthed(method, uri, jsonBody: jsonBody);
    } catch (e) {
      _log('$method failed (network/auth) err=$e', path: path);
      throw AvatarApiException(
        'Network/auth request failed',
        path: path,
        cause: e,
      );
    }

    _log(
      '$method done status=${res.statusCode} bodyLen=${res.body.length} bodyHead="${_short(res.body)}"',
      path: path,
    );

    if (!_is2xx(res.statusCode)) {
      _log('$method -> non-2xx, throwing', path: path);
      throw AvatarApiException(
        'Non-2xx response',
        path: path,
        statusCode: res.statusCode,
        body: res.body,
      );
    }

    if (res.body.trim().isEmpty) {
      _log('$method -> empty body (return null)', path: path);
      return null;
    }

    try {
      final decoded = _decodeObject(res.body);
      _log('decoded keys=${decoded.keys.toList()}', path: path);

      final unwrapped = _unwrapData(decoded);
      _log(
        'unwrapped keys=${unwrapped.keys.toList()} '
        'avatarId="${_s(unwrapped['avatarId'])}" '
        'avatarName="${_s(unwrapped['avatarName'])}" '
        'profile="${_s(unwrapped['profile'])}" '
        'walletAddress="${_s(unwrapped['walletAddress'])}"',
        path: path,
      );
      return unwrapped;
    } catch (e) {
      _log('decode/unwrap failed err=$e', path: path);
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
      _log('parse reject legacy wrapper key "avatar"', path: path);
      throw AvatarApiException(
        'Legacy payload is not allowed: wrapper key "avatar" detected',
        path: path,
        body: jsonEncode(j),
      );
    }

    final avatarId = _s(j['avatarId']);
    final walletAddress = _s(j['walletAddress']);

    if (avatarId.isEmpty) {
      _log(
        'parse invalid: missing avatarId json="${_short(jsonEncode(j))}"',
        path: path,
      );
      throw AvatarApiException(
        'Invalid payload: missing avatarId',
        path: path,
        body: jsonEncode(j),
      );
    }
    if (walletAddress.isEmpty) {
      _log(
        'parse invalid: missing walletAddress json="${_short(jsonEncode(j))}"',
        path: path,
      );
      throw AvatarApiException(
        'Invalid payload: missing walletAddress',
        path: path,
        body: jsonEncode(j),
      );
    }

    final avatarName = _s(j['avatarName']);
    if (avatarName.isEmpty) {
      _log(
        'parse invalid: missing avatarName json="${_short(jsonEncode(j))}"',
        path: path,
      );
      throw AvatarApiException(
        'Invalid payload: missing avatarName',
        path: path,
        body: jsonEncode(j),
      );
    }

    final icon = _opt(j, 'avatarIcon');
    final profile = _opt(j, 'profile');
    final link = _opt(j, 'externalLink');

    _log(
      'parse ok avatarId="$avatarId" avatarName="$avatarName" profile="${profile ?? ""}"',
      path: path,
    );

    return MeAvatar(
      avatarId: avatarId,
      walletAddress: walletAddress,
      avatarName: avatarName,
      avatarIcon: icon,
      profile: profile,
      externalLink: link,
    );
  }

  /// ✅ me avatar（avatar patch 全体）
  Future<MeAvatar?> fetchMeAvatar() async {
    const p = '/mall/me/avatar';

    final unwrapped = await _getAuthedJson(p);
    if (unwrapped == null) {
      _log('fetchMeAvatar -> null (404)', path: p);
      return null;
    }

    final me = _parseMeAvatarStrict(unwrapped, path: p);

    _log(
      'fetchMeAvatar result avatarName="${me.avatarName ?? ""}" profile="${me.profile ?? ""}"',
      path: p,
    );

    return me;
  }

  /// ✅ Edit プレフィルも “同じ契約” を使う
  Future<MeAvatar?> fetchMyAvatarProfile() async {
    return fetchMeAvatar();
  }

  /// ✅ CREATE (新規作成) POST /mall/avatars
  Future<MeAvatar> createAvatar(Map<String, dynamic> body) async {
    const p = '/mall/avatars';

    if (body.containsKey('avatarIcon')) {
      throw AvatarApiException(
        'avatarIcon is not allowed in POST /mall/avatars (icon ops are separate)',
        path: p,
        body: jsonEncode(body),
      );
    }

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

    _log('createAvatar normalized="$normalized"', path: p);

    final unwrapped = await _authedJson(
      p,
      method: 'POST',
      jsonBody: normalized,
    );

    if (unwrapped == null) {
      _log('createAvatar -> empty body, refetch me', path: p);
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

  /// ✅ UPDATE (編集) PATCH /mall/me/avatar
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
        normalized[k] = v.trim(); // "" は残す
      } else {
        normalized[k] = v;
      }
    }

    if (normalized.isEmpty) {
      throw AvatarApiException('Empty patch (nothing to update)', path: p);
    }

    _log('patchMeAvatar normalized="$normalized"', path: p);

    final unwrapped = await _authedJson(p, jsonBody: normalized);

    if (unwrapped == null) {
      _log('patchMeAvatar -> empty body, refetch me', path: p);
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

  void dispose() {
    _api.dispose();
  }
}
