// frontend\mall\lib\features\avatar\presentation\vm\avatar_form_vm.dart
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
  })  : _apiClient = apiClient ?? AvatarApiClient(),
        _submitPatch = submitPatch;

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
  // - 推奨B: これ自体は DB に保存しない（avatarIcon 文字列は固定 / me PATCHで送らない）
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
  // - 直接 id を叩かない構成でも、「初期ロード済み」判定として保持しておく
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
  ///
  /// - 二重ロードを防止（_initialLoaded）
  /// - 取得できたフィールドだけをプレフィル（null/空文字なら上書きしない）
  Future<void> loadInitialIfNeeded() async {
    if (mode != AvatarFormMode.edit) return;
    if (_initialLoaded) return;

    loadingInitial = true;
    msg = null;
    isSuccessMessage = false;
    notifyListeners();

    try {
      final dto = await _apiClient.fetchMyAvatarProfile();

      // ✅ dto / avatarId が nullable 実装でも落ちないようにする
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

      // ✅ 既存 avatarIcon を URL として保持（表示用途）
      final iconUrl = _s(dto.avatarIcon);
      if (iconUrl.isNotEmpty && _isHttpUrl(iconUrl)) {
        existingAvatarIconUrl = iconUrl;
      } else {
        existingAvatarIconUrl = null;
      }

      // ✅ 初期値を保存（差分PATCH用）
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

      // ✅ 新規選択が入ったら、表示は bytes 優先になる（existing は残してOK）
      // ✅ ただし URL は upload 成功後に setUploadedAvatarIconUrl() で注入する
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

    // ✅ clear は「新規選択の取り消し」。既存URLは残す（=元に戻る）
    // ✅ ただしアップロード済URLも「新規選択の結果」ならクリアしておくのが自然
    uploadedAvatarIconUrl = null;

    notifyListeners();
  }

  /// ✅ 推奨B: 既存画像を削除する（DB の avatarIcon 文字列は変更しない）
  ///
  /// - backend: DELETE /mall/me/avatar/icon-object を叩いて GCS object のみ削除
  /// - UI: 成功したら「表示上は」既存URLを消す（次回 GET では固定URLが返る想定だが、画像実体は消えている）
  Future<void> deleteExistingIconObject() async {
    // edit 前提（create で既存削除は意味を持たない）
    if (mode != AvatarFormMode.edit) return;

    // 初期ロード前なら avatarId が取れない可能性が高いので防御
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

      // 表示上は既存もアップロード済みも消す（固定URLはDBに残るが、実体は消えている前提）
      existingAvatarIconUrl = null;
      uploadedAvatarIconUrl = null;

      // 新規選択もリセット（削除操作後の状態を単純化）
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

  /// 既存アイコンも含めて「何か表示できるか」
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

  /// ✅ 差分PATCHを組み立てる
  ///
  /// 推奨B:
  /// - avatarIcon はここでは一切組み立てない（運用で担保 / API側でも拒否）
  ///
  /// edit:
  /// - 初期値との差分のみを含める
  /// - profile / externalLink は "" を送ってクリアできる（backend契約次第）
  ///
  /// create:
  /// - 全項目を含める（ただし空は null にして省略）
  AvatarPatchRequest buildPatchRequest() {
    final name = nameCtrl.text.trim();
    final profile = profileCtrl.text.trim();
    final link = linkCtrl.text.trim();

    if (mode == AvatarFormMode.create) {
      return AvatarPatchRequest(
        avatarName: name.isEmpty ? null : name,
        profile: profile.isEmpty ? null : profile,
        externalLink: link.isEmpty ? null : link,
        // avatarIcon: null (never send)
      );
    }

    // edit: 差分のみ
    String? patchName;
    String? patchProfile;
    String? patchLink;

    if (name != _initialAvatarName) {
      patchName = name; // 空は canSave で弾く想定
    }

    if (profile != _initialProfile) {
      patchProfile = profile; // "" を送ることでクリアを表現
    }

    if (link != _initialExternalLink) {
      patchLink = link; // "" を送ることでクリアを表現
    }

    return AvatarPatchRequest(
      avatarName: patchName,
      profile: patchProfile,
      externalLink: patchLink,
      // avatarIcon: null (never send)
    );
  }

  /// 保存（create/edit 共通）
  ///
  /// ✅ 本メソッドは
  /// - lastBuiltPatch を保持し
  /// - submitPatch（DI）を通じて実際のPATCH送信も可能
  /// とすることで「PATCHリクエストを渡せる」ようにします。
  ///
  /// IMPORTANT:
  /// - avatarIcon の更新/削除はこの save() では行わない
  /// - 画像は別エンドポイント（signed PUT / delete object）で処理する
  Future<bool> save({AvatarPatchSubmitter? submitPatch}) async {
    if (!canSave) return false;

    // edit の更新で avatarId が無いのは異常（fetch してない/壊れた状態）
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
      final patch = buildPatchRequest();
      lastBuiltPatch = patch;

      final submit = submitPatch ?? _submitPatch;
      if (submit == null) {
        throw StateError(
          'PATCH submitter is not configured. Provide submitPatch or inject _submitPatch.',
        );
      }

      // patch が空なら no-op で成功扱い（編集画面のUXを優先）
      if (mode == AvatarFormMode.edit && patch.isEmpty) {
        isSuccessMessage = true;
        msg = '変更がありません。';
        return true;
      }

      await submit(patch);

      // ✅ 成功したら「初期値」を現在値に更新（次回以降の差分判定がズレないように）
      _initialAvatarName = _s(nameCtrl.text);
      _initialProfile = _s(profileCtrl.text);
      _initialExternalLink = _s(linkCtrl.text);

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
