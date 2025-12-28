// frontend/sns/lib/features/auth/presentation/page/create_account.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/shell/presentation/components/header.dart';
import '../hook/use_create_account.dart';

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
  late final UseCreateAccount _vm;

  @override
  void initState() {
    super.initState();
    _vm = UseCreateAccount(from: widget.from, intent: widget.intent);
    _vm.addListener(_onVmChanged);
  }

  void _onVmChanged() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    _vm.removeListener(_onVmChanged);
    _vm.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final topMessage = _vm.topMessage();

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            // ✅ ヘッダー：右側にサインインは出さない。戻るだけ。
            AppHeader(
              title: 'アカウント作成',
              showBack: true,
              backTo: _vm.loginBackTo(),
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
                        if (_vm.sentMessage != null) ...[
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
                              _vm.sentMessage!,
                              style: Theme.of(context).textTheme.bodyMedium,
                            ),
                          ),
                          const SizedBox(height: 12),
                        ],

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
                          autofillHints: const [AutofillHints.email],
                          enabled: !_vm.loading,
                          decoration: const InputDecoration(
                            labelText: 'メールアドレス',
                            border: OutlineInputBorder(),
                          ),
                          onChanged: (_) => _vm.onChanged(),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _vm.passCtrl,
                          obscureText: true,
                          autofillHints: const [AutofillHints.newPassword],
                          enabled: !_vm.loading,
                          decoration: const InputDecoration(
                            labelText: 'パスワード（6文字以上）',
                            border: OutlineInputBorder(),
                          ),
                          onChanged: (_) => _vm.onChanged(),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _vm.pass2Ctrl,
                          obscureText: true,
                          autofillHints: const [AutofillHints.newPassword],
                          enabled: !_vm.loading,
                          decoration: const InputDecoration(
                            labelText: 'パスワード（確認）',
                            border: OutlineInputBorder(),
                          ),
                          onChanged: (_) => _vm.onChanged(),
                        ),
                        const SizedBox(height: 12),

                        CheckboxListTile(
                          value: _vm.agree,
                          onChanged: _vm.loading
                              ? null
                              : (v) => _vm.setAgree(v ?? false),
                          controlAffinity: ListTileControlAffinity.leading,
                          contentPadding: EdgeInsets.zero,
                          title: const Text('利用規約に同意します'),
                        ),

                        const SizedBox(height: 12),

                        ElevatedButton(
                          onPressed: _vm.canSubmit
                              ? _vm.createAndSendVerification
                              : null,
                          child: _vm.loading
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
