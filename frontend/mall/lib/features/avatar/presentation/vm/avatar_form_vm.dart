//frontend\mall\lib\features\avatar\presentation\vm\avatar_form_vm.dart
import 'dart:async';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import '../../infrastructure/avatar_api_client.dart';

enum AvatarFormMode { create, edit }

class AvatarFormVm extends ChangeNotifier {
  AvatarFormVm({required this.mode, AvatarApiClient? apiClient})
    : _apiClient = apiClient ?? const AvatarApiClient() {
    // ここで初期値を入れても良いが、edit の初期ロードがあるため loadInitial を用意
  }

  final AvatarFormMode mode;
  final AvatarApiClient _apiClient;

  // form controllers
  final nameCtrl = TextEditingController();
  final profileCtrl = TextEditingController();
  final linkCtrl = TextEditingController();

  // icon
  Uint8List? iconBytes;
  String? iconFileName;

  // ui state
  bool saving = false;
  String? msg;
  bool isSuccessMessage = false;

  bool get canSave {
    if (saving) return false;
    return nameCtrl.text.trim().isNotEmpty;
  }

  String _s(String? v) => (v ?? '').trim();

  /// edit の場合に「既存情報をフォームへ反映」する。
  /// 現状の API 仕様がこのスレッドでは不明なため、
  /// まずは fetchMeAvatar() が成功することだけ確認し、空なら no-op にしています。
  Future<void> loadInitialIfNeeded() async {
    if (mode != AvatarFormMode.edit) return;

    try {
      // NOTE: ここでプロフィール詳細（name/bio/link/iconUrl 等）を取れるAPIがあるなら置換してください。
      // 現状は me avatar の存在チェックのみ。
      final me = await _apiClient.fetchMeAvatar();
      if (me == null || _s(me.avatarId).isEmpty) {
        // avatar が無い -> edit できない。ページ側で create へ誘導しても良い
        return;
      }
    } catch (_) {
      // fail-open
    }
  }

  void onNameChanged() {
    // canSave の更新目的
    notifyListeners();
  }

  Future<void> pickIcon() async {
    try {
      final res = await FilePicker.platform.pickFiles(
        type: FileType.image,
        withData: true,
      );
      if (res == null) return;

      final f = res.files.single;
      final bytes = f.bytes;
      if (bytes == null) return;

      iconBytes = bytes;
      iconFileName = _s(f.name);
      notifyListeners();
    } catch (e) {
      msg = '画像の選択に失敗しました: $e';
      isSuccessMessage = false;
      notifyListeners();
    }
  }

  void clearIcon() {
    iconBytes = null;
    iconFileName = null;
    notifyListeners();
  }

  /// 保存（create/edit 共通）
  /// ここでは「API 未確定」なため、I/O 部分は差し替えやすい骨格のみ実装。
  Future<bool> save() async {
    if (!canSave) return false;

    saving = true;
    msg = null;
    notifyListeners();

    try {
      // 例:
      // if (mode == AvatarFormMode.create) {
      //   await _apiClient.createAvatar(...);
      // } else {
      //   await _apiClient.updateAvatar(...);
      // }
      //
      // iconBytes のアップロードは次ステップで実装予定なら、ここでは保持だけでもOK

      // 暫定: 成功扱い
      await Future<void>.delayed(const Duration(milliseconds: 250));

      isSuccessMessage = true;
      msg = mode == AvatarFormMode.create ? 'アバターを保存しました。' : 'アバターを更新しました。';

      return true;
    } catch (e) {
      isSuccessMessage = false;
      msg = '保存に失敗しました: $e';
      return false;
    } finally {
      saving = false;
      notifyListeners();
    }
  }

  @override
  void dispose() {
    nameCtrl.dispose();
    profileCtrl.dispose();
    linkCtrl.dispose();
    super.dispose();
  }
}
