// frontend/mall/lib/features/auth/application/avatar_create_service.dart
import 'package:flutter/foundation.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;
import 'package:web/web.dart' as web;

import '../../avatar/infrastructure/repository_http.dart';
import '../../../di/container.dart';

class PickIconResult {
  const PickIconResult({
    required this.bytes,
    required this.fileName,
    required this.mimeType,
    this.error,
  });

  final Uint8List? bytes;
  final String? fileName;
  final String? mimeType;

  /// エラーメッセージ（成功時は null）
  final String? error;
}

class AvatarCreateResult {
  AvatarCreateResult({
    required this.ok,
    required this.message,
    this.nextRoute,
    this.createdAvatarId,
  });

  final bool ok;
  final String message;
  final String? nextRoute;
  final String? createdAvatarId;
}

class AvatarCreateService {
  AvatarCreateService({
    AvatarRepositoryHttp? repo,
    FirebaseAuth? auth,
    this.logger,
  }) : _repo = repo ?? AppContainer.I.avatarRepositoryHttp,
       _auth = auth ?? FirebaseAuth.instance;

  final AvatarRepositoryHttp _repo;
  final FirebaseAuth _auth;

  final void Function(String s)? logger;

  /// ✅ DI 管理の repo を使うため、ここで dispose しない（AppContainer に任せる）
  void dispose() {}

  void _log(String s) => logger?.call(s);

  String s(String? v) => (v ?? '').trim();

  String backTo(String? from) {
    final f = s(from);
    if (f.isNotEmpty) return f;
    return '/billing-address';
  }

  bool isValidUrlOrEmpty(String s0) {
    final v = s(s0);
    if (v.isEmpty) return true;
    final uri = Uri.tryParse(v);
    if (uri == null) return false;
    if (!uri.hasScheme) return false;
    if (uri.scheme != 'http' && uri.scheme != 'https') return false;
    return uri.host.isNotEmpty;
  }

  // ============================
  // Icon picker (Web)
  // ============================

  Future<PickIconResult?> pickIconWeb() async {
    if (!kIsWeb) {
      return const PickIconResult(
        bytes: null,
        fileName: null,
        mimeType: null,
        error: 'このビルドでは画像選択が未対応です（Web で利用してください）。',
      );
    }

    String? objectUrl;

    try {
      final input = web.HTMLInputElement()
        ..type = 'file'
        ..accept = 'image/*'
        ..multiple = false;

      input.click();
      await input.onChange.first;

      final files = input.files;
      if (files == null || files.length == 0) return null;

      final f = files.item(0);
      if (f == null) return null;

      objectUrl = web.URL.createObjectURL(f);

      final res = await http.get(Uri.parse(objectUrl));
      if (res.statusCode < 200 || res.statusCode >= 300) {
        return PickIconResult(
          bytes: null,
          fileName: null,
          mimeType: null,
          error: '画像の取得に失敗しました: status=${res.statusCode}',
        );
      }

      final bytes = res.bodyBytes;
      if (bytes.isEmpty) {
        return const PickIconResult(
          bytes: null,
          fileName: null,
          mimeType: null,
          error: '画像の読み込みに失敗しました（bytes が空です）。',
        );
      }

      final nameStr = s(f.name);
      final typeStr = s(f.type);

      final name = nameStr.isEmpty ? null : nameStr;
      final mime = typeStr.isEmpty ? null : typeStr;

      return PickIconResult(bytes: bytes, fileName: name, mimeType: mime);
    } catch (e) {
      return PickIconResult(
        bytes: null,
        fileName: null,
        mimeType: null,
        error: '画像の選択に失敗しました: $e',
      );
    } finally {
      if (objectUrl != null) {
        try {
          web.URL.revokeObjectURL(objectUrl);
        } catch (_) {}
      }
    }
  }

  // ============================
  // Save (create avatar + upload icon + PATCH avatarIcon=https://...)
  // ============================

  String _extFromMime(String mime) {
    final m = s(mime).toLowerCase();
    switch (m) {
      case 'image/png':
        return '.png';
      case 'image/jpeg':
      case 'image/jpg':
        return '.jpg';
      case 'image/webp':
        return '.webp';
      case 'image/gif':
        return '.gif';
      default:
        return '';
    }
  }

  String _ensureFileName(String? name, String? mimeType) {
    final n = s(name);
    if (n.isNotEmpty) return n;

    final ext = _extFromMime(s(mimeType));
    final ts = DateTime.now().toUtc().millisecondsSinceEpoch;
    return ext.isEmpty ? 'avatar_$ts' : 'avatar_$ts$ext';
  }

  String _ensureMimeType(String? mimeType, String fileName) {
    final m = s(mimeType);
    if (m.isNotEmpty) return m;

    final lower = fileName.toLowerCase();
    if (lower.endsWith('.png')) return 'image/png';
    if (lower.endsWith('.jpg') || lower.endsWith('.jpeg')) return 'image/jpeg';
    if (lower.endsWith('.webp')) return 'image/webp';
    if (lower.endsWith('.gif')) return 'image/gif';
    return 'application/octet-stream';
  }

  Future<void> _syncAuthDisplayName({
    required User user,
    required String avatarName,
  }) async {
    try {
      final name = s(avatarName);
      if (name.isNotEmpty && s(user.displayName) != name) {
        await user.updateDisplayName(name);
        _log('auth profile updated displayName="$name"');
      }
      await user.reload();
      _log('auth profile reload done');
    } catch (e) {
      _log('auth profile sync skipped err=$e');
    }
  }

  Future<AvatarCreateResult> save({
    required String avatarNameRaw,
    required String profileRaw,
    required String externalLinkRaw,
    Uint8List? iconBytes,
    String? iconFileName,
    String? iconMimeType,
  }) async {
    try {
      final user = _auth.currentUser;
      if (user == null) {
        return AvatarCreateResult(ok: false, message: 'サインインが必要です。');
      }

      final userId = user.uid.trim();
      if (userId.isEmpty) {
        return AvatarCreateResult(ok: false, message: 'userId が取得できませんでした。');
      }

      final avatarName = s(avatarNameRaw);
      if (avatarName.isEmpty) {
        return AvatarCreateResult(ok: false, message: 'アバター名を入力してください。');
      }

      final link = s(externalLinkRaw);
      if (!isValidUrlOrEmpty(link)) {
        return AvatarCreateResult(
          ok: false,
          message: '外部リンクは http(s) のURLを入力してください。',
        );
      }

      final profile = s(profileRaw);

      final bytes = iconBytes;
      final hasIcon = bytes != null && bytes.isNotEmpty;

      final fileName = _ensureFileName(iconFileName, iconMimeType);
      final mimeType = _ensureMimeType(iconMimeType, fileName);
      final size = hasIcon ? bytes.lengthInBytes : 0;

      _log(
        'avatar save start userId=$userId avatarName="$avatarName" '
        'profileLen=${profile.length} externalLink="${link.isEmpty ? "-" : link}" '
        'hasIcon=$hasIcon iconBytesLen=$size file="$fileName" mime="$mimeType"',
      );

      final created = await _repo.create(
        request: CreateAvatarRequest(
          userId: userId,
          avatarName: avatarName,
          avatarIcon: null,
          profile: profile.isEmpty ? null : profile,
          externalLink: link.isEmpty ? null : link,
        ),
      );

      final avatarId = s(created.avatarId);
      if (avatarId.isEmpty) {
        return AvatarCreateResult(ok: false, message: 'avatarId が取得できませんでした。');
      }

      _log('avatar create ok avatarId=$avatarId userId=$userId');

      await _syncAuthDisplayName(user: user, avatarName: avatarName);

      if (hasIcon) {
        final signed = await _repo.issueAvatarIconUploadUrl(
          avatarId: avatarId,
          fileName: fileName,
          mimeType: mimeType,
          size: size,
        );

        final uploadUrl = s(signed.uploadUrl);
        final bucket = s(signed.bucket);
        final objectPath = s(signed.objectPath);

        if (uploadUrl.isEmpty || bucket.isEmpty || objectPath.isEmpty) {
          return AvatarCreateResult(
            ok: false,
            message: 'アイコンアップロードURLの取得に失敗しました。',
            createdAvatarId: avatarId,
          );
        }

        await _repo.uploadToSignedUrl(
          uploadUrl: uploadUrl,
          bytes: bytes,
          contentType: mimeType,
        );

        final httpsUrl = _repo.publicHttpsUrlFromBucketObject(
          bucket: bucket,
          objectPath: objectPath,
        );

        if (httpsUrl.isEmpty) {
          return AvatarCreateResult(
            ok: false,
            message: 'アイコンURLの生成に失敗しました。',
            createdAvatarId: avatarId,
          );
        }

        await _repo.update(
          id: avatarId,
          request: UpdateAvatarRequest(avatarIcon: httpsUrl),
        );
      }

      _log('avatar save done avatarId=$avatarId userId=$userId');

      return AvatarCreateResult(
        ok: true,
        message: 'アバターを作成しました。',
        nextRoute: '/',
        createdAvatarId: avatarId,
      );
    } catch (e) {
      _log('avatar save failed err=$e');
      return AvatarCreateResult(ok: false, message: e.toString());
    }
  }
}
