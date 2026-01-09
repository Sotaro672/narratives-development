// frontend/mall/lib/features/auth/application/create_account_service.dart
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/foundation.dart';

/// Create account flow (Mall):
/// 1) Firebase Auth: createUserWithEmailAndPassword
/// 2) Firebase Auth: sendEmailVerification
///
/// ✅ IMPORTANT:
/// - This service does NOT call backend.
/// - It only sends verification email.
class CreateAccountService {
  CreateAccountService({required FirebaseAuth auth}) : _auth = auth;

  final FirebaseAuth _auth;

  bool _disposed = false;

  void dispose() {
    _disposed = true;
  }

  // ------------------------------------------------------------
  // Validation helpers (used by UI hook)
  // ------------------------------------------------------------

  bool isEmailValid(String raw) {
    final v = raw.trim();
    if (v.isEmpty) return false;
    final re = RegExp(r'^[^@\s]+@[^@\s]+\.[^@\s]+$');
    return re.hasMatch(v);
  }

  bool isPasswordValid(String raw) {
    // 최소 요건: 8 chars
    return raw.trim().length >= 8;
  }

  bool isPasswordMatch(String p1, String p2) => p1 == p2;

  String loginBackTo({String? from, String? intent}) {
    final f = (from ?? '').trim();
    final i = (intent ?? '').trim();
    if (f.isEmpty && i.isEmpty) return '/login';
    final qp = <String, String>{};
    if (f.isNotEmpty) qp['from'] = f;
    if (i.isNotEmpty) qp['intent'] = i;
    final q = Uri(queryParameters: qp).query;
    return q.isEmpty ? '/login' : '/login?$q';
  }

  String topMessage({String? intent}) {
    final i = (intent ?? '').trim();
    if (i.isEmpty) return 'アカウントを作成してください。';
    if (i == 'purchase') return '購入を続けるためにアカウント作成が必要です。';
    return 'アカウントを作成してください。';
  }

  // ------------------------------------------------------------
  // Action (used by UI hook)
  // ------------------------------------------------------------

  Future<CreateAccountResult> createAndSendVerification({
    required String emailRaw,
    required String pass,
    required String pass2,
    required bool agree,
    String? displayName,
  }) async {
    _ensureNotDisposed();

    final email = emailRaw.trim();

    if (!agree) {
      return const CreateAccountResult(ok: false, error: '利用規約への同意が必要です。');
    }
    if (!isEmailValid(email)) {
      return const CreateAccountResult(
        ok: false,
        error: 'メールアドレスの形式が正しくありません。',
      );
    }
    if (!isPasswordValid(pass)) {
      return const CreateAccountResult(ok: false, error: 'パスワードは8文字以上にしてください。');
    }
    if (!isPasswordMatch(pass, pass2)) {
      return const CreateAccountResult(ok: false, error: 'パスワードが一致しません。');
    }

    try {
      await _createAccountAndSendEmailOnly(
        email: email,
        password: pass,
        displayName: displayName?.trim(),
      );

      return const CreateAccountResult(
        ok: true,
        sentMessage: '認証メールを送信しました。受信箱（迷惑メール含む）をご確認ください。',
      );
    } on FirebaseAuthException catch (e) {
      // ✅ デバッグしやすいように code をログに残す
      debugPrint(
        '[CreateAccountService] FirebaseAuthException ${e.code}: ${e.message}',
      );
      return CreateAccountResult(ok: false, error: _friendlyAuthError(e));
    } catch (e) {
      debugPrint('[CreateAccountService] Unknown error: $e');
      return CreateAccountResult(ok: false, error: '不明なエラーが発生しました: $e');
    }
  }

  // ------------------------------------------------------------
  // Internal
  // ------------------------------------------------------------

  Future<void> _createAccountAndSendEmailOnly({
    required String email,
    required String password,
    String? displayName,
  }) async {
    _ensureNotDisposed();

    // 1) Firebase create user (this also signs-in the user)
    final cred = await _auth.createUserWithEmailAndPassword(
      email: email,
      password: password,
    );

    final user = cred.user;
    if (user == null) {
      throw StateError('FirebaseAuth returned null user');
    }

    // optional displayName (best-effort)
    final name = (displayName ?? '').trim();
    if (name.isNotEmpty) {
      try {
        await user.updateDisplayName(name);
      } catch (_) {}
    }

    // 2) Send verification email
    await user.sendEmailVerification();
  }

  void _ensureNotDisposed() {
    if (_disposed) {
      throw StateError('CreateAccountService is disposed');
    }
  }

  static String _friendlyAuthError(FirebaseAuthException e) {
    switch (e.code) {
      case 'email-already-in-use':
        return 'このメールアドレスは既に登録されています。ログインをお試しください。';
      case 'invalid-email':
        return 'メールアドレスの形式が正しくありません。';
      case 'weak-password':
        return 'パスワードが弱すぎます。もう少し複雑にしてください。';
      case 'operation-not-allowed':
        return 'このサインアップ方法は現在利用できません（Firebase側でEmail/Passwordが無効の可能性）。';
      case 'network-request-failed':
        return 'ネットワークに接続できませんでした。通信状況をご確認ください。';
      default:
        // ✅ 調査しやすいように code を表示
        return '認証エラーが発生しました（${e.code}）。';
    }
  }
}

class CreateAccountResult {
  const CreateAccountResult({required this.ok, this.sentMessage, this.error});

  final bool ok;
  final String? sentMessage;
  final String? error;
}
