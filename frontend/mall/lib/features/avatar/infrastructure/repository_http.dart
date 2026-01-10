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
/// - ただし Create API は互換のため userUid も受け取る（現状の handler/usecase が期待）
/// - Authorization: Firebase ID token (Bearer)
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
  /// - Firestore は docID を NewDoc() で採番する（a.ID が空の場合）
  /// - userId はフィールドとして保存される（= Firebase uid を入れる想定）
  /// - userUid は互換/検証のため送る（現状の backend が期待）
  Future<AvatarDTO> create({required CreateAvatarRequest request}) async {
    final uri = _api.uri('/mall/avatars');
    final payload = request.toJson();

    final res = await _api.sendAuthed('POST', uri, jsonBody: payload);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      _api.throwHttpError(res, uri);
    }

    // body が空なら GET で取り直すのが理想だが、id が無いのでここでは例外
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
  /// backend が upsert 的な挙動をして 200/201 でもOK。
  /// body が空なら GET で取り直す（将来の 204/空返却対策）。
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

  /// Firestore docID（= avatarId）
  final String id;

  /// Firestore field userId（= Firebase uid を入れる想定）
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
///
/// NOTE:
/// - backend の現状が userUid を期待しているため送る
/// - userId には Firebase uid を入れる期待値
@immutable
class CreateAvatarRequest {
  const CreateAvatarRequest({
    required this.userId,
    required this.userUid, // ✅ 追加
    required this.avatarName,
    this.avatarIcon,
    this.profile,
    this.externalLink,
  });

  /// Firestore field userId（Firebase uid を入れる想定）
  final String userId;

  /// 互換/検証用（現状 backend が期待）
  final String userUid; // ✅ 追加

  final String avatarName;

  /// Optional: URL/gs://path
  final String? avatarIcon;

  final String? profile;
  final String? externalLink;

  Map<String, dynamic> toJson() {
    final m = <String, dynamic>{
      'userId': userId.trim(),
      'userUid': userUid.trim(), // ✅ 追加
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
