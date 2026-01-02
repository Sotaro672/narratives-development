// frontend/sns/lib/features/payment/presentation/page/payment.dart
import 'package:flutter/material.dart';

import '../../infrastructure/payment_repository_http.dart';

class PaymentPage extends StatefulWidget {
  const PaymentPage({super.key, this.avatarId = '', this.from});

  /// ✅ いまは「画面引数で渡される場合は表示して比較する」用途に限定。
  ///   実際のデータ解決は backend の /sns/payment（order_query.go）に寄せる。
  final String avatarId;

  /// 画面遷移元（任意）
  final String? from;

  @override
  State<PaymentPage> createState() => _PaymentPageState();
}

class _PaymentPageState extends State<PaymentPage> {
  late final PaymentRepositoryHttp _repo;
  late Future<PaymentContextDTO> _future;

  bool _busy = false;

  // ✅ incoming / resolved を比較表示する（avatarId の受け渡し撤廃に備える）
  String _resolvedAvatarId = '';

  String get _incomingAvatarId => widget.avatarId.trim();

  @override
  void initState() {
    super.initState();
    _repo = PaymentRepositoryHttp();
    _kickLoad();
  }

  @override
  void dispose() {
    _repo.dispose();
    super.dispose();
  }

  void _kickLoad() {
    _future = _load();
  }

  Future<PaymentContextDTO> _load() async {
    // ✅ 画面引数 avatarId は「必須」ではなくする
    //    - backend /sns/payment が uid -> avatarId + addresses を解決するのが前提
    final ctx = await _repo.fetchPaymentContext();

    // 画面表示用に保持（ログ/デバッグ用）
    _resolvedAvatarId = (ctx.avatarId).trim();

    return ctx;
  }

  Future<void> _reload() async {
    setState(() {
      _kickLoad();
    });
  }

  Future<void> _withBusy(Future<void> Function() fn) async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      await fn();
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  // ✅ ここでは「購入確定のAPI起票」はまだ実装しない（次工程）
  Future<void> _confirmPurchase(PaymentContextDTO ctx) async {
    await _withBusy(() async {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('購入確定は次工程で実装します（UIは準備できています）')),
      );
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Payment'),
        leading: IconButton(
          tooltip: 'Back',
          icon: const Icon(Icons.arrow_back),
          onPressed: () => Navigator.of(context).maybePop(),
        ),
        actions: [
          IconButton(
            tooltip: 'Reload',
            onPressed: _reload,
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: Stack(
        children: [
          FutureBuilder<PaymentContextDTO>(
            future: _future,
            builder: (context, snap) {
              final isLoading =
                  snap.connectionState == ConnectionState.waiting &&
                  !snap.hasData;

              if (isLoading) {
                return const Center(child: CircularProgressIndicator());
              }

              if (snap.hasError) {
                return _ErrorView(
                  errorText: snap.error.toString(),
                  onRetry: _reload,
                );
              }

              final ctx = snap.data;
              if (ctx == null) {
                return _ErrorView(errorText: 'No data', onRetry: _reload);
              }

              final canConfirm =
                  ctx.userId.trim().isNotEmpty &&
                  (ctx.shippingAddress?.isNotEmpty ?? false) &&
                  (ctx.billingAddress?.isNotEmpty ?? false);

              return SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(12, 12, 12, 24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    _HeaderCard(
                      incomingAvatarId: _incomingAvatarId,
                      resolvedAvatarId: _resolvedAvatarId.isNotEmpty
                          ? _resolvedAvatarId
                          : ctx.avatarId,
                      ctx: ctx,
                    ),
                    const SizedBox(height: 12),

                    if (_incomingAvatarId.isEmpty)
                      Padding(
                        padding: const EdgeInsets.only(bottom: 12),
                        child: Text(
                          '※ avatarId は画面引数から渡されていません（/sns/payment の解決結果で表示しています）',
                          style: Theme.of(context).textTheme.bodySmall
                              ?.copyWith(
                                color: Theme.of(context)
                                    .textTheme
                                    .bodySmall
                                    ?.color
                                    ?.withValues(alpha: 0.75),
                              ),
                          textAlign: TextAlign.center,
                        ),
                      ),

                    _AddressCard(
                      title: 'Shipping Address',
                      address: ctx.shippingAddress,
                      emptyText: '配送先住所が未登録です',
                    ),
                    const SizedBox(height: 12),

                    _AddressCard(
                      title: 'Billing Address',
                      address: ctx.billingAddress,
                      emptyText: '請求先住所が未登録です',
                    ),
                    const SizedBox(height: 16),

                    SizedBox(
                      height: 48,
                      child: FilledButton(
                        onPressed: canConfirm
                            ? () => _confirmPurchase(ctx)
                            : null,
                        child: const Text('購入を確定する'),
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      canConfirm
                          ? '※ 次工程で order/payment 起票を実装します'
                          : '※ 住所が揃うと購入確定できます（次工程で住所登録導線も整備）',
                      style: Theme.of(context).textTheme.bodySmall,
                      textAlign: TextAlign.center,
                    ),
                  ],
                ),
              );
            },
          ),
          if (_busy)
            Positioned.fill(
              child: IgnorePointer(
                ignoring: true,
                child: Container(
                  color: Colors.black.withValues(alpha: 0.06),
                  child: const Center(child: CircularProgressIndicator()),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

// ============================================================
// UI parts
// ============================================================

class _HeaderCard extends StatelessWidget {
  const _HeaderCard({
    required this.incomingAvatarId,
    required this.resolvedAvatarId,
    required this.ctx,
  });

  final String incomingAvatarId;
  final String resolvedAvatarId;
  final PaymentContextDTO ctx;

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;

    final uid = ctx.uid.trim();
    final avatarId = ctx.avatarId.trim();
    final userId = ctx.userId.trim();

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('確認', style: t.titleMedium),
            const SizedBox(height: 8),

            Text(
              'incoming avatarId: ${incomingAvatarId.isEmpty ? '(none)' : incomingAvatarId}',
              style: t.bodySmall,
            ),
            const SizedBox(height: 4),
            Text('resolved avatarId: $resolvedAvatarId', style: t.bodySmall),
            const SizedBox(height: 4),
            Text(
              'ctx.avatarId: ${avatarId.isEmpty ? '(empty)' : avatarId}',
              style: t.bodySmall,
            ),
            const SizedBox(height: 4),
            Text('uid: ${uid.isEmpty ? '(empty)' : uid}', style: t.bodySmall),
            const SizedBox(height: 4),
            Text(
              'userId: ${userId.isEmpty ? '(not resolved)' : userId}',
              style: t.bodySmall,
            ),

            if (ctx.debug != null && ctx.debug!.isNotEmpty) ...[
              const SizedBox(height: 10),
              Text('debug', style: t.labelMedium),
              const SizedBox(height: 6),
              ...ctx.debug!.entries
                  .take(12)
                  .map(
                    (e) => Padding(
                      padding: const EdgeInsets.only(bottom: 4),
                      child: Text('${e.key}: ${e.value}', style: t.bodySmall),
                    ),
                  ),
            ],
          ],
        ),
      ),
    );
  }
}

class _AddressCard extends StatelessWidget {
  const _AddressCard({
    required this.title,
    required this.address,
    required this.emptyText,
  });

  final String title;
  final Map<String, dynamic>? address;
  final String emptyText;

  String _s(dynamic v) => (v ?? '').toString().trim();

  List<MapEntry<String, String>> _toPairs(Map<String, dynamic> m) {
    final preferredKeys = <String>[
      'fullName',
      'name',
      'phone',
      'email',
      'postalCode',
      'zip',
      'prefecture',
      'state',
      'city',
      'address1',
      'address2',
      'line1',
      'line2',
      'country',
    ];

    final used = <String>{};
    final pairs = <MapEntry<String, String>>[];

    for (final k in preferredKeys) {
      if (!m.containsKey(k)) continue;
      final v = _s(m[k]);
      if (v.isEmpty) continue;
      used.add(k);
      pairs.add(MapEntry(k, v));
    }

    for (final e in m.entries) {
      final k = e.key.toString();
      if (used.contains(k)) continue;
      final v = _s(e.value);
      if (v.isEmpty) continue;
      pairs.add(MapEntry(k, v));
      if (pairs.length >= 18) break;
    }

    return pairs;
  }

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;
    final m = address;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(title, style: t.titleMedium),
            const SizedBox(height: 10),
            if (m == null || m.isEmpty)
              Text(emptyText, style: t.bodyMedium)
            else
              ..._toPairs(m).map(
                (e) => Padding(
                  padding: const EdgeInsets.only(bottom: 6),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      SizedBox(
                        width: 110,
                        child: Text(
                          e.key,
                          style: t.bodySmall?.copyWith(
                            color: Theme.of(context).textTheme.bodySmall?.color
                                ?.withValues(alpha: 0.7),
                          ),
                        ),
                      ),
                      Expanded(child: Text(e.value, style: t.bodySmall)),
                    ],
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.errorText, required this.onRetry});

  final String errorText;
  final Future<void> Function() onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(18),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text('Error'),
            const SizedBox(height: 8),
            Text(errorText, textAlign: TextAlign.center),
            const SizedBox(height: 12),
            OutlinedButton(
              onPressed: () => onRetry(),
              child: const Text('Retry'),
            ),
          ],
        ),
      ),
    );
  }
}
