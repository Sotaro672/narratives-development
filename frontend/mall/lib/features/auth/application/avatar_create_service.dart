// frontend/mall/lib/features/auth/application/avatar_create_service.dart
import 'package:flutter/foundation.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;
import 'package:web/web.dart' as web;

import '../../avatar/infrastructure/avatar_api_client.dart';

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
  AvatarCreateService({AvatarApiClient? api, FirebaseAuth? auth, this.logger})
    : _api = api ?? AvatarApiClient(),
      _auth = auth ?? FirebaseAuth.instance;

  final AvatarApiClient _api;
  final FirebaseAuth _auth;

  final void Function(String s)? logger;

  void dispose() {
    _api.dispose();
  }

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
        } catch (_) {
          // ignore
        }
      }
    }
  }

  // ============================
  // Save (create avatar + upload icon + register icon)
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

  /// ✅ FirebaseAuth の displayName を同期（photoURL は絶対正スキーマ外なので扱わない）
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

      // ✅ 認証主体 UID（absolute schema の userId）
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

      // (1) ✅ Avatar を作成（icon は別フローで確定）
      final created = await _api.createAvatar({
        'userId': userId,
        'avatarName': avatarName,
        if (profile.isNotEmpty) 'profile': profile,
        if (link.isNotEmpty) 'externalLink': link,
      });

      final avatarId = created.avatarId.trim();
      if (avatarId.isEmpty) {
        return AvatarCreateResult(ok: false, message: 'avatarId が取得できませんでした。');
      }

      _log('avatar create ok avatarId=$avatarId userId=$userId');

      await _syncAuthDisplayName(user: user, avatarName: avatarName);

      // (2) ✅ アイコンがあれば：署名付きURL→PUT→register
      // 方針:
      // - publicUrl(=avatarIcon) の生成は backend(usecase) が担う
      // - bucket/gsUrl は backend が返す
      // - クライアントは https URL 合成も me PATCH も行わない
      if (hasIcon) {
        final signed = await _api.issueAvatarIconUploadUrl(
          avatarId: avatarId,
          fileName: fileName,
          mimeType: mimeType,
          size: size,
        );

        final uploadUrl = s(signed.uploadUrl);
        final bucket = s(signed.bucket);
        final objectPath = s(signed.objectPath);

        if (uploadUrl.isEmpty || bucket.isEmpty || objectPath.isEmpty) {
          _log(
            'icon signed url invalid avatarId=$avatarId '
            'uploadUrl="$uploadUrl" bucket="$bucket" objectPath="$objectPath"',
          );
          return AvatarCreateResult(
            ok: false,
            message: 'アイコンアップロードURLの取得に失敗しました。',
            createdAvatarId: avatarId,
          );
        }

        final short = uploadUrl.length > 64
            ? '${uploadUrl.substring(0, 64)}...'
            : uploadUrl;

        _log(
          'icon signed url ok avatarId=$avatarId bucket="$bucket" objectPath="$objectPath" '
          'expiresAt="${signed.expiresAt ?? "-"}" uploadUrl="$short"',
        );

        await _api.uploadToSignedUrl(
          uploadUrl: uploadUrl,
          bytes: bytes,
          contentType: mimeType,
        );

        _log('icon upload ok bytes=$size');

        // ✅ register: backend が avatars.avatarIcon を確定（usecase で patch する想定）
        final registered = await _api.registerAvatarIcon(
          avatarId: avatarId,
          bucket: bucket,
          objectPath: objectPath,
          fileName: fileName,
          size: size,
        );

        _log(
          'icon register ok iconId="${registered.id}" url="${registered.url}"',
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
