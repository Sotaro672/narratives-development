// frontend/mall/lib/features/auth/application/avatar_create_service.dart
import 'package:flutter/foundation.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;
import 'package:web/web.dart' as web;

import '../../avatar/infrastructure/repository_http.dart';

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
  }) : _repo = repo ?? AvatarRepositoryHttp(),
       _auth = auth ?? FirebaseAuth.instance;

  final AvatarRepositoryHttp _repo;
  final FirebaseAuth _auth;

  final void Function(String s)? logger;

  void dispose() {
    _repo.dispose();
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

      // ✅ web.FileList には isEmpty が無いので length で判定する
      if (files == null || files.length == 0) return null;

      final f = files.item(0);
      if (f == null) return null;

      // ✅ ブラウザに一時URLを作らせる
      objectUrl = web.URL.createObjectURL(f);

      // ✅ そのURLを HTTP GET して bytes を取得
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

      // ✅ userChanges() へ反映（Web でも安定させる）
      await user.reload();
      _log('auth profile reload done');
    } catch (e) {
      // ここは致命にしない（作成は成功しているため）
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

      // ✅ 認証主体 UID（= absolute schema の userId）
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

      // (1) ✅ まず Avatar を作成（アイコンURLは後で PATCH）
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

      // ✅ 表示名だけは FirebaseAuth に同期（email表示を防ぐ）
      await _syncAuthDisplayName(user: user, avatarName: avatarName);

      // (2) ✅ アイコンがあれば：署名付きURL→PUTアップロード→https URL を PATCH 保存（方針A）
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
        final expiresAt = s(signed.expiresAt);

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
          'expiresAt="${expiresAt.isEmpty ? "-" : expiresAt}" uploadUrl="$short"',
        );

        await _repo.uploadToSignedUrl(
          uploadUrl: uploadUrl,
          bytes: bytes,
          contentType: mimeType,
        );

        _log('icon upload ok bytes=$size');

        // ✅ public https URL を生成し、avatarIcon として PATCH で保存する
        final httpsUrl = _repo.publicHttpsUrlFromBucketObject(
          bucket: bucket,
          objectPath: objectPath,
        );

        if (httpsUrl.isEmpty) {
          _log(
            'icon https url build failed avatarId=$avatarId bucket="$bucket" objectPath="$objectPath"',
          );
          return AvatarCreateResult(
            ok: false,
            message: 'アイコンURLの生成に失敗しました。',
            createdAvatarId: avatarId,
          );
        }

        _log('icon https url built avatarId=$avatarId url="$httpsUrl"');

        await _repo.update(
          id: avatarId,
          request: UpdateAvatarRequest(
            avatarIcon: httpsUrl, // ✅ 方針A: https://... を保存
          ),
        );

        _log('icon url patch ok avatarId=$avatarId');
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
