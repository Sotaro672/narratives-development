// frontend/mall/lib/features/avatar/presentation/vm/avatar_form_vm.dart
import 'dart:async';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import '../../infrastructure/avatar_api_client.dart';

enum AvatarFormMode { create, edit }

/// 「PATCHリクエストを渡せる」ための差分DTO
/// - null: フィールド自体を送らない（更新しない）
/// - ""  : クリアしたい（backendが "" を nil として扱う契約の場合）
///
/// IMPORTANT (推奨B):
/// - me PATCH では avatarIcon を送らない（運用で担保）
/// - 画像実体の更新/削除は別エンドポイント（signed PUT / delete object）で行う
@immutable
class AvatarPatchRequest {
  const AvatarPatchRequest({
    this.avatarName,
    this.avatarIcon,
    this.profile,
    this.externalLink,
  });

  final String? avatarName;

  /// ⚠️ 推奨Bでは原則使わない（送らない）
  /// - 互換のためフィールド自体は残すが、VM側で build 時に入れない
  final String? avatarIcon;

  final String? profile;
  final String? externalLink;

  bool get isEmpty =>
      avatarName == null &&
      avatarIcon == null &&
      profile == null &&
      externalLink == null;

  Map<String, dynamic> toJson() {
    final m = <String, dynamic>{};
    if (avatarName != null) m['avatarName'] = avatarName;
    if (avatarIcon != null) m['avatarIcon'] = avatarIcon;
    if (profile != null) m['profile'] = profile;
    if (externalLink != null) m['externalLink'] = externalLink;
    return m;
  }

  @override
  String toString() => 'AvatarPatchRequest(${toJson()})';
}

/// 画面側から「PATCHを投げる処理」を差し込めるようにするための型
typedef AvatarPatchSubmitter = Future<void> Function(AvatarPatchRequest patch);

class AvatarFormVm extends ChangeNotifier {
  AvatarFormVm({
    required this.mode,
    AvatarApiClient? apiClient,
    AvatarPatchSubmitter? submitPatch,
  }) : _apiClient = apiClient ?? AvatarApiClient(),
       _submitPatch = submitPatch;

  /// ✅ AvatarFormVm は edit 専用として運用する（create は AvatarCreateService を使う）
  final AvatarFormMode mode;

  final AvatarApiClient _apiClient;

  /// 任意: 画面/上位層から DI される PATCH 実行処理
  /// - 例: apiClient.patchMeAvatar(...) など
  final AvatarPatchSubmitter? _submitPatch;

  // form controllers
  final nameCtrl = TextEditingController();
  final profileCtrl = TextEditingController();
  final linkCtrl = TextEditingController();

  // icon (newly picked)
  Uint8List? iconBytes;
  String? iconFileName;

  // ✅ existing icon url (from backend) for edit-prefill display
  String? existingAvatarIconUrl;

  // ✅ 画像アップロード後に得た https://... を保持（UIプレビュー用）
  String? uploadedAvatarIconUrl;

  // ✅ 最後に組み立てた PATCH（デバッグ/遷移で渡す用途）
  AvatarPatchRequest? lastBuiltPatch;

  // ui state
  bool saving = false;
  bool loadingInitial = false;
  bool _initialLoaded = false;

  // ✅ edit差分判定用の初期値
  String _initialAvatarName = '';
  String _initialProfile = '';
  String _initialExternalLink = '';

  // ✅ patch実行のための識別子（me contract 由来）
  String _meAvatarId = '';

  String? msg;
  bool isSuccessMessage = false;

  bool get canSave {
    if (saving) return false;
    return nameCtrl.text.trim().isNotEmpty;
  }

  // ✅ nullable-safe trim helper
  String _s(Object? v) => (v ?? '').toString().trim();

  bool _isHttpUrl(String? v) {
    final s = (v ?? '').trim();
    if (s.isEmpty) return false;
    return s.startsWith('http://') || s.startsWith('https://');
  }

  /// 画像アップロード（別API）後に得た URL(https://...) を注入する（プレビュー用途）
  void setUploadedAvatarIconUrl(String? url) {
    final v = _s(url);
    uploadedAvatarIconUrl = v.isEmpty ? null : v;
    notifyListeners();
  }

  /// 新規選択された iconBytes があるか（= signed PUT でアップロードが必要か）
  bool get needsIconUpload => iconBytes != null && iconBytes!.isNotEmpty;

  /// edit の場合に「既存情報をフォームへ反映」する。
  Future<void> loadInitialIfNeeded() async {
    if (mode != AvatarFormMode.edit) return;
    if (_initialLoaded) return;

    loadingInitial = true;
    msg = null;
    isSuccessMessage = false;
    notifyListeners();

    try {
      final dto = await _apiClient.fetchMyAvatarProfile();

      final aid = _s(dto?.avatarId);
      if (dto == null || aid.isEmpty) {
        return;
      }

      _meAvatarId = aid;

      final avatarName = _s(dto.avatarName);
      if (avatarName.isNotEmpty) {
        nameCtrl.text = avatarName;
      }

      final profile = _s(dto.profile);
      if (profile.isNotEmpty) {
        profileCtrl.text = profile;
      }

      final link = _s(dto.externalLink);
      if (link.isNotEmpty) {
        linkCtrl.text = link;
      }

      final iconUrl = _s(dto.avatarIcon);
      if (iconUrl.isNotEmpty && _isHttpUrl(iconUrl)) {
        existingAvatarIconUrl = iconUrl;
      } else {
        existingAvatarIconUrl = null;
      }

      _initialAvatarName = _s(nameCtrl.text);
      _initialProfile = _s(profileCtrl.text);
      _initialExternalLink = _s(linkCtrl.text);

      _initialLoaded = true;
    } catch (_) {
      // fail-open
    } finally {
      loadingInitial = false;
      notifyListeners();
    }
  }

  void onNameChanged() {
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
    uploadedAvatarIconUrl = null;
    notifyListeners();
  }

  Future<void> deleteExistingIconObject() async {
    if (mode != AvatarFormMode.edit) return;

    if (_meAvatarId.trim().isEmpty) {
      msg = 'アバター情報が未ロードのため、既存画像を削除できません。';
      isSuccessMessage = false;
      notifyListeners();
      return;
    }

    if (saving || loadingInitial) return;

    saving = true;
    msg = null;
    isSuccessMessage = false;
    notifyListeners();

    try {
      await _apiClient.deleteMeAvatarIconObject();

      existingAvatarIconUrl = null;
      uploadedAvatarIconUrl = null;

      iconBytes = null;
      iconFileName = null;

      isSuccessMessage = true;
      msg = '既存画像を削除しました。';
    } catch (e) {
      isSuccessMessage = false;
      msg = '既存画像の削除に失敗しました: $e';
      rethrow;
    } finally {
      saving = false;
      notifyListeners();
    }
  }

  bool get hasAnyIconPreview {
    if (iconBytes != null && iconBytes!.isNotEmpty) return true;
    if ((uploadedAvatarIconUrl ?? '').trim().isNotEmpty) return true;
    return (existingAvatarIconUrl ?? '').trim().isNotEmpty;
  }

  /// UI表示用: 優先順位は「新規選択(bytes) > アップロード済URL > 既存URL」
  String? get iconPreviewUrl {
    final u = _s(uploadedAvatarIconUrl);
    if (_isHttpUrl(u)) return u;

    final e = _s(existingAvatarIconUrl);
    if (_isHttpUrl(e)) return e;

    return null;
  }

  AvatarPatchRequest buildPatchRequest() {
    final name = nameCtrl.text.trim();
    final profile = profileCtrl.text.trim();
    final link = linkCtrl.text.trim();

    // ✅ create は使わない前提だが、念のため build は残す
    if (mode == AvatarFormMode.create) {
      return AvatarPatchRequest(
        avatarName: name.isEmpty ? null : name,
        profile: profile.isEmpty ? null : profile,
        externalLink: link.isEmpty ? null : link,
      );
    }

    // edit: 差分のみ
    String? patchName;
    String? patchProfile;
    String? patchLink;

    if (name != _initialAvatarName) {
      patchName = name;
    }
    if (profile != _initialProfile) {
      patchProfile = profile;
    }
    if (link != _initialExternalLink) {
      patchLink = link;
    }

    return AvatarPatchRequest(
      avatarName: patchName,
      profile: patchProfile,
      externalLink: patchLink,
    );
  }

  /// ✅ edit専用の保存（PATCH）
  ///
  /// IMPORTANT:
  /// - create 画面では使わない（AvatarCreateService を使う）
  /// - submitPatch 未注入でも throw せず false を返して UI を落とさない
  Future<bool> save({AvatarPatchSubmitter? submitPatch}) async {
    // ✅ create では呼ばない契約。誤って呼ばれても落とさない。
    if (mode == AvatarFormMode.create) {
      msg = 'この画面の保存処理は create では使用しません。';
      isSuccessMessage = false;
      notifyListeners();
      return false;
    }

    if (!canSave) return false;

    if (_meAvatarId.trim().isEmpty) {
      msg = 'アバター情報が未ロードのため、更新できません。';
      isSuccessMessage = false;
      notifyListeners();
      return false;
    }

    saving = true;
    msg = null;
    isSuccessMessage = false;
    notifyListeners();

    try {
      final patch = buildPatchRequest();
      lastBuiltPatch = patch;

      // patch が空なら no-op
      if (patch.isEmpty) {
        isSuccessMessage = true;
        msg = '変更がありません。';
        return true;
      }

      final submit = submitPatch ?? _submitPatch;
      if (submit == null) {
        msg = '更新処理が設定されていません（submitPatch が未注入です）。';
        isSuccessMessage = false;
        return false;
      }

      await submit(patch);

      _initialAvatarName = _s(nameCtrl.text);
      _initialProfile = _s(profileCtrl.text);
      _initialExternalLink = _s(linkCtrl.text);

      isSuccessMessage = true;
      msg = 'アバターを更新しました。';
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
