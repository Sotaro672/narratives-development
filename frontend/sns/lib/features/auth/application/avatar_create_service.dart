// frontend/sns/lib/features/auth/application/avatar_create_service.dart
// ignore_for_file: avoid_web_libraries_in_flutter, deprecated_member_use

import 'dart:typed_data';

import 'package:flutter/foundation.dart';

// ✅ 依存追加なしで Web のファイル選択を実現（lint は上で抑制）
import 'dart:html' as html;

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

class AvatarCreateService {
  const AvatarCreateService();

  String s(String? v) => (v ?? '').trim();

  bool isValidUrlOrEmpty(String s0) {
    final v = s(s0);
    if (v.isEmpty) return true;
    final uri = Uri.tryParse(v);
    if (uri == null) return false;
    if (!uri.hasScheme) return false;
    if (uri.scheme != 'http' && uri.scheme != 'https') return false;
    return uri.host.isNotEmpty;
  }

  /// ✅ 実画像を選択（Web のみ実装 / 依存パッケージ不要）
  ///
  /// - 成功: bytes/fileName/mimeType を返す
  /// - キャンセル: null を返す
  /// - 失敗: error をセットして返す
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
      if (result is! ByteBuffer) {
        return const PickIconResult(
          bytes: null,
          fileName: null,
          mimeType: null,
          error: '画像の読み込みに失敗しました。',
        );
      }

      final bytes = Uint8List.view(result);
      final name = s(file.name).isEmpty ? null : file.name;
      final mime = s(file.type).isEmpty ? null : file.type;

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

  /// いまはダミー保存（将来: 署名付きURL取得→アップロード→create API）
  Future<void> saveDummyDelay() async {
    await Future<void>.delayed(const Duration(milliseconds: 700));
  }
}
