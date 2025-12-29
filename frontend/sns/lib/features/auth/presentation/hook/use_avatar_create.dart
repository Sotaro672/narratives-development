// frontend/sns/lib/features/auth/presentation/hook/use_avatar_create.dart

import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../application/avatar_create_service.dart';

/// AvatarCreatePage の状態/処理をまとめる（Flutter では ChangeNotifier で hook 風にする）
class UseAvatarCreate extends ChangeNotifier {
  UseAvatarCreate({required this.from, AvatarCreateService? service})
    : _service = service ?? const AvatarCreateService();

  /// optional back route
  final String? from;

  final AvatarCreateService _service;

  final nameCtrl = TextEditingController();
  final profileCtrl = TextEditingController();
  final linkCtrl = TextEditingController();

  // ✅ 実画像ファイルの bytes
  Uint8List? iconBytes;

  // ✅ ファイルメタ
  String? iconFileName; // 例: "avatar.png"
  String? iconMimeType; // 例: "image/png"

  bool saving = false;
  String? msg;

  @override
  void dispose() {
    nameCtrl.dispose();
    profileCtrl.dispose();
    linkCtrl.dispose();
    super.dispose();
  }

  String backTo() {
    final f = _service.s(from);
    if (f.isNotEmpty) return f;
    return '/billing-address';
  }

  bool get canSave {
    if (saving) return false;
    if (_service.s(nameCtrl.text).isEmpty) return false;

    // 画像を必須にする場合:
    // if (iconBytes == null) return false;

    return true;
  }

  void onNameChanged() {
    notifyListeners();
  }

  // ============================
  // Icon picker (REAL image file)
  // ============================

  Future<void> pickIcon() async {
    msg = null;
    notifyListeners();

    final res = await _service.pickIconWeb();
    if (res == null) return;

    if (res.error != null) {
      msg = res.error;
      notifyListeners();
      return;
    }

    iconBytes = res.bytes;
    iconFileName = res.fileName;
    iconMimeType = res.mimeType;

    msg = 'アイコン画像を選択しました。';
    notifyListeners();
  }

  void clearIcon() {
    iconBytes = null;
    iconFileName = null;
    iconMimeType = null;
    notifyListeners();
  }

  // ============================
  // Save (still dummy for now)
  // ============================

  Future<void> saveDummy(BuildContext context) async {
    final link = _service.s(linkCtrl.text);
    if (!_service.isValidUrlOrEmpty(link)) {
      msg = '外部リンクは http(s) のURLを入力してください。';
      notifyListeners();
      return;
    }

    saving = true;
    msg = null;
    notifyListeners();

    Object? caught;
    try {
      await _service.saveDummyDelay();

      msg = 'アバターを作成しました（ダミー）。';
      notifyListeners();

      if (!context.mounted) return;
      context.go('/');
    } catch (e) {
      caught = e;
      msg = e.toString();
      notifyListeners();
    } finally {
      saving = false;
      notifyListeners();
    }

    // ignore: unused_local_variable
    final _ = caught;
  }
}
