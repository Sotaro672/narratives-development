// frontend\mall\lib\features\auth\presentation\hook\use_avatar_create.dart
import 'dart:developer' as developer;

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

  // ===========================================================================
  // Logging (Chrome Console friendly)
  // ===========================================================================
  bool get _logEnabled => kDebugMode;

  void _log(String message) {
    if (!_logEnabled) return;

    final line = '[UseAvatarCreate] $message';

    // ✅ Flutter Web の Chrome Console に出すための保険:
    // - print(): console に出ることが多い
    // - developer.log(): DevTools/console に出やすい
    // - debugPrint(): 環境によっては console に出ないことがある
    if (kIsWeb) {
      // ignore: avoid_print
      print(line);
      developer.log(line, name: 'UseAvatarCreate');
      return;
    }

    debugPrint(line);
    developer.log(line, name: 'UseAvatarCreate');
  }

  String _maskEmail(String? email) {
    final e = (email ?? '').trim();
    if (e.isEmpty) return '';
    final at = e.indexOf('@');
    if (at <= 1) return '***';
    return '${e.substring(0, 2)}***${e.substring(at)}';
  }

  void _logAuthSnapshot(String stage) {
    if (!_logEnabled) return;

    final u = FirebaseAuth.instance.currentUser;
    if (u == null) {
      _log('$stage auth.currentUser = null (NOT signed in?)');
      return;
    }

    final uid = (u.uid).trim();
    final email = _maskEmail(u.email);
    final isAnon = u.isAnonymous;

    _log(
      '$stage auth.currentUser exists uid="${uid.isEmpty ? '(EMPTY)' : uid}" '
      'email="${email.isEmpty ? '(none)' : email}" '
      'isAnonymous=$isAnon',
    );

    if (uid.isEmpty) {
      _log('$stage WARN: uid is empty string after trim');
    }
  }

  // ===========================================================================
  // Lifecycle
  // ===========================================================================
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

    _log('pickIcon() start');
    _logAuthSnapshot('pickIcon()');

    final res = await _service.pickIconWeb();
    if (res == null) {
      // cancel
      msg = '画像選択をキャンセルしました。';
      _log('pickIcon() cancelled by user');
      notifyListeners();
      return;
    }

    if (res.error != null) {
      msg = res.error;
      _log('pickIcon() error: ${res.error}');
      notifyListeners();
      return;
    }

    iconBytes = res.bytes;
    iconFileName = res.fileName;
    iconMimeType = res.mimeType;

    final bytesLen = iconBytes?.length ?? 0;
    _log(
      'pickIcon() selected fileName="$iconFileName" mimeType="$iconMimeType" bytesLen=$bytesLen',
    );

    if (iconBytes == null || iconBytes!.isEmpty) {
      msg = '画像の読み込みに失敗しました（bytes が空です）。';
    } else {
      msg = 'アイコン画像を選択しました。';
    }
    notifyListeners();
  }

  void clearIcon() {
    _log('clearIcon()');
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

    final name = _service.s(nameCtrl.text);
    final prof = _service.s(profileCtrl.text);
    final link = _service.s(linkCtrl.text);

    _log(
      'save() start canSave=$canSave name="$name" profileLen=${prof.length} link="$link"',
    );
    _logAuthSnapshot('save() start');

    try {
      _log(
        'calling _service.save(...) iconBytesLen=${iconBytes?.length ?? 0} fileName="$iconFileName" mimeType="$iconMimeType"',
      );

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
      _log('returned from _service.save ok=${res.ok} message="${res.message}"');
      _logAuthSnapshot('after _service.save');

      if (res.ok) {
        // 暫定: avatarId = Firebase UID（AvatarPage 側の実装に合わせる）
        final uid = FirebaseAuth.instance.currentUser?.uid;
        final trimmedUid = (uid ?? '').trim();

        if (trimmedUid.isEmpty) {
          _log(
            'WARN: res.ok=true but Firebase uid is null/empty. createdAvatarId will be empty.',
          );
        }

        createdAvatarId = trimmedUid;
        _log('createdAvatarId="$createdAvatarId"');

        // ✅ AvatarPage へ遷移するための URL を作る
        final qp = <String, String>{};

        final b = backTo().trim();
        if (b.isNotEmpty) qp['from'] = b;

        final aid = createdAvatarId ?? '';
        if (aid.isNotEmpty) qp['avatarId'] = aid;

        successRedirectTo = Uri(
          path: '/avatar',
          queryParameters: qp,
        ).toString();

        _log('successRedirectTo="$successRedirectTo"');
      }

      notifyListeners();
      return res.ok;
    } catch (e, st) {
      msg = e.toString();
      _log('ERROR in save(): $e');
      if (_logEnabled) {
        // stacktrace も出す（webでも確認しやすい）
        final stStr = st.toString();
        if (stStr.isNotEmpty) _log('stacktrace: $stStr');
      }
      notifyListeners();
      return false;
    } finally {
      saving = false;
      _log('save() end saving=false msg="${(msg ?? '').trim()}"');
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
