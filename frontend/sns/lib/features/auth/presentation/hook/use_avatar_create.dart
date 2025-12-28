// frontend/sns/lib/features/auth/presentation/hook/use_avatar_create.dart
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

/// AvatarCreatePage の状態/処理をまとめる（Flutter では ChangeNotifier で hook 風にする）
class UseAvatarCreate extends ChangeNotifier {
  UseAvatarCreate({required this.from});

  /// optional back route
  final String? from;

  final nameCtrl = TextEditingController();
  final profileCtrl = TextEditingController();
  final linkCtrl = TextEditingController();

  Uint8List? iconBytes; // ダミー
  bool saving = false;
  String? msg;

  @override
  void dispose() {
    nameCtrl.dispose();
    profileCtrl.dispose();
    linkCtrl.dispose();
    super.dispose();
  }

  String _s(String? v) => (v ?? '').trim();

  String backTo() {
    final f = _s(from);
    if (f.isNotEmpty) return f;
    return '/billing-address';
  }

  bool get canSave {
    if (saving) return false;
    if (_s(nameCtrl.text).isEmpty) return false;
    // 画像を必須にする場合:
    // if (iconBytes == null) return false;
    return true;
  }

  bool isValidUrlOrEmpty(String s) {
    final v = _s(s);
    if (v.isEmpty) return true;
    final uri = Uri.tryParse(v);
    if (uri == null) return false;
    if (!uri.hasScheme) return false;
    if (uri.scheme != 'http' && uri.scheme != 'https') return false;
    return uri.host.isNotEmpty;
  }

  void onNameChanged() {
    notifyListeners();
  }

  Future<void> pickIconDummy() async {
    iconBytes = Uint8List.fromList(List<int>.generate(64, (i) => i));
    msg = 'アイコン画像を選択しました（ダミー）。';
    notifyListeners();
  }

  void clearIcon() {
    iconBytes = null;
    notifyListeners();
  }

  Future<void> saveDummy(BuildContext context) async {
    final link = _s(linkCtrl.text);
    if (!isValidUrlOrEmpty(link)) {
      msg = '外部リンクは http(s) のURLを入力してください。';
      notifyListeners();
      return;
    }

    saving = true;
    msg = null;
    notifyListeners();

    Object? caught;
    try {
      await Future<void>.delayed(const Duration(milliseconds: 700));

      msg = 'アバターを作成しました（ダミー）。';
      notifyListeners();

      if (!context.mounted) return;
      // ✅ 保存後は Home に戻る
      context.go('/');
    } catch (e) {
      caught = e;
      msg = e.toString();
      notifyListeners();
    } finally {
      saving = false;
      notifyListeners();
    }

    // caught は将来ログ用途などに使える（未使用でもOK）
    // ignore: unused_local_variable
    final _ = caught;
  }
}
