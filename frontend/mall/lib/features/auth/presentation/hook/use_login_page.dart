// frontend\mall\lib\features\auth\presentation\hook\use_login_page.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';

/// LoginPage 用の状態/処理をまとめる Hook 風クラス（Flutter では ChangeNotifier で実現）
class UseLoginPage extends ChangeNotifier {
  UseLoginPage({required this.from, required this.intent});

  final String? from;
  final String? intent;

  final emailCtrl = TextEditingController();
  final passCtrl = TextEditingController();

  bool loading = false;
  String? error;

  void disposeControllers() {
    emailCtrl.dispose();
    passCtrl.dispose();
  }

  String _s(String v) => v.trim();

  String backTo() {
    final dest = (from ?? '/').trim();
    return dest.isNotEmpty ? dest : '/';
  }

  String topMessage() {
    final it = (intent ?? '').trim();
    return it == 'purchase' ? '購入を完了するにはログインしてください。' : '続行するにはログインしてください。';
  }

  Future<void> signIn(BuildContext context) async {
    final email = _s(emailCtrl.text);
    final pass = passCtrl.text;

    error = null;
    notifyListeners();

    if (email.isEmpty || pass.isEmpty) {
      error = 'メールアドレスとパスワードは必須です。';
      notifyListeners();
      return;
    }

    loading = true;
    notifyListeners();

    try {
      await FirebaseAuth.instance.signInWithEmailAndPassword(
        email: email,
        password: pass,
      );

      final dest = backTo();
      if (!context.mounted) return;
      context.go(dest);
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

  void goCreateAccount(BuildContext context) {
    final from = backTo();
    final it = (intent ?? '').trim();

    final qp = <String, String>{};
    if (from.trim().isNotEmpty) qp['from'] = from;
    if (it.isNotEmpty) qp['intent'] = it;

    final uri = Uri(
      path: '/create-account',
      queryParameters: qp.isEmpty ? null : qp,
    );

    context.go(uri.toString());
  }

  Future<void> sendPasswordReset(BuildContext context) async {
    final email = _s(emailCtrl.text);

    error = null;
    notifyListeners();

    if (email.isEmpty) {
      error = 'パスワードをリセットするにはメールアドレスを入力してください。';
      notifyListeners();
      return;
    }

    loading = true;
    notifyListeners();

    try {
      await FirebaseAuth.instance.sendPasswordResetEmail(email: email);
      if (!context.mounted) return;

      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('パスワードリセット用のメールを送信しました。')));
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
      case 'user-disabled':
        return 'このアカウントは無効化されています。';
      case 'user-not-found':
        return 'アカウントが見つかりません。';
      case 'wrong-password':
        return 'パスワードが間違っています。';
      case 'email-already-in-use':
        return 'このメールアドレスは既に使用されています。';
      case 'weak-password':
        return 'パスワードが弱すぎます。';
      case 'too-many-requests':
        return '試行回数が多すぎます。しばらくしてからお試しください。';
      case 'operation-not-allowed':
        return 'このログイン方法は有効化されていません。';
      default:
        return e.message ?? 'ログインに失敗しました。';
    }
  }
}
