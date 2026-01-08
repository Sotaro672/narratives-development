// frontend\mall\lib\features\auth\presentation\page\login_page.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/shell/presentation/components/header.dart';
import '../hook/use_login_page.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({super.key, this.from, this.intent});

  /// 任意: ログイン後に戻る先（例: /catalog/xxx）
  final String? from;

  /// 任意: リダイレクトされた理由（例: "purchase"）
  final String? intent;

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  late final UseLoginPage _vm;

  @override
  void initState() {
    super.initState();
    _vm = UseLoginPage(from: widget.from, intent: widget.intent);
    _vm.addListener(_onVmChanged);
  }

  void _onVmChanged() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    _vm.removeListener(_onVmChanged);
    _vm.disposeControllers();
    _vm.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final topMessage = _vm.topMessage();
    final backTo = _vm.backTo();

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            // ✅ Login では「戻る」だけ表示（Sign in は出さない）
            // ✅ ここが “Continue without login” の移譲先
            AppHeader(
              title: 'ログイン',
              showBack: true,
              backTo: backTo,
              actions: const [], // ← ここ重要（右側ボタン非表示）
              onTapTitle: () => context.go('/'),
            ),
            Expanded(
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 420),
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Text(
                          'sns',
                          textAlign: TextAlign.center,
                          style: Theme.of(context).textTheme.headlineSmall,
                        ),
                        const SizedBox(height: 6),
                        Text(
                          topMessage,
                          textAlign: TextAlign.center,
                          style: Theme.of(context).textTheme.bodyMedium,
                        ),
                        const SizedBox(height: 16),
                        if (_vm.error != null) ...[
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
                              _vm.error!,
                              style: Theme.of(context).textTheme.bodyMedium,
                            ),
                          ),
                          const SizedBox(height: 12),
                        ],
                        TextField(
                          controller: _vm.emailCtrl,
                          keyboardType: TextInputType.emailAddress,
                          autofillHints: const [
                            AutofillHints.username,
                            AutofillHints.email,
                          ],
                          enabled: !_vm.loading,
                          decoration: const InputDecoration(
                            labelText: 'メールアドレス',
                            border: OutlineInputBorder(),
                          ),
                          onSubmitted: (_) => _vm.signIn(context),
                        ),
                        const SizedBox(height: 12),
                        TextField(
                          controller: _vm.passCtrl,
                          obscureText: true,
                          autofillHints: const [AutofillHints.password],
                          enabled: !_vm.loading,
                          decoration: const InputDecoration(
                            labelText: 'パスワード',
                            border: OutlineInputBorder(),
                          ),
                          onSubmitted: (_) => _vm.signIn(context),
                        ),
                        const SizedBox(height: 12),
                        ElevatedButton(
                          onPressed: _vm.loading
                              ? null
                              : () => _vm.signIn(context),
                          child: _vm.loading
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Text('ログイン'),
                        ),
                        const SizedBox(height: 8),

                        // ✅ Create account は “作成ページへ遷移” に変更
                        OutlinedButton(
                          onPressed: _vm.loading
                              ? null
                              : () => _vm.goCreateAccount(context),
                          child: const Text('アカウントを作成'),
                        ),
                        const SizedBox(height: 8),
                        TextButton(
                          onPressed: _vm.loading
                              ? null
                              : () => _vm.sendPasswordReset(context),
                          child: const Text('パスワードをお忘れですか？'),
                        ),

                        // ✅ Continue without login は削除（戻るボタンへ移譲済み）
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
