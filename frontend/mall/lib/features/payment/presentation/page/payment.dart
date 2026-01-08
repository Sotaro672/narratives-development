// frontend\mall\lib\features\payment\presentation\page\payment.dart
import 'package:flutter/material.dart';

import '../hook/use_payment.dart';

class PaymentPage extends StatefulWidget {
  const PaymentPage({super.key, this.avatarId = '', this.from});

  final String avatarId;
  final String? from;

  @override
  State<PaymentPage> createState() => _PaymentPageState();
}

class _PaymentPageState extends State<PaymentPage> {
  late final UsePaymentController _uc;
  late Future<PaymentPageVM> _future;

  @override
  void initState() {
    super.initState();
    _uc = UsePaymentController();
    _future = _uc.load(qpAvatarId: widget.avatarId);
  }

  @override
  void dispose() {
    _uc.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    // ✅ AppShell/AppMain の中で使う前提なので Scaffold は作らない
    return SafeArea(
      child: FutureBuilder<PaymentPageVM>(
        future: _future,
        builder: (context, snap) {
          if (snap.connectionState == ConnectionState.waiting &&
              !snap.hasData) {
            return const Center(child: CircularProgressIndicator());
          }

          if (snap.hasError) {
            return _ErrorBox(
              title: 'Payment load failed',
              message: snap.error.toString(),
            );
          }

          final vm = snap.data;
          if (vm == null) {
            return const _ErrorBox(title: 'No data', message: 'vm is null');
          }

          final cards = <Widget>[
            _BillingCard(vm: vm.billing),
            const SizedBox(height: 12),
            _ShippingCard(vm: vm.shipping),
            const SizedBox(height: 12),
            _CartCard(vm: vm.cart),
          ];

          return Padding(
            padding: const EdgeInsets.fromLTRB(12, 12, 12, 24),
            child: LayoutBuilder(
              builder: (context, constraints) {
                if (constraints.hasBoundedHeight) {
                  return ListView(children: cards);
                }
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: cards,
                );
              },
            ),
          );
        },
      ),
    );
  }
}

// ------------------------------------------------------------
// Cards (style-only)
// ------------------------------------------------------------

class _ShippingCard extends StatelessWidget {
  const _ShippingCard({required this.vm});
  final ShippingCardVM vm;

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('配送先住所', style: t.titleMedium),
            const SizedBox(height: 8),
            if (vm.isEmpty)
              Text('(empty)', style: t.bodyMedium)
            else
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  for (final line in vm.lines)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 4),
                      child: Text(line, style: t.bodyMedium),
                    ),
                ],
              ),
          ],
        ),
      ),
    );
  }
}

class _BillingCard extends StatelessWidget {
  const _BillingCard({required this.vm});
  final BillingCardVM vm;

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('お支払い選択', style: t.titleMedium),
            const SizedBox(height: 8),
            if (vm.isEmpty)
              Text('(empty)', style: t.bodyMedium)
            else ...[
              if (vm.holderLine.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(bottom: 4),
                  child: Text(vm.holderLine, style: t.bodyMedium),
                ),
              Padding(
                padding: const EdgeInsets.only(bottom: 4),
                child: Text(vm.cardNumberLine, style: t.bodyMedium),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _CartCard extends StatelessWidget {
  const _CartCard({required this.vm});
  final CartCardVM vm;

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;

    return Card(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('購入商品', style: t.titleMedium),
            const SizedBox(height: 8),
            if (vm.isEmpty)
              Text('カートは空です', style: t.bodyMedium)
            else ...[
              ListView.separated(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                itemCount: vm.items.length,
                separatorBuilder: (_, __) => Divider(
                  height: 1,
                  thickness: 1,
                  color: Theme.of(context).dividerColor.withValues(alpha: 0.35),
                ),
                itemBuilder: (context, i) {
                  final it = vm.items[i];

                  final hasImgUrl =
                      it.imageUrl != null && it.imageUrl!.isNotEmpty;

                  return ListTile(
                    contentPadding: const EdgeInsets.symmetric(horizontal: 4),
                    leading: SizedBox(
                      width: 44,
                      height: 44,
                      child: ClipRRect(
                        borderRadius: BorderRadius.circular(6),
                        child: hasImgUrl
                            ? Image.network(
                                it.imageUrl!,
                                fit: BoxFit.cover,
                                errorBuilder: (_, __, ___) => const Icon(
                                  Icons.image_not_supported_outlined,
                                ),
                              )
                            : const Icon(Icons.inventory_2_outlined),
                      ),
                    ),
                    title: Text(
                      it.title,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: t.bodyMedium,
                    ),
                    subtitle: Text(
                      it.subtitleLines.join('\n'),
                      maxLines: 4,
                      overflow: TextOverflow.ellipsis,
                      style: t.bodySmall,
                    ),
                    trailing: Text(it.trailingPrice, style: t.bodyMedium),
                  );
                },
              ),

              // ✅ 合計価格行
              const SizedBox(height: 10),
              Divider(
                height: 1,
                thickness: 1,
                color: Theme.of(context).dividerColor.withValues(alpha: 0.35),
              ),
              const SizedBox(height: 10),
              Row(
                children: [
                  Text('合計価格', style: t.titleSmall),
                  const Spacer(),
                  Text(vm.totalLine, style: t.titleSmall),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }
}

// ------------------------------------------------------------
// Error (style-only)
// ------------------------------------------------------------

class _ErrorBox extends StatelessWidget {
  const _ErrorBox({required this.title, required this.message});

  final String title;
  final String message;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(title),
              const SizedBox(height: 8),
              Text(message, style: const TextStyle(fontFamily: 'monospace')),
            ],
          ),
        ),
      ),
    );
  }
}
