// frontend\mall\lib\features\avatar\presentation\vm\avatar_form_vm.dart
import 'dart:async';
import 'dart:typed_data';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';

import '../../infrastructure/avatar_api_client.dart';

enum AvatarFormMode { create, edit }

class AvatarFormVm extends ChangeNotifier {
  AvatarFormVm({required this.mode, AvatarApiClient? apiClient})
    // ❌ const AvatarApiClient() は不可（MallAuthedApi 統一後）
    : _apiClient = apiClient ?? AvatarApiClient();

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
  bool loadingInitial = false;
  bool _initialLoaded = false;

  String? msg;
  bool isSuccessMessage = false;

  bool get canSave {
    if (saving) return false;
    return nameCtrl.text.trim().isNotEmpty;
  }

  String _s(String? v) => (v ?? '').trim();

  /// edit の場合に「既存情報をフォームへ反映」する。
  ///
  /// - 二重ロードを防止（_initialLoaded）
  /// - 取得できたフィールドだけをプレフィル（null/空文字なら上書きしない）
  /// - API が avatarId しか返さない場合でも安全に動作（上書きしない）
  Future<void> loadInitialIfNeeded() async {
    if (mode != AvatarFormMode.edit) return;
    if (_initialLoaded) return;

    loadingInitial = true;
    msg = null;
    isSuccessMessage = false;
    notifyListeners();

    try {
      // ✅ edit プレフィル用 API（なければ avatarId だけ返ってきても OK）
      final dto = await _apiClient.fetchMyAvatarProfile();
      if (dto == null || dto.avatarId.trim().isEmpty) {
        // avatar が無い -> edit できない（ページ側で create へ誘導しても良い）
        return;
      }

      // ✅ dto は "MeAvatar"（avatar patch 全体）として扱う前提
      // - dto.avatarName (nullable でも安全に)
      // - dto.profile (nullable)
      // - dto.externalLink (nullable)
      // - dto.avatarIcon (nullable)

      final avatarName = (dto.avatarName ?? '').trim();
      if (avatarName.isNotEmpty) {
        nameCtrl.text = avatarName;
      }

      final profile = (dto.profile ?? '').trim();
      if (profile.isNotEmpty) {
        profileCtrl.text = profile;
      }

      final link = (dto.externalLink ?? '').trim();
      if (link.isNotEmpty) {
        linkCtrl.text = link;
      }

      // avatarIcon は bytes で扱っているため、ここでは未反映（必要なら別途実装）
      _initialLoaded = true;
    } catch (_) {
      // fail-open
    } finally {
      loadingInitial = false;
      notifyListeners();
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
  ///
  /// ここでは「API 未確定」なため、I/O 部分は差し替えやすい骨格のみ実装。
  Future<bool> save() async {
    if (!canSave) return false;

    saving = true;
    msg = null;
    isSuccessMessage = false;
    notifyListeners();

    try {
      final name = nameCtrl.text.trim();
      final profile = profileCtrl.text.trim();
      final link = linkCtrl.text.trim();

      // TODO: ここで create / update API を呼ぶ
      // 例:
      // if (mode == AvatarFormMode.create) {
      //   await _apiClient.createAvatar(
      //     avatarName: name,
      //     profile: profile.isEmpty ? null : profile,
      //     externalLink: link.isEmpty ? null : link,
      //     avatarIconBytes: iconBytes,
      //     avatarIconFileName: iconFileName,
      //   );
      // } else {
      //   await _apiClient.updateAvatar(
      //     avatarName: name,
      //     profile: profile.isEmpty ? null : profile,
      //     externalLink: link.isEmpty ? null : link,
      //     avatarIconBytes: iconBytes,
      //     avatarIconFileName: iconFileName,
      //   );
      // }

      // 暫定: 成功扱い
      await Future<void>.delayed(const Duration(milliseconds: 250));

      // 近い将来の API 呼び出し用（未使用lint回避）
      // ignore: unused_local_variable
      final _ = (name, profile, link);

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
