// frontend\mall\lib\features\auth\presentation\page\billing_address.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/shell/presentation/components/header.dart';
import '../hook/use_billing_address.dart';

/// ✅ 請求先（決済）情報入力（雛形）
/// - クレジットカード番号
/// - 契約者名義
/// - 裏の3桁コード（CVC）
class BillingAddressPage extends StatefulWidget {
  const BillingAddressPage({super.key, this.from});

  /// optional back route
  final String? from;

  @override
  State<BillingAddressPage> createState() => _BillingAddressPageState();
}

class _BillingAddressPageState extends State<BillingAddressPage> {
  late final UseBillingAddress _vm;

  @override
  void initState() {
    super.initState();
    _vm = UseBillingAddress(from: widget.from);
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
    final backTo = _vm.backTo();

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            AppHeader(
              title: '請求先情報',
              showBack: true,
              backTo: backTo,
              actions: const [],
              onTapTitle: () => context.go('/'),
            ),
            Expanded(
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 520),
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        const _InfoBox(
                          kind: _InfoKind.info,
                          text:
                              'クレジットカード情報を入力してください。\n※ 実運用ではカード番号を自サーバーに保存/送信せず、決済SDKでトークン化します。',
                        ),
                        const SizedBox(height: 16),

                        Text(
                          'クレジットカード番号',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _vm.cardNumberCtrl,
                          keyboardType: TextInputType.number,
                          onChanged: (_) => _vm.onCardNumberChanged(),
                          decoration: const InputDecoration(
                            labelText: 'カード番号',
                            border: OutlineInputBorder(),
                            hintText: '例: 4242 4242 4242 4242',
                          ),
                        ),
                        const SizedBox(height: 16),

                        Text(
                          '契約者名義',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _vm.cardHolderCtrl,
                          textInputAction: TextInputAction.next,
                          onChanged: (_) => _vm.onFormChanged(),
                          decoration: const InputDecoration(
                            labelText: '名義（カードに記載の英字）',
                            border: OutlineInputBorder(),
                            hintText: '例: TARO YAMADA',
                          ),
                        ),
                        const SizedBox(height: 16),

                        Text(
                          '裏の3桁コード',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _vm.cvcCtrl,
                          keyboardType: TextInputType.number,
                          obscureText: true,
                          maxLength: 4, // AMEX 4桁も許容（vm 側でも許容）
                          onChanged: (_) => _vm.onFormChanged(),
                          decoration: const InputDecoration(
                            labelText: 'CVC',
                            border: OutlineInputBorder(),
                            hintText: '例: 123',
                            counterText: '',
                          ),
                        ),
                        const SizedBox(height: 20),

                        ElevatedButton(
                          onPressed: _vm.canSave
                              ? () => _vm.save(context)
                              : null,
                          child: _vm.saving
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Text('この請求情報を保存する'),
                        ),

                        if (_vm.msg != null) ...[
                          const SizedBox(height: 12),
                          _InfoBox(
                            kind: _vm.msg!.contains('保存しました')
                                ? _InfoKind.ok
                                : _InfoKind.error,
                            text: _vm.msg!,
                          ),
                        ],
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

enum _InfoKind { info, ok, error }

class _InfoBox extends StatelessWidget {
  const _InfoBox({required this.kind, required this.text});

  final _InfoKind kind;
  final String text;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    late final Color bg;
    switch (kind) {
      case _InfoKind.ok:
        bg = scheme.primaryContainer.withValues(alpha: 0.55);
        break;
      case _InfoKind.error:
        bg = scheme.errorContainer.withValues(alpha: 0.55);
        break;
      case _InfoKind.info:
        bg = scheme.surfaceContainerHighest.withValues(alpha: 0.55);
        break;
    }

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(text, style: Theme.of(context).textTheme.bodyMedium),
    );
  }
}
