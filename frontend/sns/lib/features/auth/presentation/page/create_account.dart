//frontend/sns/lib/features/auth/presentation/page/create_account.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../../../../app/shell/presentation/components/header.dart';

class CreateAccountPage extends StatefulWidget {
  const CreateAccountPage({super.key, this.from, this.intent});

  /// Optional: where to go after signup (e.g. /catalog/xxx)
  final String? from;

  /// Optional: why user was redirected (e.g. "purchase")
  final String? intent;

  @override
  State<CreateAccountPage> createState() => _CreateAccountPageState();
}

class _CreateAccountPageState extends State<CreateAccountPage> {
  final _emailCtrl = TextEditingController();
  final _passCtrl = TextEditingController();
  final _pass2Ctrl = TextEditingController();

  bool _agree = false;
  bool _loading = false;
  String? _error;

  /// ✅ 認証メール送信後に画面内へ表示するメッセージ
  String? _sentMessage;

  @override
  void dispose() {
    _emailCtrl.dispose();
    _passCtrl.dispose();
    _pass2Ctrl.dispose();
    super.dispose();
  }

  String _s(String v) => v.trim();

  bool get _isEmailValid {
    final email = _s(_emailCtrl.text);
    // ざっくり判定（厳密でなくてOK）
    return email.isNotEmpty && email.contains('@') && email.contains('.');
  }

  bool get _isPasswordValid {
    final pass = _passCtrl.text;
    return pass.length >= 6;
  }

  bool get _isPasswordMatch {
    return _passCtrl.text == _pass2Ctrl.text && _pass2Ctrl.text.isNotEmpty;
  }

  bool get _canSubmit {
    return !_loading &&
        _agree &&
        _isEmailValid &&
        _isPasswordValid &&
        _isPasswordMatch;
  }

  String _loginBackTo() {
    final qp = <String, String>{};
    final from = (widget.from ?? '').trim();
    final intent = (widget.intent ?? '').trim();
    if (from.isNotEmpty) qp['from'] = from;
    if (intent.isNotEmpty) qp['intent'] = intent;
    final uri = Uri(path: '/login', queryParameters: qp.isEmpty ? null : qp);
    return uri.toString();
  }

  Future<void> _createAndSendVerification() async {
    setState(() {
      _error = null;
      _sentMessage = null; // ✅ 再送などで古い成功メッセージを消す
    });

    final email = _s(_emailCtrl.text);
    final pass = _passCtrl.text;

    if (!_isEmailValid) {
      setState(() => _error = '有効なメールアドレスを入力してください。');
      return;
    }
    if (!_isPasswordValid) {
      setState(() => _error = 'パスワードは6文字以上にしてください。');
      return;
    }
    if (!_isPasswordMatch) {
      setState(() => _error = 'パスワードが一致しません。');
      return;
    }
    if (!_agree) {
      setState(() => _error = '利用規約に同意してください。');
      return;
    }

    setState(() => _loading = true);

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

      if (!mounted) return;

      // ✅ 画面内に成功メッセージを表示（SnackBar ではなく “画面へ表示”）
      setState(() {
        _sentMessage =
            '認証メールを送信しました。受信ボックスを確認してください。\n'
            '認証メールからアカウント作成を続行してください。';
      });

      // ✅ ここでは遷移しない（ユーザーがメッセージを確認できるように）
    } on FirebaseAuthException catch (e) {
      setState(() => _error = _friendlyAuthError(e));
    } catch (e) {
      setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _loading = false);
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

  @override
  Widget build(BuildContext context) {
    final intent = (widget.intent ?? '').trim();
    final topMessage = intent == 'purchase'
        ? '購入を続けるにはアカウント作成が必要です。'
        : '続けるにはアカウント作成が必要です。';

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            // ✅ ヘッダー：右側にサインインは出さない。戻るだけ。
            AppHeader(
              title: 'アカウント作成',
              showBack: true,
              backTo: _loginBackTo(),
              actions: const [],
              onTapTitle: () => context.go('/'),
            ),

            Expanded(
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 460),
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Text(
                          topMessage,
                          textAlign: TextAlign.center,
                          style: Theme.of(context).textTheme.bodyMedium,
                        ),
                        const SizedBox(height: 16),

                        // ✅ 成功メッセージ（画面へ表示）
                        if (_sentMessage != null) ...[
                          Container(
                            padding: const EdgeInsets.all(12),
                            decoration: BoxDecoration(
                              color: Theme.of(context)
                                  .colorScheme
                                  .secondaryContainer
                                  .withValues(alpha: 0.6),
                              borderRadius: BorderRadius.circular(12),
                            ),
                            child: Text(
                              _sentMessage!,
                              style: Theme.of(context).textTheme.bodyMedium,
                            ),
                          ),
                          const SizedBox(height: 12),
                        ],

                        if (_error != null) ...[
                          Container(
                            padding: const EdgeInsets.all(12),
                            decoration: BoxDecoration(
                              color: Theme.of(context)
                                  .colorScheme
                                  .errorContainer
                                  .withValues(alpha: 0.6),
                              borderRadius: BorderRadius.circular(12),
                            ),
                            child: Text(
                              _error!,
                              style: Theme.of(context).textTheme.bodyMedium,
                            ),
                          ),
                          const SizedBox(height: 12),
                        ],

                        TextField(
                          controller: _emailCtrl,
                          keyboardType: TextInputType.emailAddress,
                          autofillHints: const [AutofillHints.email],
                          enabled: !_loading,
                          decoration: const InputDecoration(
                            labelText: 'メールアドレス',
                            border: OutlineInputBorder(),
                          ),
                          onChanged: (_) => setState(() {}),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _passCtrl,
                          obscureText: true,
                          autofillHints: const [AutofillHints.newPassword],
                          enabled: !_loading,
                          decoration: const InputDecoration(
                            labelText: 'パスワード（6文字以上）',
                            border: OutlineInputBorder(),
                          ),
                          onChanged: (_) => setState(() {}),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _pass2Ctrl,
                          obscureText: true,
                          autofillHints: const [AutofillHints.newPassword],
                          enabled: !_loading,
                          decoration: const InputDecoration(
                            labelText: 'パスワード（確認）',
                            border: OutlineInputBorder(),
                          ),
                          onChanged: (_) => setState(() {}),
                        ),
                        const SizedBox(height: 12),

                        CheckboxListTile(
                          value: _agree,
                          onChanged: _loading
                              ? null
                              : (v) => setState(() => _agree = v ?? false),
                          controlAffinity: ListTileControlAffinity.leading,
                          contentPadding: EdgeInsets.zero,
                          title: const Text('利用規約に同意します'),
                        ),

                        const SizedBox(height: 12),

                        ElevatedButton(
                          onPressed: _canSubmit
                              ? _createAndSendVerification
                              : null,
                          child: _loading
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Text('認証メールを送信'),
                        ),

                        // ✅ 「サインインに戻る」ボタンは削除（戻るはヘッダーに集約）
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
