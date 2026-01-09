// frontend\mall\lib\features\auth\presentation\hook\use_create_account.dart
import 'package:flutter/material.dart';

import '../../application/create_account_service.dart';

/// CreateAccountPage の状態/処理をまとめる（Flutter では ChangeNotifier で hook 風にする）
class UseCreateAccount extends ChangeNotifier {
  UseCreateAccount({
    required this.from,
    required this.intent,
    CreateAccountService? service,
  }) : _service = service ?? CreateAccountService();

  /// Optional: where to go after signup (e.g. /catalog/xxx)
  final String? from;

  /// Optional: why user was redirected (e.g. "purchase")
  final String? intent;

  final CreateAccountService _service;

  final emailCtrl = TextEditingController();
  final passCtrl = TextEditingController();
  final pass2Ctrl = TextEditingController();

  bool agree = false;
  bool loading = false;

  /// ✅ エラー表示（UIへ必ず出す）
  String? error;

  /// ✅ 認証メール送信後に画面内へ表示するメッセージ
  String? sentMessage;

  // ------------------------------------------------------------
  // computed (delegate)
  // ------------------------------------------------------------

  bool get isEmailValid => _service.isEmailValid(emailCtrl.text);

  bool get isPasswordValid => _service.isPasswordValid(passCtrl.text);

  bool get isPasswordMatch =>
      _service.isPasswordMatch(passCtrl.text, pass2Ctrl.text);

  bool get canSubmit =>
      !loading && agree && isEmailValid && isPasswordValid && isPasswordMatch;

  String loginBackTo() => _service.loginBackTo(from: from, intent: intent);

  String topMessage() => _service.topMessage(intent: intent);

  void onChanged() {
    // ✅ 入力変更時に古いメッセージを消して混乱を防ぐ
    if (error != null || sentMessage != null) {
      error = null;
      sentMessage = null;
    }
    _safeNotify();
  }

  void setAgree(bool v) {
    agree = v;
    // ✅ チェック変更時も古いメッセージを消す
    if (error != null || sentMessage != null) {
      error = null;
      sentMessage = null;
    }
    _safeNotify();
  }

  // ------------------------------------------------------------
  // action (delegate)
  // ------------------------------------------------------------

  Future<void> createAndSendVerification() async {
    // ✅ 二重送信防止（UIのボタンdisableに加え、ロジックでもガード）
    if (loading) return;

    error = null;
    sentMessage = null;
    loading = true;
    _safeNotify();

    bool finished = false;

    try {
      final res = await _service.createAndSendVerification(
        emailRaw: emailCtrl.text,
        pass: passCtrl.text,
        pass2: pass2Ctrl.text,
        agree: agree,
      );

      if (_disposed) return;

      if (res.ok) {
        sentMessage = res.sentMessage;
      } else {
        error = res.error ?? '不明なエラーが発生しました。';
      }
      finished = true;
    } catch (e) {
      if (_disposed) return;
      error = '不明なエラー: $e';
      finished = true;
    } finally {
      // ❌ finally で return しない（lint回避）
      if (!_disposed) {
        loading = false;
        _safeNotify();
      } else {
        // disposed 済みなら何もしない
      }
    }

    // finished はデバッグ用途の名残（将来ログに使うなら残せる）
    // ignore: unused_local_variable
    finished = finished;
  }

  // ------------------------------------------------------------
  // lifecycle guard
  // ------------------------------------------------------------

  bool _disposed = false;

  void _safeNotify() {
    if (_disposed) return;
    notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    emailCtrl.dispose();
    passCtrl.dispose();
    pass2Ctrl.dispose();
    super.dispose();
  }
}
