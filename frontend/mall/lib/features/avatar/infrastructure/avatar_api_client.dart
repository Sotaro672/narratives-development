// frontend/mall/lib/features/avatar/infrastructure/avatar_api_client.dart
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
  // logging helpers
  // ----------------------------------------------------------------------------

  void _log(String msg, {String? path}) {
    if (!_enableLogging) return;

    final p = (path ?? '').trim();
    final prefix = p.isEmpty ? '[AvatarApiClient] ' : '[AvatarApiClient][$p] ';
    final out = '$prefix$msg';

    if (_logger != null) {
      _logger(out);
      return;
    }
    debugPrint(out);
  }

  String _short(String s, {int max = 280}) {
    final t = s.trim();
    if (t.length <= max) return t;
    return '${t.substring(0, max)}...';
  }

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
  // Contract
  // - me 系(正): /mall/me/avatars
  // - create 系(正): POST /mall/avatars
  // ===========================================================================

  MeAvatar _parseMeAvatarStrict(
    Map<String, dynamic> j, {
    required String path,
  }) {
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

    _log('parse ok avatarId="$avatarId" avatarName="$avatarName"', path: path);

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
    const p = '/mall/me/avatars';

    final unwrapped = await _getAuthedJson(p);
    if (unwrapped == null) {
      _log('fetchMeAvatar -> null (404)', path: p);
      return null;
    }
    return _parseMeAvatarStrict(unwrapped, path: p);
  }

  Future<MeAvatar?> fetchMyAvatarProfile() async {
    return fetchMeAvatar();
  }

  /// ✅ CREATE (新規作成) POST /mall/avatars
  Future<MeAvatar> createAvatar(Map<String, dynamic> body) async {
    const p = '/mall/avatars';

    // icon ops are separate
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
      // server should return created avatar; if not, refetch
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

  /// ✅ UPDATE (編集) PATCH /mall/me/avatars
  Future<MeAvatar> patchMeAvatar(Map<String, dynamic> patch) async {
    const p = '/mall/me/avatars';

    if (patch.containsKey('avatarIcon')) {
      throw AvatarApiException(
        'avatarIcon is not allowed in PATCH /mall/me/avatars (icon ops are separate)',
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
  // Icon upload (A) : /mall/avatars/{id}/icon-upload-url + /mall/avatars/{id}/icon
  // ===========================================================================

  /// ✅ /mall/avatars/{avatarId}/icon-upload-url
  /// Backend returns:
  /// {
  ///   "uploadUrl": "...",
  ///   "bucket": "...",
  ///   "objectPath": "...",
  ///   "gsUrl": "gs://.../...",
  ///   "expiresAt": "....Z"
  /// }
  ///
  /// NOTE:
  /// - publicUrl は usecase 側で生成し avatarIcon として返す方針のため、このDTOからは削除。
  Future<AvatarIconUploadUrl> issueAvatarIconUploadUrl({
    required String avatarId,
    String? fileName,
    required String mimeType,
    required int size,
  }) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is empty');

    final p = '/mall/avatars/$aid/icon-upload-url';

    final mt = mimeType.trim();
    final fn = (fileName ?? '').trim();

    final body = <String, dynamic>{
      if (fn.isNotEmpty) 'fileName': fn,
      'mimeType': mt.isEmpty ? 'application/octet-stream' : mt,
      'size': size,
    };

    _log('POST start payload=$body', path: p);

    final unwrapped = await _authedJson(p, method: 'POST', jsonBody: body);
    if (unwrapped == null) {
      throw AvatarApiException('Empty response (expected uploadUrl)', path: p);
    }

    final dto = AvatarIconUploadUrl.fromJson(unwrapped);

    _log(
      'issueAvatarIconUploadUrl ok '
      'bucket="${dto.bucket}" objectPath="${dto.objectPath}" '
      'expiresAt="${dto.expiresAt ?? ""}" gsUrl="${dto.gsUrl}"',
      path: p,
    );

    return dto;
  }

  /// ✅ /mall/avatars/{avatarId}/icon
  /// body:
  /// { bucket, objectPath, fileName?, size? }
  /// resp:
  /// { id, avatarId?, url, fileName?, size? }
  Future<AvatarIconResponse> registerAvatarIcon({
    required String avatarId,
    required String bucket,
    required String objectPath,
    String? fileName,
    int? size,
  }) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is empty');

    final p = '/mall/avatars/$aid/icon';

    final b = bucket.trim();
    final obj = objectPath.trim();

    if (b.isEmpty) throw ArgumentError('bucket is empty');
    if (obj.isEmpty) throw ArgumentError('objectPath is empty');

    final fn = (fileName ?? '').trim();

    final body = <String, dynamic>{
      'bucket': b,
      'objectPath': obj,
      if (fn.isNotEmpty) 'fileName': fn,
      if (size != null) 'size': size,
    };

    _log('POST start payload=$body', path: p);

    final unwrapped = await _authedJson(p, method: 'POST', jsonBody: body);
    if (unwrapped == null) {
      throw AvatarApiException('Empty response (expected icon json)', path: p);
    }

    final dto = AvatarIconResponse.fromJson(unwrapped);

    _log('registerAvatarIcon ok iconId="${dto.id}" url="${dto.url}"', path: p);

    return dto;
  }

  Future<void> uploadToSignedUrl({
    required String uploadUrl,
    required Uint8List bytes,
    required String contentType,
  }) async {
    final u = uploadUrl.trim();
    if (u.isEmpty) throw ArgumentError('uploadUrl is empty');
    if (bytes.isEmpty) throw ArgumentError('bytes is empty');

    final uri = Uri.parse(u);
    final ct = contentType.trim().isEmpty
        ? 'application/octet-stream'
        : contentType.trim();

    _log('PUT start signedUrl=$uri bytesLen=${bytes.length} contentType="$ct"');

    http.Response res;
    try {
      res = await http.put(
        uri,
        headers: <String, String>{'Content-Type': ct},
        body: bytes,
      );
    } catch (e) {
      _log('PUT failed err=$e');
      throw AvatarApiException('Signed URL PUT failed', cause: e);
    }

    _log(
      'PUT done status=${res.statusCode} bodyLen=${res.body.length} bodyHead="${_short(res.body)}"',
    );

    if (!_is2xx(res.statusCode)) {
      throw AvatarApiException(
        'Signed URL PUT non-2xx',
        statusCode: res.statusCode,
        body: res.body,
      );
    }
  }

  void dispose() {
    _api.dispose();
  }
}

/// ---------------------------------------------------------------------------
/// DTO: /mall/avatars/{id}/icon-upload-url response
/// ---------------------------------------------------------------------------
@immutable
class AvatarIconUploadUrl {
  const AvatarIconUploadUrl({
    required this.uploadUrl,
    required this.bucket,
    required this.objectPath,
    required this.gsUrl,
    this.expiresAt,
  });

  final String uploadUrl;
  final String bucket;
  final String objectPath;
  final String gsUrl;
  final String? expiresAt;

  static String _s(Object? v) => (v ?? '').toString().trim();
  static String? _optS(Object? v) {
    final s = _s(v);
    return s.isEmpty ? null : s;
  }

  factory AvatarIconUploadUrl.fromJson(Map<String, dynamic> json) {
    return AvatarIconUploadUrl(
      uploadUrl: _s(json['uploadUrl']),
      bucket: _s(json['bucket']),
      objectPath: _s(json['objectPath']),
      gsUrl: _s(json['gsUrl']),
      expiresAt: _optS(json['expiresAt']),
    );
  }
}

/// ---------------------------------------------------------------------------
/// DTO: /mall/avatars/{id}/icon response
/// { id, avatarId?, url, fileName?, size? }
/// ---------------------------------------------------------------------------
@immutable
class AvatarIconResponse {
  const AvatarIconResponse({
    required this.id,
    required this.url,
    this.avatarId,
    this.fileName,
    this.size,
  });

  final String id;
  final String url;
  final String? avatarId;
  final String? fileName;
  final int? size;

  static String _s(Object? v) => (v ?? '').toString().trim();
  static String? _optS(Object? v) {
    final s = _s(v);
    return s.isEmpty ? null : s;
  }

  static int? _optInt(Object? v) {
    if (v == null) return null;
    if (v is int) return v;
    if (v is num) return v.toInt();
    final s = _s(v);
    return int.tryParse(s);
  }

  factory AvatarIconResponse.fromJson(Map<String, dynamic> json) {
    return AvatarIconResponse(
      id: _s(json['id']),
      avatarId: _optS(json['avatarId']),
      url: _s(json['url']),
      fileName: _optS(json['fileName']),
      size: _optInt(json['size']),
    );
  }
}
