// frontend/sns/lib/features/auth/presentation/page/billing_address.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/shell/presentation/components/header.dart';

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
  final _cardNumberCtrl = TextEditingController();
  final _cardHolderCtrl = TextEditingController();
  final _cvcCtrl = TextEditingController();

  bool _saving = false;
  String? _msg;

  @override
  void dispose() {
    _cardNumberCtrl.dispose();
    _cardHolderCtrl.dispose();
    _cvcCtrl.dispose();
    super.dispose();
  }

  String _s(String? v) => (v ?? '').trim();

  String _backTo() {
    final from = _s(widget.from);
    if (from.isNotEmpty) return from;
    return '/shipping-address';
  }

  bool get _canSave {
    if (_saving) return false;

    final card = _normalizeCardNumber(_cardNumberCtrl.text);
    final holder = _s(_cardHolderCtrl.text);
    final cvc = _normalizeDigits(_cvcCtrl.text);

    // 雛形: ざっくり必須チェック（Luhn等は後で）
    if (card.length < 12) return false; // AMEX等も考慮し ">=12" 程度に
    if (holder.isEmpty) return false;
    if (cvc.length != 3) return false;
    return true;
  }

  String _normalizeDigits(String s) => s.replaceAll(RegExp(r'[^0-9]'), '');

  String _normalizeCardNumber(String s) {
    // ハイフン/スペースを除去
    return s.replaceAll(RegExp(r'[^0-9]'), '');
  }

  String _formatCardNumberForDisplay(String s) {
    final digits = _normalizeCardNumber(s);
    final buf = StringBuffer();
    for (var i = 0; i < digits.length; i++) {
      if (i != 0 && i % 4 == 0) buf.write(' ');
      buf.write(digits[i]);
    }
    return buf.toString();
  }

  void _onCardNumberChanged() {
    // 入力中の見た目を整える（強制しすぎない程度）
    final current = _cardNumberCtrl.text;
    final formatted = _formatCardNumberForDisplay(current);
    if (formatted == current) return;

    final sel = _cardNumberCtrl.selection;
    _cardNumberCtrl.value = TextEditingValue(
      text: formatted,
      selection: TextSelection.collapsed(
        offset: (sel.baseOffset + (formatted.length - current.length)).clamp(
          0,
          formatted.length,
        ),
      ),
    );
  }

  Future<void> _saveDummy() async {
    if (mounted) {
      setState(() {
        _saving = true;
        _msg = null;
      });
    }

    try {
      await Future<void>.delayed(const Duration(milliseconds: 600));

      if (!mounted) return;
      setState(() {
        _msg = '請求情報を保存しました（ダミー）。';
      });

      // ✅ 保存後に avatar_create へ遷移
      context.go('/avatar-create');
    } catch (e) {
      if (mounted) {
        setState(() {
          _msg = e.toString();
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _saving = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final backTo = _backTo();

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
                          controller: _cardNumberCtrl,
                          keyboardType: TextInputType.number,
                          onChanged: (_) => _onCardNumberChanged(),
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
                          controller: _cardHolderCtrl,
                          textInputAction: TextInputAction.next,
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
                          controller: _cvcCtrl,
                          keyboardType: TextInputType.number,
                          obscureText: true,
                          maxLength: 3,
                          decoration: const InputDecoration(
                            labelText: 'CVC',
                            border: OutlineInputBorder(),
                            hintText: '例: 123',
                            counterText: '',
                          ),
                        ),
                        const SizedBox(height: 20),

                        ElevatedButton(
                          onPressed: _canSave ? _saveDummy : null,
                          child: _saving
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Text('この請求情報を保存する'),
                        ),

                        if (_msg != null) ...[
                          const SizedBox(height: 12),
                          _InfoBox(
                            kind: _msg!.contains('保存しました')
                                ? _InfoKind.ok
                                : _InfoKind.error,
                            text: _msg!,
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
