// frontend/sns/lib/features/auth/presentation/hook/use_create_account.dart
import 'package:flutter/material.dart';
import 'package:firebase_auth/firebase_auth.dart';

/// CreateAccountPage の状態/処理をまとめる（Flutter では ChangeNotifier で hook 風にする）
class UseCreateAccount extends ChangeNotifier {
  UseCreateAccount({required this.from, required this.intent});

  /// Optional: where to go after signup (e.g. /catalog/xxx)
  final String? from;

  /// Optional: why user was redirected (e.g. "purchase")
  final String? intent;

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

  String _s(String v) => v.trim();

  // ------------------------------------------------------------
  // computed
  // ------------------------------------------------------------

  bool get isEmailValid {
    final email = _s(emailCtrl.text);
    // ざっくり判定（厳密でなくてOK）
    return email.isNotEmpty && email.contains('@') && email.contains('.');
  }

  bool get isPasswordValid {
    final pass = passCtrl.text;
    return pass.length >= 6;
  }

  bool get isPasswordMatch {
    return passCtrl.text == pass2Ctrl.text && pass2Ctrl.text.isNotEmpty;
  }

  bool get canSubmit {
    return !loading &&
        agree &&
        isEmailValid &&
        isPasswordValid &&
        isPasswordMatch;
  }

  String loginBackTo() {
    final qp = <String, String>{};
    final f = (from ?? '').trim();
    final it = (intent ?? '').trim();
    if (f.isNotEmpty) qp['from'] = f;
    if (it.isNotEmpty) qp['intent'] = it;
    final uri = Uri(path: '/login', queryParameters: qp.isEmpty ? null : qp);
    return uri.toString();
  }

  String topMessage() {
    final it = (intent ?? '').trim();
    return it == 'purchase' ? '購入を続けるにはアカウント作成が必要です。' : '続けるにはアカウント作成が必要です。';
  }

  void onChanged() {
    // TextField の onChanged から呼ぶ（UI 側の setState を不要にする）
    notifyListeners();
  }

  void setAgree(bool v) {
    agree = v;
    notifyListeners();
  }

  // ------------------------------------------------------------
  // actions
  // ------------------------------------------------------------

  Future<void> createAndSendVerification() async {
    error = null;
    sentMessage = null; // ✅ 再送などで古い成功メッセージを消す
    notifyListeners();

    final email = _s(emailCtrl.text);
    final pass = passCtrl.text;

    if (!isEmailValid) {
      error = '有効なメールアドレスを入力してください。';
      notifyListeners();
      return;
    }
    if (!isPasswordValid) {
      error = 'パスワードは6文字以上にしてください。';
      notifyListeners();
      return;
    }
    if (!isPasswordMatch) {
      error = 'パスワードが一致しません。';
      notifyListeners();
      return;
    }
    if (!agree) {
      error = '利用規約に同意してください。';
      notifyListeners();
      return;
    }

    loading = true;
    notifyListeners();

    try {
      final cred = await FirebaseAuth.instance.createUserWithEmailAndPassword(
        email: email,
        password: pass,
      );

      final user = cred.user ?? FirebaseAuth.instance.currentUser;
      if (user == null) {
        throw StateError('アカウント作成後にユーザー情報が取得できませんでした。');
      }

      // ✅ 認証メール送信
      await user.sendEmailVerification();

      // ✅ 画面内に成功メッセージを表示（SnackBar ではなく “画面へ表示”）
      sentMessage =
          '認証メールを送信しました。受信ボックスを確認してください。\n'
          '認証メールからアカウント作成を続行してください。';
      notifyListeners();

      // ✅ ここでは遷移しない（ユーザーがメッセージを確認できるように）
    } on FirebaseAuthException catch (e) {
      error = _friendlyAuthError(e);
      notifyListeners();
    } catch (e) {
      error = e.toString();
      notifyListeners();
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  String _friendlyAuthError(FirebaseAuthException e) {
    switch (e.code) {
      case 'invalid-email':
        return 'メールアドレスの形式が正しくありません。';
      case 'email-already-in-use':
        return 'このメールアドレスは既に使用されています。';
      case 'weak-password':
        return 'パスワードが弱すぎます。';
      case 'operation-not-allowed':
        return 'このログイン方法は有効化されていません。';
      default:
        return e.message ?? 'アカウント作成に失敗しました。';
    }
  }
}
