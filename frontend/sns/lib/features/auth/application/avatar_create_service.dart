// ignore_for_file: avoid_web_libraries_in_flutter, deprecated_member_use

import 'dart:typed_data';

import 'package:flutter/foundation.dart';
import 'package:firebase_auth/firebase_auth.dart';

// ✅ 依存追加なしで Web のファイル選択を実現（lint は上で抑制）
import 'dart:html' as html;

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

    try {
      final input = html.FileUploadInputElement()
        ..accept = 'image/*'
        ..multiple = false;

      input.click();
      await input.onChange.first;

      final files = input.files;
      if (files == null || files.isEmpty) return null;

      final file = files.first;

      final reader = html.FileReader();
      reader.readAsArrayBuffer(file);
      await reader.onLoad.first;

      final result = reader.result;

      Uint8List? bytes;
      if (result is ByteBuffer) {
        // ✅ 安全のため copy（view だと環境によって不安定なことがある）
        bytes = Uint8List.fromList(result.asUint8List());
      } else if (result is Uint8List) {
        bytes = Uint8List.fromList(result);
      } else {
        return const PickIconResult(
          bytes: null,
          fileName: null,
          mimeType: null,
          error: '画像の読み込みに失敗しました（result の型が不明です）。',
        );
      }

      final name = s(file.name).isEmpty ? null : file.name;
      final mime = s(file.type).isEmpty ? null : file.type;

      if (bytes.isEmpty) {
        return const PickIconResult(
          bytes: null,
          fileName: null,
          mimeType: null,
          error: '画像の読み込みに失敗しました（bytes が空です）。',
        );
      }

      return PickIconResult(bytes: bytes, fileName: name, mimeType: mime);
    } catch (e) {
      return PickIconResult(
        bytes: null,
        fileName: null,
        mimeType: null,
        error: '画像の選択に失敗しました: $e',
      );
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
      final created = await _repo.create(
        request: CreateAvatarRequest(
          userId: uid,
          firebaseUid: uid,
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
