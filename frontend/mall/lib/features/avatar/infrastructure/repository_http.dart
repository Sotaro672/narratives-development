// frontend\mall\lib\features\avatar\infrastructure\avatar_repository_http.dart
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:firebase_auth/firebase_auth.dart';

import 'api.dart';

/// Simple HTTP repository for Mall avatar endpoints.
///
/// ✅ Contract:
/// - POST   /mall/avatars        (create only)
/// - GET    /mall/me/avatar      (read my avatar)
/// - PATCH  /mall/me/avatar      (update my avatar)
///
/// ✅ Absolute schema (Backend 正規キー):
/// - avatarId
/// - userId
/// - avatarName
/// - avatarIcon (nullable)
/// - profile (nullable)
/// - externalLink (nullable)
/// - walletAddress (required in /mall/me/avatar.patch)
class AvatarRepositoryHttp {
  AvatarRepositoryHttp({
    http.Client? client,
    FirebaseAuth? auth,
    String? baseUrl,
  }) : _api = MallAuthedApi(client: client, auth: auth, baseUrl: baseUrl);

  final MallAuthedApi _api;

  // ---------------------------------------------------------------------------
  // ✅ Pattern B: Me endpoints (read/update)
  // ---------------------------------------------------------------------------

  /// GET /mall/me/avatar
  Future<MeAvatarDTO> getMe() async {
    final uri = _api.uri('/mall/me/avatar');

    if (kDebugMode) {
      debugPrint('[AvatarRepositoryHttp] GET $uri');
    }

    final res = await _api.sendAuthed('GET', uri);

    if (kDebugMode) {
      debugPrint(
        '[AvatarRepositoryHttp] GET /mall/me/avatar status=${res.statusCode} bodyLen=${res.body.length}',
      );
      // body 全文は長い/機微になりがちなので、先頭だけ
      final head = res.body.length > 240
          ? res.body.substring(0, 240)
          : res.body;
      debugPrint('[AvatarRepositoryHttp] GET body head=$head');
    }

    if (res.statusCode < 200 || res.statusCode >= 300) {
      _api.throwHttpError(res, uri);
    }

    if (res.body.trim().isEmpty) {
      throw const FormatException('Empty response body (expected MeAvatarDTO)');
    }

    final decoded = _api.unwrapData(_api.decodeObject(res.body));

    if (kDebugMode) {
      // decode 後の形も確認したい（patch/avatarName の所在確認用）
      debugPrint(
        '[AvatarRepositoryHttp] decoded keys=${decoded.keys.toList()}',
      );
      final patch = decoded['patch'];
      if (patch is Map) {
        final p = Map<String, dynamic>.from(patch);
        debugPrint('[AvatarRepositoryHttp] patch keys=${p.keys.toList()}');
        final an = (p['avatarName'] ?? '').toString().trim();
        final prof = (p['profile'] ?? '').toString().trim();
        debugPrint(
          '[AvatarRepositoryHttp] decoded patch.avatarName="${an.isEmpty ? "-" : an}" profileLen=${prof.length}',
        );
      } else {
        debugPrint('[AvatarRepositoryHttp] decoded.patch is not a Map');
      }
    }

    final dto = MeAvatarDTO.fromJson(decoded);

    if (kDebugMode) {
      debugPrint(
        '[AvatarRepositoryHttp] MeAvatarDTO parsed avatarId="${dto.avatarId}" '
        'avatarName="${(dto.avatarName ?? '').trim().isEmpty ? "-" : dto.avatarName}" '
        'profileLen=${(dto.profile ?? '').trim().length} '
        'walletAddressLen=${dto.walletAddress.trim().length}',
      );
    }

    return dto;
  }

  /// PATCH /mall/me/avatar
  ///
  /// Backend may return empty body -> then call getMe() to re-fetch.
  Future<MeAvatarDTO> updateMe({required UpdateMeAvatarRequest request}) async {
    final uri = _api.uri('/mall/me/avatar');
    final payload = request.toJson();

    if (kDebugMode) {
      debugPrint('[AvatarRepositoryHttp] PATCH $uri payload=$payload');
    }

    final res = await _api.sendAuthed(
      'PATCH',
      uri,
      jsonBody: payload,
      allowEmptyBody: true,
    );

    if (kDebugMode) {
      debugPrint(
        '[AvatarRepositoryHttp] PATCH /mall/me/avatar status=${res.statusCode} bodyLen=${res.body.length}',
      );
      final head = res.body.length > 240
          ? res.body.substring(0, 240)
          : res.body;
      debugPrint('[AvatarRepositoryHttp] PATCH body head=$head');
    }

    if (res.statusCode < 200 || res.statusCode >= 300) {
      _api.throwHttpError(res, uri);
    }

    if (res.body.trim().isEmpty) {
      if (kDebugMode) {
        debugPrint(
          '[AvatarRepositoryHttp] PATCH returned empty body -> refetch via getMe()',
        );
      }
      return getMe();
    }

    final decoded = _api.unwrapData(_api.decodeObject(res.body));

    if (kDebugMode) {
      debugPrint(
        '[AvatarRepositoryHttp] PATCH decoded keys=${decoded.keys.toList()}',
      );
      final patch = decoded['patch'];
      if (patch is Map) {
        final p = Map<String, dynamic>.from(patch);
        final an = (p['avatarName'] ?? '').toString().trim();
        debugPrint(
          '[AvatarRepositoryHttp] PATCH decoded patch.avatarName="${an.isEmpty ? "-" : an}"',
        );
      }
    }

    final dto = MeAvatarDTO.fromJson(decoded);

    if (kDebugMode) {
      debugPrint(
        '[AvatarRepositoryHttp] PATCH MeAvatarDTO parsed avatarId="${dto.avatarId}" '
        'avatarName="${(dto.avatarName ?? '').trim().isEmpty ? "-" : dto.avatarName}"',
      );
    }

    return dto;
  }

  // ---------------------------------------------------------------------------
  // ✅ Create only
  // ---------------------------------------------------------------------------

  /// POST /mall/avatars
  ///
  /// ✅ 要件:
  /// - 新規作成のみ /mall/avatars を叩く
  Future<AvatarDTO> create({required CreateAvatarRequest request}) async {
    final uri = _api.uri('/mall/avatars');
    final payload = request.toJson();

    if (kDebugMode) {
      debugPrint('[AvatarRepositoryHttp] POST $uri payload=$payload');
    }

    final res = await _api.sendAuthed(
      'POST',
      uri,
      jsonBody: payload,
      allowEmptyBody: true,
    );

    if (kDebugMode) {
      debugPrint(
        '[AvatarRepositoryHttp] POST /mall/avatars status=${res.statusCode} bodyLen=${res.body.length}',
      );
      final head = res.body.length > 240
          ? res.body.substring(0, 240)
          : res.body;
      debugPrint('[AvatarRepositoryHttp] POST body head=$head');
    }

    if (res.statusCode < 200 || res.statusCode >= 300) {
      _api.throwHttpError(res, uri);
    }

    if (res.body.trim().isEmpty) {
      throw const FormatException('Empty response body (expected AvatarDTO)');
    }

    final decoded = _api.unwrapData(_api.decodeObject(res.body));

    if (kDebugMode) {
      debugPrint(
        '[AvatarRepositoryHttp] POST decoded keys=${decoded.keys.toList()} avatarName="${(decoded['avatarName'] ?? '').toString().trim()}"',
      );
    }

    final dto = AvatarDTO.fromJson(decoded);

    if (kDebugMode) {
      debugPrint(
        '[AvatarRepositoryHttp] AvatarDTO parsed avatarId="${dto.avatarId}" avatarName="${dto.avatarName}"',
      );
    }

    return dto;
  }

  // ---------------------------------------------------------------------------
  // ✅ Signed upload URL for avatar icon (id required by backend)
  // ---------------------------------------------------------------------------

  /// POST /mall/avatars/{id}/icon-upload-url
  ///
  /// NOTE:
  /// - 現状バックエンドが avatarId を要求するならここは維持
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

    if (kDebugMode) {
      debugPrint('[AvatarRepositoryHttp] POST $uri payload=$payload');
    }

    final res = await _api.sendAuthed('POST', uri, jsonBody: payload);

    if (kDebugMode) {
      debugPrint(
        '[AvatarRepositoryHttp] POST icon-upload-url status=${res.statusCode} bodyLen=${res.body.length}',
      );
      final head = res.body.length > 240
          ? res.body.substring(0, 240)
          : res.body;
      debugPrint('[AvatarRepositoryHttp] POST icon-upload-url body head=$head');
    }

    if (res.statusCode < 200 || res.statusCode >= 300) {
      _api.throwHttpError(res, uri);
    }

    final decoded = _api.unwrapData(_api.decodeObject(res.body));
    final dto = AvatarIconUploadUrlDTO.fromJson(decoded);

    if (kDebugMode) {
      final short = dto.uploadUrl.length > 64
          ? '${dto.uploadUrl.substring(0, 64)}...'
          : dto.uploadUrl;
      debugPrint(
        '[AvatarRepositoryHttp] icon-upload-url parsed bucket="${dto.bucket}" objectPath="${dto.objectPath}" uploadUrl="$short"',
      );
    }

    return dto;
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

  /// helper: bucket/objectPath から public https URL を生成
  String publicHttpsUrlFromBucketObject({
    required String bucket,
    required String objectPath,
  }) {
    final b = bucket.trim();
    final op = objectPath.trim();
    if (b.isEmpty || op.isEmpty) return '';
    final encoded = op.split('/').map(Uri.encodeComponent).join('/');
    return 'https://storage.googleapis.com/$b/$encoded';
  }

  /// ワンストップ: 署名URL発行 → PUT → public https URL を返す
  Future<String> uploadAvatarIconAndGetPublicUrl({
    required String avatarId,
    required String fileName,
    required String mimeType,
    required Uint8List bytes,
  }) async {
    final dto = await issueAvatarIconUploadUrl(
      avatarId: avatarId,
      fileName: fileName,
      mimeType: mimeType,
      size: bytes.lengthInBytes,
    );

    await uploadToSignedUrl(
      uploadUrl: dto.uploadUrl,
      bytes: bytes,
      contentType: mimeType,
    );

    final url = publicHttpsUrlFromBucketObject(
      bucket: dto.bucket,
      objectPath: dto.objectPath,
    );

    return url.trim();
  }

  void dispose() {
    _api.dispose();
  }
}

// -----------------------------------------------------------------------------
// DTOs / Requests
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

DateTime? _parseOptDateTime(Object? v) {
  final s = _s(v);
  if (s.isEmpty) return null;
  try {
    return DateTime.parse(s).toUtc();
  } catch (_) {
    return null;
  }
}

/// ✅ DTO for /mall/me/avatar
@immutable
class MeAvatarDTO {
  const MeAvatarDTO({
    required this.avatarId,
    required this.walletAddress,
    this.userId,
    this.avatarName,
    this.avatarIcon,
    this.profile,
    this.externalLink,
    this.deletedAt,
  });

  final String avatarId;
  final String walletAddress;

  final String? userId;
  final String? avatarName;
  final String? avatarIcon;
  final String? profile;
  final String? externalLink;
  final DateTime? deletedAt;

  factory MeAvatarDTO.fromJson(Map<String, dynamic> json) {
    final avatarId = _s(json['avatarId']);

    final p = json['patch'];
    if (p is! Map) {
      throw const FormatException('MeAvatarDTO: missing "patch" object');
    }
    final patch = Map<String, dynamic>.from(p);

    final walletAddress = _s(patch['walletAddress']);
    if (avatarId.isEmpty) {
      throw const FormatException('MeAvatarDTO: avatarId is required');
    }
    if (walletAddress.isEmpty) {
      throw const FormatException(
        'MeAvatarDTO: patch.walletAddress is required',
      );
    }

    return MeAvatarDTO(
      avatarId: avatarId,
      walletAddress: walletAddress,
      userId: _optS(patch['userId']),
      avatarName: _optS(patch['avatarName']),
      avatarIcon: _optS(patch['avatarIcon']),
      profile: _optS(patch['profile']),
      externalLink: _optS(patch['externalLink']),
      deletedAt: _parseOptDateTime(patch['deletedAt']),
    );
  }
}

/// PATCH /mall/me/avatar body
@immutable
class UpdateMeAvatarRequest {
  const UpdateMeAvatarRequest({
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

@immutable
class AvatarIconUploadUrlDTO {
  const AvatarIconUploadUrlDTO({
    required this.uploadUrl,
    required this.bucket,
    required this.objectPath,
    required this.gsUrl,
    required this.expiresAt,
  });

  final String uploadUrl;
  final String bucket;
  final String objectPath;
  final String gsUrl;
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

  final String avatarId;
  final String userId;
  final String avatarName;

  final String? avatarIcon;
  final String? profile;
  final String? externalLink;

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

  final String userId;
  final String avatarName;

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
