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
            _BillingCard(ctx: vm.ctx),
            const SizedBox(height: 12),
            _ShippingCard(ctx: vm.ctx),
            const SizedBox(height: 12),
            // ✅ CartItemDTO の中身を表示に使う
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

    // ✅ 「JP」は表示しない（国コードが JP の場合は出さない）
    final line3 = (country.toUpperCase() == 'JP') ? '' : country;

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
            Text('配送先住所', style: t.titleMedium),
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

    // ✅ fullName 等は拾わない。cardholderName のみ。
    final holder = _s(m?['cardholderName']);
    final cardNumber = _maskCardNumber(_s(m?['cardNumber']));

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('お支払い選択', style: t.titleMedium),
            const SizedBox(height: 8),
            if (m == null || m.isEmpty)
              Text('(empty)', style: t.bodyMedium)
            else ...[
              if (holder.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(bottom: 4),
                  child: Text('カード名義: $holder', style: t.bodyMedium),
                ),
              Padding(
                padding: const EdgeInsets.only(bottom: 4),
                child: Text('カード番号: $cardNumber', style: t.bodyMedium),
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

  String _yen(int? v) {
    if (v == null) return '-';
    return '¥$v';
  }

  String _yen0(int v) => '¥$v';

  bool _looksLikeUrl(String? s) {
    final t = (s ?? '').trim();
    if (t.isEmpty) return false;
    return t.startsWith('http://') || t.startsWith('https://');
  }

  String _pickTitle(CartItemDTO it) {
    final a = (it.title ?? '').trim();
    if (a.isNotEmpty) return a;

    final b = (it.productName ?? '').trim();
    if (b.isNotEmpty) return b;

    return it.modelId.trim().isNotEmpty ? it.modelId : '-';
  }

  String _pickVariantLine(CartItemDTO it) {
    final size = (it.size ?? '').trim();
    final color = (it.color ?? '').trim();
    final parts = <String>[
      if (size.isNotEmpty) 'サイズ: $size',
      if (color.isNotEmpty) 'カラー: $color',
    ];
    return parts.join(' / ');
  }

  int _calcTotal(CartDTO cart) {
    var sum = 0;
    for (final it in cart.items.values) {
      final p = it.price ?? 0;
      final q = it.qty;
      if (p > 0 && q > 0) sum += p * q;
    }
    return sum;
  }

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;

    // ✅ itemKey -> CartItemDTO を並べ替えて表示
    final entries = cart.items.entries.toList()
      ..sort((a, b) => a.key.compareTo(b.key));

    final total = _calcTotal(cart);

    return Card(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('購入商品', style: t.titleMedium),
            const SizedBox(height: 8),
            if (entries.isEmpty)
              Text('カートは空です', style: t.bodyMedium)
            else ...[
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
                  final it = e.value;

                  final title = _pickTitle(it);
                  final variantLine = _pickVariantLine(it);

                  final price = _yen(it.price);
                  final qty = it.qty;

                  final img = it.listImage;
                  final hasImgUrl = _looksLikeUrl(img);

                  return ListTile(
                    contentPadding: const EdgeInsets.symmetric(horizontal: 4),
                    leading: SizedBox(
                      width: 44,
                      height: 44,
                      child: ClipRRect(
                        borderRadius: BorderRadius.circular(6),
                        child: hasImgUrl
                            ? Image.network(
                                img!,
                                fit: BoxFit.cover,
                                errorBuilder: (_, __, ___) => const Icon(
                                  Icons.image_not_supported_outlined,
                                ),
                              )
                            : const Icon(Icons.inventory_2_outlined),
                      ),
                    ),
                    title: Text(
                      title,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: t.bodyMedium,
                    ),
                    subtitle: Text(
                      [
                        if (variantLine.isNotEmpty) variantLine,
                        '数量: $qty',
                        '価格: $price',
                      ].join('\n'),
                      maxLines: 4,
                      overflow: TextOverflow.ellipsis,
                      style: t.bodySmall,
                    ),
                    trailing: Text(
                      price == '-' ? '' : price,
                      style: t.bodyMedium,
                    ),
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
                  Text(_yen0(total), style: t.titleSmall),
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
