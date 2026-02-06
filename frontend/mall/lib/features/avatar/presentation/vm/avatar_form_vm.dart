// frontend\mall\lib\features\avatar\presentation\vm\avatar_form_vm.dart
import 'dart:async';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../../infrastructure/avatar_api_client.dart';

enum AvatarFormMode { create, edit }

@immutable
class AvatarCreateRequest {
  const AvatarCreateRequest({
    required this.userId,
    required this.userUid,
    required this.avatarName,
    this.avatarIcon,
    this.profile,
    this.externalLink,
  });

  final String userId;
  final String userUid;
  final String avatarName;

  /// optional
  final String? avatarIcon;
  final String? profile;
  final String? externalLink;

  Map<String, dynamic> toJson() {
    final m = <String, dynamic>{
      'userId': userId.trim(),
      'userUid': userUid.trim(),
      'avatarName': avatarName.trim(),
    };

    final icon = (avatarIcon ?? '').trim();
    if (icon.isNotEmpty) m['avatarIcon'] = icon;

    final p = (profile ?? '').trim();
    if (p.isNotEmpty) m['profile'] = p;

    final l = (externalLink ?? '').trim();
    if (l.isNotEmpty) m['externalLink'] = l;

    return m;
  }

  @override
  String toString() => 'AvatarCreateRequest(${toJson()})';
}

@immutable
class AvatarPatchRequest {
  const AvatarPatchRequest({
    this.avatarName,
    this.avatarIcon,
    this.profile,
    this.externalLink,
  });

  final String? avatarName;
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

typedef AvatarPatchSubmitter = Future<void> Function(AvatarPatchRequest patch);
typedef AvatarCreateSubmitter = Future<void> Function(AvatarCreateRequest body);

/// ✅ 既存アイコン実体削除（GCS object delete）を差し込むための型
/// - 例: () => apiClient.deleteMeAvatarIconObject()
typedef AvatarDeleteIconObject = Future<void> Function();

class AvatarFormVm extends ChangeNotifier {
  AvatarFormVm({
    required this.mode,
    AvatarApiClient? apiClient,
    AvatarPatchSubmitter? submitPatch,
    AvatarCreateSubmitter? submitCreate,
    AvatarDeleteIconObject? deleteIconObject,
  }) : _apiClient = apiClient ?? AvatarApiClient(),
       _submitPatch = submitPatch,
       _submitCreate = submitCreate,
       _deleteIconObject = deleteIconObject;

  final AvatarFormMode mode;
  final AvatarApiClient _apiClient;

  final AvatarPatchSubmitter? _submitPatch;
  final AvatarCreateSubmitter? _submitCreate;

  /// ✅ 任意: 画面/上位層から DI される「既存アイコン実体削除」処理
  /// - AvatarApiClient に deleteMeAvatarIconObject() が無くても VM はコンパイルできる
  final AvatarDeleteIconObject? _deleteIconObject;

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

  AvatarPatchRequest? lastBuiltPatch;
  AvatarCreateRequest? lastBuiltCreate;

  bool saving = false;
  bool loadingInitial = false;
  bool _initialLoaded = false;

  String _initialAvatarName = '';
  String _initialProfile = '';
  String _initialExternalLink = '';

  String _meAvatarId = '';

  String? msg;
  bool isSuccessMessage = false;

  bool get canSave {
    if (saving) return false;
    return nameCtrl.text.trim().isNotEmpty;
  }

  String _s(Object? v) => (v ?? '').toString().trim();

  bool _isHttpUrl(String? v) {
    final s = (v ?? '').trim();
    if (s.isEmpty) return false;
    return s.startsWith('http://') || s.startsWith('https://');
  }

  String _currentUid() => (FirebaseAuth.instance.currentUser?.uid ?? '').trim();

  void setUploadedAvatarIconUrl(String? url) {
    final v = _s(url);
    uploadedAvatarIconUrl = v.isEmpty ? null : v;
    notifyListeners();
  }

  bool get needsIconUpload => iconBytes != null && iconBytes!.isNotEmpty;

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

  /// ✅ 推奨B: 既存画像を削除する（DB の avatarIcon 文字列は変更しない）
  ///
  /// - 画面/上位層から deleteIconObject を DI して実行する
  /// - 例: AvatarFormVm(deleteIconObject: () => apiClient.deleteMeAvatarIconObject())
  Future<void> deleteExistingIconObject() async {
    if (mode != AvatarFormMode.edit) return;

    if (_meAvatarId.trim().isEmpty) {
      msg = 'アバター情報が未ロードのため、既存画像を削除できません。';
      isSuccessMessage = false;
      notifyListeners();
      return;
    }

    if (saving || loadingInitial) return;

    final deleter = _deleteIconObject;
    if (deleter == null) {
      msg = '既存画像削除の処理が未設定です。deleteIconObject を DI してください。';
      isSuccessMessage = false;
      notifyListeners();
      return;
    }

    saving = true;
    msg = null;
    isSuccessMessage = false;
    notifyListeners();

    try {
      await deleter();

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

  String? get iconPreviewUrl {
    final u = _s(uploadedAvatarIconUrl);
    if (_isHttpUrl(u)) return u;

    final e = _s(existingAvatarIconUrl);
    if (_isHttpUrl(e)) return e;

    return null;
  }

  AvatarCreateRequest buildCreateRequest() {
    final uid = _currentUid();
    final name = nameCtrl.text.trim();
    final profile = profileCtrl.text.trim();
    final link = linkCtrl.text.trim();

    return AvatarCreateRequest(
      userId: uid,
      userUid: uid,
      avatarName: name,
      profile: profile.isEmpty ? null : profile,
      externalLink: link.isEmpty ? null : link,
    );
  }

  AvatarPatchRequest buildPatchRequest() {
    final name = nameCtrl.text.trim();
    final profile = profileCtrl.text.trim();
    final link = linkCtrl.text.trim();

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

  Future<bool> save({
    AvatarPatchSubmitter? submitPatch,
    AvatarCreateSubmitter? submitCreate,
  }) async {
    if (!canSave) return false;

    if (mode == AvatarFormMode.create) {
      final uid = _currentUid();
      if (uid.isEmpty) {
        msg = 'ログイン情報（Firebase UID）が取得できないため、作成できません。';
        isSuccessMessage = false;
        notifyListeners();
        return false;
      }
    }

    if (mode == AvatarFormMode.edit && _meAvatarId.trim().isEmpty) {
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
      if (mode == AvatarFormMode.create) {
        final body = buildCreateRequest();
        lastBuiltCreate = body;

        final submit = submitCreate ?? _submitCreate;
        if (submit == null) {
          throw StateError(
            'CREATE submitter is not configured. Provide submitCreate or inject _submitCreate.',
          );
        }

        await submit(body);

        isSuccessMessage = true;
        msg = 'アバターを保存しました。';
        return true;
      }

      final patch = buildPatchRequest();
      lastBuiltPatch = patch;

      final submit = submitPatch ?? _submitPatch;
      if (submit == null) {
        throw StateError(
          'PATCH submitter is not configured. Provide submitPatch or inject _submitPatch.',
        );
      }

      if (patch.isEmpty) {
        isSuccessMessage = true;
        msg = '変更がありません。';
        return true;
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
