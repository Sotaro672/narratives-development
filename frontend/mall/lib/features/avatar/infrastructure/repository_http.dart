// frontend/mall/lib/features/avatar/infrastructure/avatar_repository_http.dart
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:firebase_auth/firebase_auth.dart';

import 'api.dart';

/// Simple HTTP repository for Mall avatar endpoints.
///
/// Back-end handler spec (mall):
/// - POST   /mall/avatars
/// - PATCH  /mall/avatars/{id}
/// - DELETE /mall/avatars/{id}
/// - GET    /mall/avatars/{id}
/// - GET    /mall/avatars/{id}?aggregate=1|true  (Avatar + State + Icons)
/// - POST   /mall/avatars/{id}/wallet            (open wallet)
/// - POST   /mall/avatars/{id}/icon              (register/replace icon)
///
/// ✅ SignedURL (B案 /avatars 配下に寄せる):
/// - POST   /mall/avatars/{id}/icon-upload-url   (issue signed upload url)
///
/// NOTE (重要):
/// - Firestore 側は「docID = avatarId」「フィールド userId = Firebase uid」を保持する設計
/// - Authorization: Firebase ID token (Bearer)
///
/// ✅ 絶対正スキーマ（Backend 正規キー）:
/// - avatarId
/// - userId
/// - avatarName
/// - avatarIcon (nullable)
/// - profile (nullable)
/// - externalLink (nullable)
/// - walletAddress (nullable) ※更新は /wallet のみ（PATCHでは禁止）
class AvatarRepositoryHttp {
  AvatarRepositoryHttp({
    http.Client? client,
    FirebaseAuth? auth,
    String? baseUrl,
  }) : _api = MallAuthedApi(client: client, auth: auth, baseUrl: baseUrl);

  final MallAuthedApi _api;

  // ---------------------------------------------------------------------------
  // Public API
  // ---------------------------------------------------------------------------

  /// GET /mall/avatars/{id}
  Future<AvatarDTO> getById({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    final uri = _api.uri('/mall/avatars/$rid');
    final res = await _api.sendAuthed('GET', uri);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      _api.throwHttpError(res, uri);
    }

    final decoded = _api.unwrapData(_api.decodeObject(res.body));
    return AvatarDTO.fromJson(decoded);
  }

  /// GET /mall/avatars/{id}?aggregate=1
  Future<AvatarAggregateDTO> getAggregate({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    final uri = _api.uri('/mall/avatars/$rid', const {'aggregate': '1'});
    final res = await _api.sendAuthed('GET', uri);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      _api.throwHttpError(res, uri);
    }

    final decoded = _api.unwrapData(_api.decodeObject(res.body));
    return AvatarAggregateDTO.fromJson(decoded);
  }

  /// POST /mall/avatars
  ///
  /// NOTE:
  /// - Firestore は docID を NewDoc() で採番する（avatarId が空の場合）
  /// - userId はフィールドとして保存される（= Firebase uid を入れる想定）
  Future<AvatarDTO> create({required CreateAvatarRequest request}) async {
    final uri = _api.uri('/mall/avatars');
    final payload = request.toJson();

    final res = await _api.sendAuthed('POST', uri, jsonBody: payload);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      _api.throwHttpError(res, uri);
    }

    // body が空なら GET で取り直すのが理想だが、avatarId が無いのでここでは例外
    final decoded = _api.unwrapData(_api.decodeObject(res.body));
    return AvatarDTO.fromJson(decoded);
  }

  // ---------------------------------------------------------------------------
  // ✅ Signed upload URL for avatar icon (B案)
  // ---------------------------------------------------------------------------

  /// POST /mall/avatars/{id}/icon-upload-url
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

    final uri = _api.uri('/mall/avatars/$aid/icon-upload-url');
    final payload = <String, dynamic>{
      'fileName': fn,
      'mimeType': mt,
      'size': size,
    };

    final res = await _api.sendAuthed('POST', uri, jsonBody: payload);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      _api.throwHttpError(res, uri);
    }

    final decoded = _api.unwrapData(_api.decodeObject(res.body));
    return AvatarIconUploadUrlDTO.fromJson(decoded);
  }

  /// PUT to signed URL (upload icon bytes)
  Future<void> uploadToSignedUrl({
    required String uploadUrl,
    required Uint8List bytes,
    required String contentType,
  }) => _api.uploadToSignedUrl(
    uploadUrl: uploadUrl,
    bytes: bytes,
    contentType: contentType,
  );

  /// POST /mall/avatars/{id}/wallet
  ///
  /// ✅ Open wallet for existing avatar (server will set walletAddress).
  Future<AvatarDTO> openWallet({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    final uri = _api.uri('/mall/avatars/$rid/wallet');

    final res = await _api.sendAuthed(
      'POST',
      uri,
      jsonBody: const <String, dynamic>{},
      allowEmptyBody: true,
    );

    if (res.statusCode < 200 || res.statusCode >= 300) {
      _api.throwHttpError(res, uri);
    }

    if (res.body.trim().isEmpty) {
      return getById(id: rid);
    }

    final decoded = _api.unwrapData(_api.decodeObject(res.body));
    return AvatarDTO.fromJson(decoded);
  }

  /// POST /mall/avatars/{id}/icon
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

    final uri = _api.uri('/mall/avatars/$rid/icon');

    final payload = <String, dynamic>{};

    final b = (bucket ?? '').trim();
    if (b.isNotEmpty) payload['bucket'] = b;

    final op = (objectPath ?? '').trim();
    if (op.isNotEmpty) payload['objectPath'] = op;

    final fn = (fileName ?? '').trim();
    if (fn.isNotEmpty) payload['fileName'] = fn;

    if (size != null) payload['size'] = size;

    // 互換入力として受ける（backend が gs://... を parse できる想定）
    final ai = (avatarIcon ?? '').trim();
    if (ai.isNotEmpty) payload['avatarIcon'] = ai;

    final res = await _api.sendAuthed('POST', uri, jsonBody: payload);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      _api.throwHttpError(res, uri);
    }

    if (res.body.trim().isEmpty) {
      throw const FormatException(
        'Empty response body (expected AvatarIconDTO)',
      );
    }

    final decoded = _api.unwrapData(_api.decodeObject(res.body));
    return AvatarIconDTO.fromJson(decoded);
  }

  /// PATCH /mall/avatars/{id}
  ///
  /// NOTE:
  /// - walletAddress は backend 側で update 禁止（/wallet でのみ更新）
  Future<AvatarDTO> update({
    required String id,
    required UpdateAvatarRequest request,
  }) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    final uri = _api.uri('/mall/avatars/$rid');
    final payload = request.toJson();

    final res = await _api.sendAuthed('PATCH', uri, jsonBody: payload);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      _api.throwHttpError(res, uri);
    }

    if (res.body.trim().isEmpty) {
      return getById(id: rid);
    }

    final decoded = _api.unwrapData(_api.decodeObject(res.body));
    return AvatarDTO.fromJson(decoded);
  }

  /// DELETE /mall/avatars/{id}
  Future<void> delete({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    final uri = _api.uri('/mall/avatars/$rid');
    final res = await _api.sendAuthed('DELETE', uri);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      _api.throwHttpError(res, uri);
    }
  }

  void dispose() {
    _api.dispose();
  }
}

// -----------------------------------------------------------------------------
// DTOs / Requests (絶対正スキーマ準拠・互換吸収なし)
// -----------------------------------------------------------------------------

String _s(Object? v) => (v ?? '').toString().trim();

String? _optS(Object? v) {
  final s = _s(v);
  return s.isEmpty ? null : s;
}

DateTime _parseDateTime(Object? v) {
  final s = _s(v);
  if (s.isEmpty) return DateTime.fromMillisecondsSinceEpoch(0, isUtc: true);
  return DateTime.parse(s).toUtc();
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
    return AvatarIconUploadUrlDTO(
      uploadUrl: _s(json['uploadUrl']),
      bucket: _s(json['bucket']),
      objectPath: _s(json['objectPath']),
      gsUrl: _s(json['gsUrl']),
      expiresAt: _s(json['expiresAt']),
    );
  }
}

@immutable
class AvatarDTO {
  const AvatarDTO({
    required this.avatarId,
    required this.userId,
    required this.avatarName,
    required this.avatarIcon,
    required this.profile,
    required this.externalLink,
    required this.walletAddress,
    required this.createdAt,
    required this.updatedAt,
  });

  /// Firestore docID（= avatarId）
  final String avatarId;

  /// Firestore field userId（= Firebase uid）
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
    return AvatarDTO(
      avatarId: _s(json['avatarId']),
      userId: _s(json['userId']),
      avatarName: _s(json['avatarName']),
      avatarIcon: _optS(json['avatarIcon']),
      profile: _optS(json['profile']),
      externalLink: _optS(json['externalLink']),
      walletAddress: _optS(json['walletAddress']),
      createdAt: _parseDateTime(json['createdAt']),
      updatedAt: _parseDateTime(json['updatedAt']),
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
    DateTime? parseOpt(Object? v) {
      final s = _s(v);
      if (s.isEmpty) return null;
      return DateTime.parse(s).toUtc();
    }

    return AvatarStateDTO(
      avatarId: _s(json['avatarId']),
      lastActiveAt: parseOpt(json['lastActiveAt']),
      updatedAt: parseOpt(json['updatedAt']),
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
    int? parseOptInt(Object? v) {
      if (v == null) return null;
      if (v is int) return v;
      if (v is double) return v.toInt();
      final s = _s(v);
      if (s.isEmpty) return null;
      return int.tryParse(s);
    }

    return AvatarIconDTO(
      id: _s(json['id']),
      avatarId: _optS(json['avatarId']),
      url: _s(json['url']),
      fileName: _optS(json['fileName']),
      size: parseOptInt(json['size']),
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
    Map<String, dynamic> asObj(Object? v) {
      if (v is Map<String, dynamic>) return v;
      if (v is Map) return Map<String, dynamic>.from(v);
      return <String, dynamic>{};
    }

    List asList(Object? v) {
      if (v is List) return v;
      return const [];
    }

    final avatarRaw = json['avatar'];
    final stateRaw = json['state'];
    final iconsRaw = json['icons'];

    final avatar = AvatarDTO.fromJson(asObj(avatarRaw));

    AvatarStateDTO? state;
    if (stateRaw != null) {
      final obj = asObj(stateRaw);
      if (obj.isNotEmpty) state = AvatarStateDTO.fromJson(obj);
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

/// POST body（絶対正スキーマ）
@immutable
class CreateAvatarRequest {
  const CreateAvatarRequest({
    required this.userId,
    required this.avatarName,
    this.avatarIcon,
    this.profile,
    this.externalLink,
  });

  /// Firestore field userId（Firebase uid）
  final String userId;

  final String avatarName;

  /// Optional: URL/gs://path
  final String? avatarIcon;

  final String? profile;
  final String? externalLink;

  Map<String, dynamic> toJson() {
    final m = <String, dynamic>{
      'userId': userId.trim(),
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
///
/// ✅ 絶対正: walletAddress は PATCH では送らない（/wallet のみ）
@immutable
class UpdateAvatarRequest {
  const UpdateAvatarRequest({
    this.avatarName,
    this.avatarIcon,
    this.profile,
    this.externalLink,
  });

  final String? avatarName;
  final String? avatarIcon;
  final String? profile;
  final String? externalLink;

  Map<String, dynamic> toJson() {
    final m = <String, dynamic>{};

    if (avatarName != null) m['avatarName'] = avatarName!.trim();

    // clear したい場合は "" を送る（= trim で "" になっても送る）
    if (avatarIcon != null) m['avatarIcon'] = avatarIcon!.trim();
    if (profile != null) m['profile'] = profile!.trim();
    if (externalLink != null) m['externalLink'] = externalLink!.trim();

    return m;
  }
}
