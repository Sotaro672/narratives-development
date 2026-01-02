// frontend/sns/lib/features/payment/presentation/page/payment.dart
import 'package:flutter/material.dart';

import '../../infrastructure/payment_repository_http.dart';

// ✅ Cart を読む
import '../../../cart/infrastructure/cart_repository_http.dart';

class PaymentPage extends StatefulWidget {
  const PaymentPage({super.key, this.avatarId = '', this.from});

  final String avatarId;
  final String? from;

  @override
  State<PaymentPage> createState() => _PaymentPageState();
}

class _PaymentPageState extends State<PaymentPage> {
  late final PaymentRepositoryHttp _paymentRepo;
  late final CartRepositoryHttp _cartRepo;

  late Future<_PaymentVM> _future;

  @override
  void initState() {
    super.initState();
    _paymentRepo = PaymentRepositoryHttp();
    _cartRepo = CartRepositoryHttp();
    _future = _load();
  }

  @override
  void dispose() {
    _paymentRepo.dispose();
    _cartRepo.dispose();
    super.dispose();
  }

  Future<_PaymentVM> _load() async {
    final ctx = await _paymentRepo.fetchPaymentContext();

    // ✅ Cart は「URLで渡している avatarId（現状 uid）」で引くのが正
    final qpAvatarId = widget.avatarId.trim();
    final cartKey = qpAvatarId.isNotEmpty
        ? qpAvatarId
        : (ctx.userId.trim().isNotEmpty ? ctx.userId.trim() : ctx.uid.trim());

    CartDTO cart;
    try {
      cart = await _cartRepo.fetchCart(avatarId: cartKey);
    } catch (e) {
      // 404 は空カート扱い
      if (e is CartHttpException && e.statusCode == 404) {
        cart = _emptyCart(cartKey);
      } else {
        rethrow;
      }
    }

    return _PaymentVM(ctx: ctx, cart: cart, cartKey: cartKey);
  }

  static CartDTO _emptyCart(String avatarId) {
    return CartDTO(
      avatarId: avatarId,
      items: const {},
      createdAt: null,
      updatedAt: null,
      expiresAt: null,
    );
  }

  @override
  Widget build(BuildContext context) {
    // ✅ AppShell/AppMain の中で使う前提なので Scaffold は作らない
    // ✅ AppMain が SingleChildScrollView になる可能性があるため、
    //    このページ内で ListView を固定で返さない（unbounded 対策）
    return SafeArea(
      child: FutureBuilder<_PaymentVM>(
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
            _UserCard(ctx: vm.ctx),
            const SizedBox(height: 12),
            _ShippingCard(ctx: vm.ctx),
            const SizedBox(height: 12),
            _BillingCard(ctx: vm.ctx),
            const SizedBox(height: 12),
            _CartCard(cart: vm.cart),
          ];

          return Padding(
            padding: const EdgeInsets.fromLTRB(12, 12, 12, 24),
            child: LayoutBuilder(
              builder: (context, constraints) {
                // ✅ 高さが bounded の時だけ ListView（スクロールはこの中で完結できる）
                if (constraints.hasBoundedHeight) {
                  return ListView(children: cards);
                }

                // ✅ AppMain(SingleChildScrollView) 配下などは unbounded になり得るので Column
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
// ViewModel
// ------------------------------------------------------------

class _PaymentVM {
  _PaymentVM({required this.ctx, required this.cart, required this.cartKey});

  final PaymentContextDTO ctx;
  final CartDTO cart;

  /// どのキーで cart を引いたか（必要ならログ用途）
  final String cartKey;
}

// ------------------------------------------------------------
// Cards
// ------------------------------------------------------------

class _UserCard extends StatelessWidget {
  const _UserCard({required this.ctx});
  final PaymentContextDTO ctx;

  String _pickName() {
    final d = ctx.debug;
    final ship = ctx.shippingAddress;
    final bill = ctx.billingAddress;

    String s(dynamic v) => (v ?? '').toString().trim();

    return s(d?['fullName']).isNotEmpty
        ? s(d?['fullName'])
        : s(ship?['fullName']).isNotEmpty
        ? s(ship?['fullName'])
        : s(ship?['name']).isNotEmpty
        ? s(ship?['name'])
        : s(bill?['cardholderName']).isNotEmpty
        ? s(bill?['cardholderName'])
        : '-';
  }

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;
    final name = _pickName();

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('User', style: t.titleMedium),
            const SizedBox(height: 8),
            Text('name: $name', style: t.bodyMedium),
          ],
        ),
      ),
    );
  }
}

class _ShippingCard extends StatelessWidget {
  const _ShippingCard({required this.ctx});
  final PaymentContextDTO ctx;

  String _s(dynamic v) => (v ?? '').toString().trim();

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;
    final m = ctx.shippingAddress;

    final zip = _s(m?['zipCode']);
    final state = _s(m?['state']);
    final city = _s(m?['city']);
    final street = _s(m?['street']);
    final street2 = _s(m?['street2']);
    final country = _s(m?['country']);

    final line1 = zip.isNotEmpty ? '〒 $zip' : '';
    final line2 = [
      state,
      city,
      street,
      street2,
    ].where((e) => e.trim().isNotEmpty).join(' ');
    final line3 = country;

    final lines = <String>[
      if (line1.trim().isNotEmpty) line1,
      if (line2.trim().isNotEmpty) line2,
      if (line3.trim().isNotEmpty) line3,
    ];

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Shipping Address', style: t.titleMedium),
            const SizedBox(height: 8),
            if (m == null || m.isEmpty)
              Text('(empty)', style: t.bodyMedium)
            else
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  for (final line in lines)
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
  const _BillingCard({required this.ctx});
  final PaymentContextDTO ctx;

  String _s(dynamic v) => (v ?? '').toString().trim();

  String _maskCardNumber(String raw) {
    final s = raw.replaceAll(' ', '').trim();
    if (s.isEmpty) return '-';
    final last4 = s.length >= 4 ? s.substring(s.length - 4) : s;
    return '**** **** **** $last4';
  }

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;
    final m = ctx.billingAddress;

    final holder = _s(m?['cardholderName']);
    final cardNumber = _maskCardNumber(_s(m?['cardNumber']));

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Billing Address', style: t.titleMedium),
            const SizedBox(height: 8),
            if (m == null || m.isEmpty)
              Text('(empty)', style: t.bodyMedium)
            else ...[
              if (holder.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(bottom: 4),
                  child: Text('cardholderName: $holder', style: t.bodyMedium),
                ),
              Padding(
                padding: const EdgeInsets.only(bottom: 4),
                child: Text('cardNumber: $cardNumber', style: t.bodyMedium),
              ),
              // ⚠️ cvc は表示しない
            ],
          ],
        ),
      ),
    );
  }
}

class _CartCard extends StatelessWidget {
  const _CartCard({required this.cart});
  final CartDTO cart;

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;

    final entries = cart.items.entries.toList()
      ..sort((a, b) => a.key.compareTo(b.key));

    return Card(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Cart', style: t.titleMedium),
            const SizedBox(height: 8),
            if (entries.isEmpty)
              Text('カートは空です', style: t.bodyMedium)
            else
              ListView.separated(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                itemCount: entries.length,
                separatorBuilder: (_, __) => Divider(
                  height: 1,
                  thickness: 1,
                  color: Theme.of(context).dividerColor.withValues(alpha: 0.35),
                ),
                itemBuilder: (context, i) {
                  final e = entries[i];
                  final itemKey = e.key;
                  final it = e.value;

                  return ListTile(
                    contentPadding: const EdgeInsets.symmetric(horizontal: 4),
                    title: Text(
                      it.modelId,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: t.bodyMedium,
                    ),
                    subtitle: Text(
                      'itemKey: $itemKey\ninventoryId: ${it.inventoryId}\nlistId: ${it.listId}\nqty: ${it.qty}',
                      maxLines: 4,
                      overflow: TextOverflow.ellipsis,
                      style: t.bodySmall,
                    ),
                  );
                },
              ),
          ],
        ),
      ),
    );
  }
}

// ------------------------------------------------------------
// Error
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
