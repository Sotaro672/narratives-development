// frontend\sns\lib\features\auth\application\avatar_create_service.dart

import 'package:flutter/foundation.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import 'package:web/web.dart' as web;

import '../../avatar/avatar_repository_http.dart';

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
      if (files == null || files.length == 0) return null;

      final f = files.item(0);
      if (f == null) return null;

      // ✅ ブラウザに一時URLを作らせる
      objectUrl = web.URL.createObjectURL(f);

      // ✅ そのURLを HTTP GET して bytes を取得（ArrayBuffer/Uint8Array 不要）
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

      // ✅ package:web の File.name/type は（この環境では）Dart String
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
      // ✅ 一時URLは必ず破棄
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
  // Save (create avatar)
  // ============================

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

      // ✅ 認証主体 UID
      final uid = user.uid.trim();
      if (uid.isEmpty) {
        return AvatarCreateResult(ok: false, message: 'uid が取得できませんでした。');
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

      _log(
        'avatar save start uid=$uid name="$avatarName" '
        'profileLen=${profile.length} link="${link.isEmpty ? "-" : link}" '
        'iconBytesLen=${iconBytes?.lengthInBytes ?? 0} file="${s(iconFileName)}" mime="${s(iconMimeType)}"',
      );

      // ✅ いまは avatarIcon を送らない（アップロード連携は次のステップ）
      // ✅ backend 仕様変更に合わせて firebaseUid -> userUid
      // ✅ userId は (現状) uid を流用
      final created = await _repo.create(
        request: CreateAvatarRequest(
          userId: uid,
          userUid: uid,
          avatarName: avatarName,
          avatarIcon: null,
          profile: profile.isEmpty ? null : profile,
          externalLink: link.isEmpty ? null : link,
        ),
      );

      _log('avatar save ok avatarId=${created.id} uid=$uid');

      return AvatarCreateResult(
        ok: true,
        message: 'アバターを作成しました。',
        nextRoute: '/',
        createdAvatarId: created.id,
      );
    } catch (e) {
      _log('avatar save failed err=$e');
      return AvatarCreateResult(ok: false, message: e.toString());
    }
  }
}
