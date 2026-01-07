// frontend/sns/lib/features/auth/presentation/hook/use_avatar_create.dart
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import '../../application/avatar_create_service.dart';

/// AvatarCreatePage の状態/処理をまとめる（Flutter では ChangeNotifier で hook 風にする）
///
/// 方針:
/// - use_build_context_synchronously を避けるため、Hook 側では BuildContext を受け取らない
/// - 画面遷移（GoRouter）は Page 側の責務に寄せる
/// - 保存完了/失敗は msg と saving を通じて Page に通知する
class UseAvatarCreate extends ChangeNotifier {
  UseAvatarCreate({required this.from, AvatarCreateService? service})
    : _service = service ?? AvatarCreateService();

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

  // ✅ NEW: 作成成功時に Page が遷移先として使える情報
  String? createdAvatarId;
  String? successRedirectTo;

  @override
  void dispose() {
    nameCtrl.dispose();
    profileCtrl.dispose();
    linkCtrl.dispose();
    _service.dispose();
    super.dispose();
  }

  String backTo() => _service.backTo(from);

  bool get canSave {
    if (saving) return false;
    if (_service.s(nameCtrl.text).isEmpty) return false;
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
    if (res == null) {
      // cancel
      msg = '画像選択をキャンセルしました。';
      notifyListeners();
      return;
    }

    if (res.error != null) {
      msg = res.error;
      notifyListeners();
      return;
    }

    iconBytes = res.bytes;
    iconFileName = res.fileName;
    iconMimeType = res.mimeType;

    if (iconBytes == null || iconBytes!.isEmpty) {
      msg = '画像の読み込みに失敗しました（bytes が空です）。';
    } else {
      msg = 'アイコン画像を選択しました。';
    }
    notifyListeners();
  }

  void clearIcon() {
    iconBytes = null;
    iconFileName = null;
    iconMimeType = null;
    notifyListeners();
  }

  // ============================
  // Save
  // ============================

  /// ✅ BuildContext を受け取らない（Page 側で遷移する）
  /// 成功したら true / 失敗したら false を返す
  ///
  /// ✅ NEW:
  /// - 成功時: createdAvatarId / successRedirectTo をセット
  ///   Page 側で `if (ok) context.go(hook.successRedirectTo!)` のように使える
  Future<bool> save() async {
    saving = true;
    msg = null;

    // ✅ reset
    createdAvatarId = null;
    successRedirectTo = null;

    notifyListeners();

    try {
      final res = await _service.save(
        avatarNameRaw: nameCtrl.text,
        profileRaw: profileCtrl.text,
        externalLinkRaw: linkCtrl.text,
        // NOTE: 現段階では iconBytes は「プレビュー用」に保持するだけ。
        // アップロード連携（署名付きURL等）を入れたら service 側で利用する。
        iconBytes: iconBytes,
        iconFileName: iconFileName,
        iconMimeType: iconMimeType,
      );

      msg = res.message;

      if (res.ok) {
        // 暫定: avatarId = Firebase UID（AvatarPage 側の実装に合わせる）
        final uid = FirebaseAuth.instance.currentUser?.uid;
        createdAvatarId = (uid ?? '').trim();

        // ✅ AvatarPage へ遷移するための URL を作る
        // - AvatarPage は自分でも avatarId を URL に載せ直すが、最初から付けておくと安定
        // - from は “戻り先” として渡せる（router 側で拾う想定）
        final qp = <String, String>{};

        final b = backTo().trim();
        if (b.isNotEmpty) qp['from'] = b;

        final aid = createdAvatarId ?? '';
        if (aid.isNotEmpty) qp['avatarId'] = aid;

        successRedirectTo = Uri(
          path: '/avatar',
          queryParameters: qp,
        ).toString();
      }

      notifyListeners();
      return res.ok;
    } catch (e) {
      msg = e.toString();
      notifyListeners();
      return false;
    } finally {
      saving = false;
      notifyListeners();
    }
  }

  // ============================
  // Helpers for Page
  // ============================

  bool get isSuccessMessage {
    final m = (msg ?? '').trim();
    if (m.isEmpty) return false;
    return m.contains('作成しました') || m.contains('保存しました');
  }

  bool get canRedirectToAvatar {
    final ok = (successRedirectTo ?? '').trim().isNotEmpty;
    return ok;
  }
}
