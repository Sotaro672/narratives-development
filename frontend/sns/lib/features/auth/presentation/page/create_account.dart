//frontend\sns\lib\features\auth\presentation\page\create_account.dart
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

  String _afterSuccessDest() {
    final dest = (widget.from ?? '/').trim();
    return dest.isNotEmpty ? dest : '/';
  }

  Future<void> _createAndSendVerification() async {
    setState(() => _error = null);

    final email = _s(_emailCtrl.text);
    final pass = _passCtrl.text;

    if (!_isEmailValid) {
      setState(() => _error = 'Enter a valid email address.');
      return;
    }
    if (!_isPasswordValid) {
      setState(() => _error = 'Password must be at least 6 characters.');
      return;
    }
    if (!_isPasswordMatch) {
      setState(() => _error = 'Passwords do not match.');
      return;
    }
    if (!_agree) {
      setState(() => _error = 'Please accept the Terms.');
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
        throw StateError('User is null after sign up.');
      }

      // ✅ 認証メール送信
      await user.sendEmailVerification();

      if (!mounted) return;

      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Verification email sent. Please check your inbox.'),
        ),
      );

      // ✅ そのまま閲覧に戻す（必要なら後で「未認証は購入不可」等の制御を追加）
      context.go(_afterSuccessDest());
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
        return 'Invalid email address.';
      case 'email-already-in-use':
        return 'Email is already in use.';
      case 'weak-password':
        return 'Password is too weak.';
      case 'operation-not-allowed':
        return 'This sign-in method is not enabled.';
      default:
        return e.message ?? 'Create account failed.';
    }
  }

  @override
  Widget build(BuildContext context) {
    final intent = (widget.intent ?? '').trim();
    final topMessage = intent == 'purchase'
        ? 'Create an account to continue your purchase.'
        : 'Create an account to continue.';

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            // ✅ ヘッダー：右側に Sign in は出さない。戻るだけ。
            AppHeader(
              title: 'Create account',
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
                            labelText: 'Email',
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
                            labelText: 'Password (min 6 chars)',
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
                            labelText: 'Confirm password',
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
                          title: const Text('I agree to the Terms of Service'),
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
                              : const Text('Send verification email'),
                        ),

                        const SizedBox(height: 8),

                        TextButton(
                          onPressed: _loading
                              ? null
                              : () => context.go(_loginBackTo()),
                          child: const Text('Back to Sign in'),
                        ),
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
