// frontend/sns/lib/features/auth/presentation/page/shipping_address.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/shell/presentation/components/header.dart';
import '../hook/use_shipping_address.dart';

/// ✅ 認証メールリンクの着地点 + 配送先住所入力
class ShippingAddressPage extends StatefulWidget {
  const ShippingAddressPage({
    super.key,
    this.mode,
    this.oobCode,
    this.continueUrl,
    this.lang,
    this.from,
    this.intent,
  });

  /// Firebase action params
  final String? mode; // e.g. verifyEmail
  final String? oobCode;
  final String? continueUrl;
  final String? lang;

  /// optional app params
  final String? from;
  final String? intent;

  @override
  State<ShippingAddressPage> createState() => _ShippingAddressPageState();
}

class _ShippingAddressPageState extends State<ShippingAddressPage> {
  late final UseShippingAddress _vm;

  @override
  void initState() {
    super.initState();
    _vm = UseShippingAddress(
      mode: widget.mode,
      oobCode: widget.oobCode,
      continueUrl: widget.continueUrl,
      lang: widget.lang,
      from: widget.from,
      intent: widget.intent,
    );
    _vm.init();
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

    final loggedIn = _vm.loggedIn;
    final emailVerified = _vm.emailVerified;

    final cameFromEmailLink = _vm.cameFromEmailLink;

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            AppHeader(
              title: '配送先住所',
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
                        if (cameFromEmailLink) ...[
                          Text(
                            'メール認証の確認',
                            style: Theme.of(context).textTheme.titleLarge,
                          ),
                          const SizedBox(height: 8),
                          if (_vm.verifying) ...[
                            const Text('確認中です…'),
                            const SizedBox(height: 12),
                            const LinearProgressIndicator(),
                            const SizedBox(height: 16),
                          ] else if (_vm.verifyError != null) ...[
                            _InfoBox(
                              kind: _InfoKind.error,
                              text: _vm.verifyError!,
                            ),
                            const SizedBox(height: 12),
                          ] else if (_vm.verified) ...[
                            const _InfoBox(
                              kind: _InfoKind.ok,
                              text: 'メール認証が完了しました。続けて配送先情報を入力してください。',
                            ),
                            const SizedBox(height: 12),
                          ] else ...[
                            const _InfoBox(
                              kind: _InfoKind.info,
                              text: '認証リンクを確認します。',
                            ),
                            const SizedBox(height: 12),
                          ],
                        ] else ...[
                          const _InfoBox(
                            kind: _InfoKind.info,
                            text: '配送先情報を入力してください。',
                          ),
                          const SizedBox(height: 12),
                        ],

                        if (!loggedIn) ...[
                          const _InfoBox(
                            kind: _InfoKind.info,
                            text: '※ 住所の保存にはサインインが必要です。',
                          ),
                          const SizedBox(height: 8),
                          OutlinedButton(
                            onPressed: () => _vm.goSignIn(context),
                            child: const Text('サインインへ'),
                          ),
                          const SizedBox(height: 16),
                        ] else if (!emailVerified) ...[
                          const _InfoBox(
                            kind: _InfoKind.info,
                            text: '※ 現在サインイン中ですが、メール認証が未完了の可能性があります。',
                          ),
                          const SizedBox(height: 16),
                        ],

                        Text(
                          'お届け先氏名',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),

                        // ✅ 並び替え：
                        // 1行目：苗字｜苗字かな
                        Row(
                          children: [
                            Expanded(
                              child: TextField(
                                controller: _vm.lastNameCtrl,
                                textInputAction: TextInputAction.next,
                                decoration: const InputDecoration(
                                  labelText: '苗字',
                                  border: OutlineInputBorder(),
                                ),
                              ),
                            ),
                            const SizedBox(width: 12),
                            Expanded(
                              child: TextField(
                                controller: _vm.lastNameKanaCtrl,
                                textInputAction: TextInputAction.next,
                                decoration: const InputDecoration(
                                  labelText: '苗字かな',
                                  border: OutlineInputBorder(),
                                ),
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 12),

                        // 2行目：名前｜名前かな
                        Row(
                          children: [
                            Expanded(
                              child: TextField(
                                controller: _vm.firstNameCtrl,
                                textInputAction: TextInputAction.next,
                                decoration: const InputDecoration(
                                  labelText: '名前',
                                  border: OutlineInputBorder(),
                                ),
                              ),
                            ),
                            const SizedBox(width: 12),
                            Expanded(
                              child: TextField(
                                controller: _vm.firstNameKanaCtrl,
                                textInputAction: TextInputAction.next,
                                decoration: const InputDecoration(
                                  labelText: '名前かな',
                                  border: OutlineInputBorder(),
                                ),
                              ),
                            ),
                          ],
                        ),

                        const SizedBox(height: 16),

                        Text(
                          '配送先住所',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),

                        TextField(
                          controller: _vm.zipCtrl,
                          keyboardType: TextInputType.number,
                          decoration: InputDecoration(
                            labelText: '郵便番号（7桁）',
                            border: const OutlineInputBorder(),
                            helperText: '例: 1000001（ハイフン不要）',
                            suffixIcon: _vm.zipLoading
                                ? const Padding(
                                    padding: EdgeInsets.all(12),
                                    child: SizedBox(
                                      width: 16,
                                      height: 16,
                                      child: CircularProgressIndicator(
                                        strokeWidth: 2,
                                      ),
                                    ),
                                  )
                                : IconButton(
                                    tooltip: '住所を自動入力',
                                    onPressed: () => _vm.onZipSearchPressed(),
                                    icon: const Icon(Icons.search),
                                  ),
                            errorText:
                                (_vm.zipError == null ||
                                    _vm.zipError!.trim().isEmpty)
                                ? null
                                : _vm.zipError,
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _vm.prefCtrl,
                          decoration: const InputDecoration(
                            labelText: '都道府県',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _vm.cityCtrl,
                          decoration: const InputDecoration(
                            labelText: '市区町村',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _vm.addr1Ctrl,
                          decoration: const InputDecoration(
                            labelText: '住所１（町名・番地など）',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _vm.addr2Ctrl,
                          decoration: const InputDecoration(
                            labelText: '住所２（建物名・部屋番号など）',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 16),

                        ElevatedButton(
                          onPressed: _vm.canSaveAddress
                              ? () => _vm.saveAddressToBackend(context)
                              : null,
                          child: _vm.saving
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Text('この住所を保存する'),
                        ),

                        if (_vm.saveMsg != null) ...[
                          const SizedBox(height: 12),
                          _InfoBox(
                            kind: _vm.saveMsg!.contains('保存しました')
                                ? _InfoKind.ok
                                : _InfoKind.info,
                            text: _vm.saveMsg!,
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
