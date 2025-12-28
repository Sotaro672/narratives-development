// frontend/sns/lib/features/auth/application/create_account_service.dart
import 'package:firebase_auth/firebase_auth.dart';

class CreateAccountResult {
  const CreateAccountResult.success(this.sentMessage) : ok = true, error = null;

  const CreateAccountResult.failure(this.error)
    : ok = false,
      sentMessage = null;

  final bool ok;
  final String? sentMessage;
  final String? error;
}

class CreateAccountService {
  CreateAccountService({FirebaseAuth? auth})
    : _auth = auth ?? FirebaseAuth.instance;

  final FirebaseAuth _auth;

  String s(String v) => v.trim();

  // ------------------------------------------------------------
  // validation helpers
  // ------------------------------------------------------------

  bool isEmailValid(String emailRaw) {
    final email = s(emailRaw);
    // ざっくり判定（厳密でなくてOK）
    return email.isNotEmpty && email.contains('@') && email.contains('.');
  }

  bool isPasswordValid(String pass) => pass.length >= 6;

  bool isPasswordMatch(String pass1, String pass2) =>
      pass1 == pass2 && pass2.isNotEmpty;

  String topMessage({required String? intent}) {
    final it = (intent ?? '').trim();
    return it == 'purchase' ? '購入を続けるにはアカウント作成が必要です。' : '続けるにはアカウント作成が必要です。';
  }

  String loginBackTo({required String? from, required String? intent}) {
    final qp = <String, String>{};
    final f = (from ?? '').trim();
    final it = (intent ?? '').trim();
    if (f.isNotEmpty) qp['from'] = f;
    if (it.isNotEmpty) qp['intent'] = it;
    final uri = Uri(path: '/login', queryParameters: qp.isEmpty ? null : qp);
    return uri.toString();
  }

  // ------------------------------------------------------------
  // main action
  // ------------------------------------------------------------

  Future<CreateAccountResult> createAndSendVerification({
    required String emailRaw,
    required String pass,
    required String pass2,
    required bool agree,
  }) async {
    final email = s(emailRaw);

    if (!isEmailValid(email)) {
      return const CreateAccountResult.failure('有効なメールアドレスを入力してください。');
    }
    if (!isPasswordValid(pass)) {
      return const CreateAccountResult.failure('パスワードは6文字以上にしてください。');
    }
    if (!isPasswordMatch(pass, pass2)) {
      return const CreateAccountResult.failure('パスワードが一致しません。');
    }
    if (!agree) {
      return const CreateAccountResult.failure('利用規約に同意してください。');
    }

    try {
      final cred = await _auth.createUserWithEmailAndPassword(
        email: email,
        password: pass,
      );

      final user = cred.user ?? _auth.currentUser;
      if (user == null) {
        return const CreateAccountResult.failure('アカウント作成後にユーザー情報が取得できませんでした。');
      }

      // ✅ 認証メール送信
      await user.sendEmailVerification();

      return const CreateAccountResult.success(
        '認証メールを送信しました。受信ボックスを確認してください。\n'
        '認証メールからアカウント作成を続行してください。',
      );
    } on FirebaseAuthException catch (e) {
      return CreateAccountResult.failure(_friendlyAuthError(e));
    } catch (e) {
      return CreateAccountResult.failure(e.toString());
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
