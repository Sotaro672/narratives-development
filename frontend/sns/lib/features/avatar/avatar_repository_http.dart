// frontend/sns/lib/features/avatar/avatar_repository_http.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:firebase_auth/firebase_auth.dart';

/// ✅ API_BASE に統一（billingAddress と同じ）
const String _fallbackBaseUrl =
    'https://narratives-backend-871263659099.asia-northeast1.run.app';

String _resolveApiBase({String? override}) {
  final o = (override ?? '').trim();
  if (o.isNotEmpty) return _normalizeBase(o);

  const v1 = String.fromEnvironment('API_BASE', defaultValue: '');
  final s1 = v1.trim();
  if (s1.isNotEmpty) return _normalizeBase(s1);

  const v2 = String.fromEnvironment('API_BASE_URL', defaultValue: '');
  final s2 = v2.trim();
  if (s2.isNotEmpty) return _normalizeBase(s2);

  return _normalizeBase(_fallbackBaseUrl);
}

String _normalizeBase(String base) {
  final b = base.trim();
  if (b.endsWith('/')) return b.substring(0, b.length - 1);
  return b;
}

/// Simple HTTP repository for SNS avatar endpoints.
///
/// Back-end handler spec (sns):
/// - POST   /sns/avatars
/// - PATCH  /sns/avatars/{id}
/// - DELETE /sns/avatars/{id}
/// - GET    /sns/avatars/{id}
/// - GET    /sns/avatars/{id}?aggregate=1|true  (Avatar + State + Icons)
/// - POST   /sns/avatars/{id}/wallet            (open wallet)
/// - POST   /sns/avatars/{id}/icon              (register/replace icon)
///
/// ✅ SignedURL (B案 /avatars 配下に寄せる):
/// - POST   /sns/avatars/{id}/icon-upload-url   (issue signed upload url)
///
/// NOTE:
/// - Authorization: Firebase ID token (Bearer)
class AvatarRepositoryHttp {
  AvatarRepositoryHttp({
    http.Client? client,
    FirebaseAuth? auth,
    String? baseUrl,
  }) : _client = client ?? http.Client(),
       _auth = auth ?? FirebaseAuth.instance,
       _base = _resolveApiBase(override: baseUrl) {
    if (_base.trim().isEmpty) {
      throw Exception(
        'API_BASE is not set (use --dart-define=API_BASE=https://...)',
      );
    }
    _log('[AvatarRepositoryHttp] init baseUrl=$_base');
  }

  final http.Client _client;
  final FirebaseAuth _auth;
  final String _base;

  // ✅ release でもログを出したい場合: --dart-define=ENABLE_HTTP_LOG=true
  static const bool _envHttpLog = bool.fromEnvironment(
    'ENABLE_HTTP_LOG',
    defaultValue: false,
  );

  bool get _logEnabled => kDebugMode || _envHttpLog;

  Uri _uri(String path, [Map<String, String>? query]) {
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$_base$p').replace(queryParameters: query);
  }

  Future<Map<String, String>> _authHeaders() async {
    final headers = <String, String>{'Accept': 'application/json'};

    try {
      final u = _auth.currentUser;
      if (u != null) {
        final token = await u.getIdToken();
        headers['Authorization'] = 'Bearer $token';
      }
    } catch (e) {
      _log('[AvatarRepositoryHttp] token error: $e');
    }

    return headers;
  }

  // ---------------------------------------------------------------------------
  // Public API
  // ---------------------------------------------------------------------------

  /// GET /sns/avatars/{id}
  Future<AvatarDTO> getById({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    final uri = _uri('/sns/avatars/$rid');

    final headers = await _authHeaders();
    _logRequest('GET', uri, headers: headers, payload: null);

    final res = await _client.get(uri, headers: headers);
    _logResponse('GET', uri, res.statusCode, res.body);

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    final decoded = _decodeObject(body);
    return AvatarDTO.fromJson(decoded);
  }

  /// GET /sns/avatars/{id}?aggregate=1
  Future<AvatarAggregateDTO> getAggregate({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    final uri = _uri('/sns/avatars/$rid', const {'aggregate': '1'});

    final headers = await _authHeaders();
    _logRequest('GET', uri, headers: headers, payload: null);

    final res = await _client.get(uri, headers: headers);
    _logResponse('GET', uri, res.statusCode, res.body);

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    final decoded = _decodeObject(body);
    return AvatarAggregateDTO.fromJson(decoded);
  }

  /// POST /sns/avatars
  Future<AvatarDTO> create({required CreateAvatarRequest request}) async {
    final uri = _uri('/sns/avatars');

    final headers = await _authHeaders();
    headers['Content-Type'] = 'application/json';

    final payload = request.toJson();
    _logRequest('POST', uri, headers: headers, payload: payload);

    final res = await _client.post(
      uri,
      headers: headers,
      body: jsonEncode(payload),
    );

    _logResponse('POST', uri, res.statusCode, res.body);

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    final decoded = _decodeObject(body);
    return AvatarDTO.fromJson(decoded);
  }

  // ---------------------------------------------------------------------------
  // ✅ NEW: Signed upload URL for avatar icon (B案)
  // ---------------------------------------------------------------------------

  /// POST /sns/avatars/{id}/icon-upload-url
  ///
  /// body:
  /// {
  ///   "fileName": "xxx.png",
  ///   "mimeType": "image/png",
  ///   "size": 12345
  /// }
  ///
  /// response (example):
  /// {
  ///   "uploadUrl": "https://storage.googleapis.com/....signed....",
  ///   "bucket": "narratives-development_avatar_icon",
  ///   "expiresAt": "2025-01-01T00:00:00Z"
  /// }
  Future<AvatarIconUploadUrlDTO> issueAvatarIconUploadUrl({
    required String avatarId,
    required String fileName,
    required String mimeType,
    required int size,
  }) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is empty');

    final fn = fileName.trim();
    if (fn.isEmpty) throw ArgumentError('fileName is empty');

    final mt = mimeType.trim();
    if (mt.isEmpty) throw ArgumentError('mimeType is empty');

    final uri = _uri('/sns/avatars/$aid/icon-upload-url');

    final headers = await _authHeaders();
    headers['Content-Type'] = 'application/json';

    final payload = <String, dynamic>{
      'fileName': fn,
      'mimeType': mt,
      'size': size,
    };

    _logRequest('POST', uri, headers: headers, payload: payload);

    final res = await _client.post(
      uri,
      headers: headers,
      body: jsonEncode(payload),
    );

    _logResponse('POST', uri, res.statusCode, res.body);

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    final decoded = _decodeObject(body);
    return AvatarIconUploadUrlDTO.fromJson(decoded);
  }

  /// PUT to signed URL (upload icon bytes)
  ///
  /// - Signed URL は Authorization 不要（署名が認証）
  /// - headers は Content-Type を合わせる（署名と一致しないと 403 になる）
  Future<void> uploadToSignedUrl({
    required String uploadUrl,
    required Uint8List bytes,
    required String contentType,
  }) async {
    final u = uploadUrl.trim();
    if (u.isEmpty) throw ArgumentError('uploadUrl is empty');
    if (bytes.isEmpty) throw ArgumentError('bytes is empty');

    final ct = contentType.trim().isEmpty
        ? 'application/octet-stream'
        : contentType.trim();

    final uri = Uri.parse(u);

    // ✅ 署名付きURLは OAuth ヘッダ等を付けない
    final headers = <String, String>{'Content-Type': ct};

    _logRequest(
      'PUT',
      uri,
      headers: headers,
      payload: {'bytes': bytes.lengthInBytes, 'contentType': ct},
    );

    final res = await _client.put(uri, headers: headers, body: bytes);

    _logResponse('PUT', uri, res.statusCode, res.body);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: 'upload failed (status=${res.statusCode})',
        url: uri.toString(),
      );
    }
  }

  /// POST /sns/avatars/{id}/wallet
  ///
  /// ✅ Open wallet for existing avatar (server will set walletAddress).
  Future<AvatarDTO> openWallet({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    final uri = _uri('/sns/avatars/$rid/wallet');

    final headers = await _authHeaders();
    headers['Content-Type'] = 'application/json';

    _logRequest('POST', uri, headers: headers, payload: const {});

    final res = await _client.post(
      uri,
      headers: headers,
      body: jsonEncode(const <String, dynamic>{}),
    );

    _logResponse('POST', uri, res.statusCode, res.body);

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    if (body.trim().isEmpty) {
      return getById(id: rid);
    }

    final decoded = _decodeObject(body);
    return AvatarDTO.fromJson(decoded);
  }

  /// POST /sns/avatars/{id}/icon
  ///
  /// ✅ 事前に GCS にアップロード済みの object を AvatarIcon として登録（置換）
  Future<AvatarIconDTO> replaceAvatarIcon({
    required String avatarId,
    String? bucket,
    String? objectPath,
    String? fileName,
    int? size,
    String? avatarIcon,
  }) async {
    final rid = avatarId.trim();
    if (rid.isEmpty) throw ArgumentError('avatarId is empty');

    final uri = _uri('/sns/avatars/$rid/icon');

    final headers = await _authHeaders();
    headers['Content-Type'] = 'application/json';

    final payload = <String, dynamic>{};

    final b = (bucket ?? '').trim();
    if (b.isNotEmpty) payload['bucket'] = b;

    final op = (objectPath ?? '').trim();
    if (op.isNotEmpty) payload['objectPath'] = op;

    final fn = (fileName ?? '').trim();
    if (fn.isNotEmpty) payload['fileName'] = fn;

    if (size != null) payload['size'] = size;

    final ai = (avatarIcon ?? '').trim();
    if (ai.isNotEmpty) payload['avatarIcon'] = ai;

    _logRequest('POST', uri, headers: headers, payload: payload);

    final res = await _client.post(
      uri,
      headers: headers,
      body: jsonEncode(payload),
    );

    _logResponse('POST', uri, res.statusCode, res.body);

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    if (body.trim().isEmpty) {
      throw const FormatException(
        'Empty response body (expected AvatarIconDTO)',
      );
    }

    final decoded = _decodeObject(body);
    return AvatarIconDTO.fromJson(decoded);
  }

  /// PATCH /sns/avatars/{id}
  ///
  /// backend が upsert 的な挙動をして 200/201 でもOK。
  /// body が空なら GET で取り直す（将来の 204/空返却対策）。
  Future<AvatarDTO> update({
    required String id,
    required UpdateAvatarRequest request,
  }) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    final uri = _uri('/sns/avatars/$rid');

    final headers = await _authHeaders();
    headers['Content-Type'] = 'application/json';

    final payload = request.toJson();
    _logRequest('PATCH', uri, headers: headers, payload: payload);

    final res = await _client.patch(
      uri,
      headers: headers,
      body: jsonEncode(payload),
    );

    _logResponse('PATCH', uri, res.statusCode, res.body);

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    if (body.trim().isEmpty) {
      return getById(id: rid);
    }

    final decoded = _decodeObject(body);
    return AvatarDTO.fromJson(decoded);
  }

  /// DELETE /sns/avatars/{id}
  Future<void> delete({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    final uri = _uri('/sns/avatars/$rid');

    final headers = await _authHeaders();
    _logRequest('DELETE', uri, headers: headers, payload: null);

    final res = await _client.delete(uri, headers: headers);

    _logResponse('DELETE', uri, res.statusCode, res.body);

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }
  }

  void dispose() {
    _client.close();
  }

  // ---------------------------------------------------------------------------
  // helpers
  // ---------------------------------------------------------------------------

  Map<String, dynamic> _decodeObject(String body) {
    final b = body.trim();
    if (b.isEmpty) {
      throw const FormatException('Empty response body (expected object)');
    }
    final decoded = jsonDecode(b);
    if (decoded is Map<String, dynamic>) return decoded;
    if (decoded is Map) return Map<String, dynamic>.from(decoded);
    throw const FormatException('Invalid JSON shape (expected object)');
  }

  String? _extractError(String body) {
    try {
      final decoded = jsonDecode(body);
      if (decoded is Map && decoded['error'] != null) {
        return decoded['error'].toString();
      }
    } catch (_) {
      // ignore
    }
    return null;
  }

  // ---------------------------------------------------------------------------
  // Logging (debug or ENABLE_HTTP_LOG=true)
  // ---------------------------------------------------------------------------

  void _log(String msg) {
    if (!_logEnabled) return;
    debugPrint(msg);
  }

  void _logRequest(
    String method,
    Uri uri, {
    required Map<String, String> headers,
    required Map<String, dynamic>? payload,
  }) {
    if (!_logEnabled) return;

    // Authorization は伏せる
    final safeHeaders = <String, String>{};
    headers.forEach((k, v) {
      if (k.toLowerCase() == 'authorization') {
        safeHeaders[k] = 'Bearer ***';
      } else {
        safeHeaders[k] = v;
      }
    });

    final b = StringBuffer();
    b.writeln('[AvatarRepositoryHttp] request');
    b.writeln('  method=$method');
    b.writeln('  url=$uri');
    b.writeln('  headers=${jsonEncode(safeHeaders)}');
    if (payload != null) {
      b.writeln('  payload=${_truncate(jsonEncode(payload), 1500)}');
    }
    debugPrint(b.toString());
  }

  void _logResponse(String method, Uri uri, int status, String body) {
    if (!_logEnabled) return;

    final truncated = _truncate(body, 1500);
    final b = StringBuffer();
    b.writeln('[AvatarRepositoryHttp] response');
    b.writeln('  method=$method');
    b.writeln('  url=$uri');
    b.writeln('  status=$status');
    if (truncated.isNotEmpty) {
      b.writeln('  body=$truncated');
    }
    debugPrint(b.toString());
  }

  String _truncate(String s, int max) {
    final t = s.trim();
    if (t.length <= max) return t;
    return '${t.substring(0, max)}...(truncated ${t.length - max} chars)';
  }
}

// -----------------------------------------------------------------------------
// DTOs / Requests
// -----------------------------------------------------------------------------

String _s(dynamic v) => (v ?? '').toString().trim();

DateTime _parseDateTime(dynamic v) {
  final s = _s(v);
  if (s.isEmpty) return DateTime.fromMillisecondsSinceEpoch(0, isUtc: true);
  return DateTime.parse(s).toUtc();
}

String? _optS(dynamic v) {
  final s = _s(v);
  if (s.isEmpty) return null;
  return s;
}

@immutable
class AvatarIconUploadUrlDTO {
  const AvatarIconUploadUrlDTO({
    required this.uploadUrl,
    required this.bucket,
    required this.objectPath,
    required this.gsUrl,
    required this.expiresAt,
  });

  /// PUT 先の署名付きURL
  final String uploadUrl;

  /// GCS bucket
  final String bucket;

  /// GCS object path
  final String objectPath;

  /// gs://bucket/object
  final String gsUrl;

  /// RFC3339 (UTC)
  final String expiresAt;

  factory AvatarIconUploadUrlDTO.fromJson(Map<String, dynamic> json) {
    dynamic pickAny(List<String> keys) {
      for (final k in keys) {
        if (json.containsKey(k)) return json[k];
      }
      return null;
    }

    final uploadUrl = _s(
      pickAny(const ['uploadUrl', 'UploadURL', 'signedUrl', 'SignedUrl']),
    );
    final bucket = _s(pickAny(const ['bucket', 'Bucket']));
    final objectPath = _s(
      pickAny(const ['objectPath', 'ObjectPath', 'path', 'Path']),
    );
    final gsUrl = _s(pickAny(const ['gsUrl', 'GsUrl', 'gsURL', 'GsURL']));
    final expiresAt = _s(pickAny(const ['expiresAt', 'ExpiresAt']));

    return AvatarIconUploadUrlDTO(
      uploadUrl: uploadUrl,
      bucket: bucket,
      objectPath: objectPath,
      gsUrl: gsUrl,
      expiresAt: expiresAt,
    );
  }
}

@immutable
class AvatarDTO {
  const AvatarDTO({
    required this.id,
    required this.userId,
    required this.avatarName,
    required this.avatarIcon,
    required this.profile,
    required this.externalLink,
    required this.walletAddress,
    required this.createdAt,
    required this.updatedAt,
  });

  final String id;
  final String userId;

  final String avatarName;

  /// Optional: URL/gs://path
  final String? avatarIcon;

  final String? profile;
  final String? externalLink;

  /// Optional: Solana public key (base58)
  final String? walletAddress;

  final DateTime createdAt;
  final DateTime updatedAt;

  factory AvatarDTO.fromJson(Map<String, dynamic> json) {
    // 可能性のあるキーを広めに吸収
    String pick(Map<String, dynamic> j, List<String> keys) {
      for (final k in keys) {
        final v = _s(j[k]);
        if (v.isNotEmpty) return v;
      }
      return '';
    }

    dynamic pickAny(Map<String, dynamic> j, List<String> keys) {
      for (final k in keys) {
        if (j.containsKey(k)) return j[k];
      }
      return null;
    }

    final id = pick(json, const ['id', 'ID', 'avatarId', 'AvatarID']);
    final userId = pick(json, const ['userId', 'UserID']);
    final avatarName = pick(json, const ['avatarName', 'AvatarName']);

    final avatarIcon = _optS(pickAny(json, const ['avatarIcon', 'AvatarIcon']));
    final profile = _optS(pickAny(json, const ['profile', 'Profile']));
    final externalLink = _optS(
      pickAny(json, const ['externalLink', 'ExternalLink']),
    );

    final walletAddress = _optS(
      pickAny(json, const ['walletAddress', 'WalletAddress']),
    );

    final createdAt = _parseDateTime(
      pickAny(json, const ['createdAt', 'CreatedAt']),
    );
    final updatedAt = _parseDateTime(
      pickAny(json, const ['updatedAt', 'UpdatedAt']),
    );

    return AvatarDTO(
      id: id,
      userId: userId,
      avatarName: avatarName,
      avatarIcon: avatarIcon,
      profile: profile,
      externalLink: externalLink,
      walletAddress: walletAddress,
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
  }
}

@immutable
class AvatarStateDTO {
  const AvatarStateDTO({
    required this.avatarId,
    required this.lastActiveAt,
    required this.updatedAt,
  });

  final String avatarId;
  final DateTime? lastActiveAt;
  final DateTime? updatedAt;

  factory AvatarStateDTO.fromJson(Map<String, dynamic> json) {
    dynamic pickAny(List<String> keys) {
      for (final k in keys) {
        if (json.containsKey(k)) return json[k];
      }
      return null;
    }

    final avatarId = _s(pickAny(const ['avatarId', 'AvatarID', 'AvatarId']));
    final lastActiveAtRaw = pickAny(const ['lastActiveAt', 'LastActiveAt']);
    final updatedAtRaw = pickAny(const ['updatedAt', 'UpdatedAt']);

    DateTime? parseOpt(dynamic v) {
      final s = _s(v);
      if (s.isEmpty) return null;
      return DateTime.parse(s).toUtc();
    }

    return AvatarStateDTO(
      avatarId: avatarId,
      lastActiveAt: parseOpt(lastActiveAtRaw),
      updatedAt: parseOpt(updatedAtRaw),
    );
  }
}

@immutable
class AvatarIconDTO {
  const AvatarIconDTO({
    required this.id,
    required this.avatarId,
    required this.url,
    required this.fileName,
    required this.size,
  });

  final String id;
  final String? avatarId;
  final String url;
  final String? fileName;
  final int? size;

  factory AvatarIconDTO.fromJson(Map<String, dynamic> json) {
    dynamic pickAny(List<String> keys) {
      for (final k in keys) {
        if (json.containsKey(k)) return json[k];
      }
      return null;
    }

    final id = _s(pickAny(const ['id', 'ID']));
    final avatarId = _optS(pickAny(const ['avatarId', 'AvatarID', 'AvatarId']));
    final url = _s(pickAny(const ['url', 'URL']));
    final fileName = _optS(pickAny(const ['fileName', 'FileName']));
    final sizeRaw = pickAny(const ['size', 'Size']);

    int? parseOptInt(dynamic v) {
      if (v == null) return null;
      if (v is int) return v;
      if (v is double) return v.toInt();
      final s = _s(v);
      if (s.isEmpty) return null;
      return int.tryParse(s);
    }

    return AvatarIconDTO(
      id: id,
      avatarId: avatarId,
      url: url,
      fileName: fileName,
      size: parseOptInt(sizeRaw),
    );
  }
}

@immutable
class AvatarAggregateDTO {
  const AvatarAggregateDTO({
    required this.avatar,
    required this.state,
    required this.icons,
  });

  final AvatarDTO avatar;
  final AvatarStateDTO? state;
  final List<AvatarIconDTO> icons;

  factory AvatarAggregateDTO.fromJson(Map<String, dynamic> json) {
    // Go の json.Encoder はフィールド名がそのまま出ることがある（Avatar/State/Icons）
    dynamic pickAny(List<String> keys) {
      for (final k in keys) {
        if (json.containsKey(k)) return json[k];
      }
      return null;
    }

    final avatarRaw = pickAny(const ['avatar', 'Avatar']);
    final stateRaw = pickAny(const ['state', 'State']);
    final iconsRaw = pickAny(const ['icons', 'Icons']);

    Map<String, dynamic> asObj(dynamic v) {
      if (v is Map<String, dynamic>) return v;
      if (v is Map) return Map<String, dynamic>.from(v);
      return <String, dynamic>{};
    }

    List asList(dynamic v) {
      if (v is List) return v;
      return const [];
    }

    final avatar = AvatarDTO.fromJson(asObj(avatarRaw));

    AvatarStateDTO? state;
    if (stateRaw != null) {
      final obj = asObj(stateRaw);
      if (obj.isNotEmpty) {
        state = AvatarStateDTO.fromJson(obj);
      }
    }

    final icons = <AvatarIconDTO>[];
    for (final it in asList(iconsRaw)) {
      final obj = asObj(it);
      if (obj.isEmpty) continue;
      icons.add(AvatarIconDTO.fromJson(obj));
    }

    return AvatarAggregateDTO(avatar: avatar, state: state, icons: icons);
  }
}

/// POST body
@immutable
class CreateAvatarRequest {
  const CreateAvatarRequest({
    required this.userId,
    required this.userUid,
    required this.avatarName,
    this.avatarIcon,
    this.profile,
    this.externalLink,
  });

  final String userId;

  /// ✅ firebaseUid ではなく userUid
  final String userUid;

  final String avatarName;

  /// Optional: URL/gs://path
  final String? avatarIcon;

  final String? profile;
  final String? externalLink;

  Map<String, dynamic> toJson() {
    final m = <String, dynamic>{
      'userId': userId.trim(),
      'userUid': userUid.trim(),
      'avatarName': avatarName.trim(),
    };

    final icon = (avatarIcon ?? '').trim();
    if (icon.isNotEmpty) m['avatarIcon'] = icon;

    final prof = (profile ?? '').trim();
    if (prof.isNotEmpty) m['profile'] = prof;

    final link = (externalLink ?? '').trim();
    if (link.isNotEmpty) m['externalLink'] = link;

    return m;
  }
}

/// PATCH body (partial update)
///
/// backend 側は trimPtrNilAware で以下のように解釈する想定:
/// - key が無い: 更新しない
/// - 値が "" : clear (nil 保存)
/// - 値が "x": 更新
@immutable
class UpdateAvatarRequest {
  const UpdateAvatarRequest({
    this.avatarName,
    this.avatarIcon,
    this.profile,
    this.externalLink,
    this.walletAddress,
  });

  final String? avatarName;
  final String? avatarIcon;
  final String? profile;
  final String? externalLink;

  /// Optional: allow patch walletAddress if backend supports it (usually opened by /wallet)
  final String? walletAddress;

  Map<String, dynamic> toJson() {
    final m = <String, dynamic>{};

    if (avatarName != null) m['avatarName'] = avatarName!.trim();

    // clear したい場合は "" を送る（= trim で "" になっても送る）
    if (avatarIcon != null) m['avatarIcon'] = avatarIcon!.trim();
    if (profile != null) m['profile'] = profile!.trim();
    if (externalLink != null) m['externalLink'] = externalLink!.trim();

    if (walletAddress != null) m['walletAddress'] = walletAddress!.trim();

    return m;
  }
}

@immutable
class HttpException implements Exception {
  const HttpException({
    required this.statusCode,
    required this.message,
    required this.url,
  });

  final int statusCode;
  final String message;
  final String url;

  @override
  String toString() => 'HttpException($statusCode) $message ($url)';
}
