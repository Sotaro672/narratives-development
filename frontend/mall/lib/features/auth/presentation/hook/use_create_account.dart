// frontend/sns/lib/features/auth/presentation/hook/use_create_account.dart
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
  String? error;

  /// ✅ 認証メール送信後に画面内へ表示するメッセージ
  String? sentMessage;

  @override
  void dispose() {
    emailCtrl.dispose();
    passCtrl.dispose();
    pass2Ctrl.dispose();
    super.dispose();
  }

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
    notifyListeners();
  }

  void setAgree(bool v) {
    agree = v;
    notifyListeners();
  }

  // ------------------------------------------------------------
  // action (delegate)
  // ------------------------------------------------------------

  Future<void> createAndSendVerification() async {
    error = null;
    sentMessage = null; // ✅ 再送などで古い成功メッセージを消す
    loading = true;
    notifyListeners();

    final res = await _service.createAndSendVerification(
      emailRaw: emailCtrl.text,
      pass: passCtrl.text,
      pass2: pass2Ctrl.text,
      agree: agree,
    );

    loading = false;
    if (res.ok) {
      sentMessage = res.sentMessage;
    } else {
      error = res.error;
    }
    notifyListeners();
  }
}
