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
  late final UsePayment _uc;
  late Future<PaymentPageVM> _future;

  @override
  void initState() {
    super.initState();
    _uc = UsePayment();
    _future = _load();
  }

  Future<PaymentPageVM> _load() async {
    // UsePayment.load の戻り型が何であっても吸収できるように “raw” として受ける
    final raw = await _uc.load(qpAvatarId: widget.avatarId);
    return PaymentPageVM.fromAny(raw);
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
// ViewModels (this page is self-contained)
// ------------------------------------------------------------

class PaymentPageVM {
  PaymentPageVM({
    required this.billing,
    required this.shipping,
    required this.cart,
  });

  final BillingCardVM billing;
  final ShippingCardVM shipping;
  final CartCardVM cart;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  static int _i(dynamic v) {
    if (v is int) return v;
    if (v is double) return v.toInt();
    return int.tryParse(_s(v)) ?? 0;
  }

  static Map<String, dynamic>? _mapAny(dynamic v) {
    if (v == null) return null;
    if (v is Map<String, dynamic>) return v;
    if (v is Map) return v.cast<String, dynamic>();
    return null;
  }

  static String _pick(Map<String, dynamic> m, List<String> keys) {
    for (final k in keys) {
      final s = _s(m[k]);
      if (s.isNotEmpty) return s;
    }
    return '';
  }

  static List<String> _addressLines(Map<String, dynamic>? m) {
    if (m == null || m.isEmpty) return const [];

    // よくある住所キーの名揺れを吸収
    final postal = _pick(m, ['postalCode', 'zip', 'zipCode']);
    final pref = _pick(m, ['prefecture', 'state', 'region']);
    final city = _pick(m, ['city', 'municipality']);
    final line1 = _pick(m, ['address1', 'line1', 'street1', 'address']);
    final line2 = _pick(m, ['address2', 'line2', 'street2']);

    final name = _pick(m, [
      'name',
      'fullName',
      'recipientName',
      'holderName',
      'billingName',
      'shippingName',
    ]);
    final phone = _pick(m, ['phone', 'tel', 'phoneNumber']);

    final lines = <String>[];
    if (name.isNotEmpty) lines.add(name);

    final addrTop = [
      if (postal.isNotEmpty) '〒$postal',
      if (pref.isNotEmpty) pref,
      if (city.isNotEmpty) city,
    ].join(' ');
    if (addrTop.trim().isNotEmpty) lines.add(addrTop.trim());

    final addrMid = [line1, line2].where((e) => e.trim().isNotEmpty).join(' ');
    if (addrMid.trim().isNotEmpty) lines.add(addrMid.trim());

    if (phone.isNotEmpty) lines.add('TEL: $phone');

    // 最低限 “空ではない” 判定を成立させる
    if (lines.isEmpty) {
      // 何かしら入ってるはずなのでダンプ用に代表値を置く
      final id = _pick(m, [
        'id',
        'billingAddressId',
        'shippingAddressId',
        'addressId',
      ]);
      if (id.isNotEmpty) lines.add('id: $id');
    }
    return lines;
  }

  static PaymentPageVM fromAny(dynamic raw) {
    // raw.ctx.shippingAddress / raw.ctx.billingAddress を想定（無ければ直下を探す）
    Map<String, dynamic>? shippingMap;
    Map<String, dynamic>? billingMap;
    dynamic rawCart;

    try {
      // ignore: avoid_dynamic_calls
      final ctx = (raw as dynamic).ctx;
      // ignore: avoid_dynamic_calls
      shippingMap = _mapAny(ctx.shippingAddress);
      // ignore: avoid_dynamic_calls
      billingMap = _mapAny(ctx.billingAddress);
      // ignore: avoid_dynamic_calls
      rawCart = (raw as dynamic).rawCart;
    } catch (_) {
      // fallback: raw 自体が map 形式の可能性もある
      final rm = _mapAny(raw) ?? <String, dynamic>{};
      shippingMap = _mapAny(rm['shippingAddress']);
      billingMap = _mapAny(rm['billingAddress']);
      rawCart = rm['cart'] ?? rm['rawCart'];
    }

    final shippingLines = _addressLines(shippingMap);
    final shippingVM = ShippingCardVM(lines: shippingLines);

    // ✅ use_payment.dart が保持している billingAddress を優先表示（カード名義/カード番号）
    final billingVM = BillingCardVM.fromBillingAddressMap(billingMap);

    // cart
    final items = <CartItemVM>[];
    var total = 0;

    dynamic itemsAny;
    try {
      // ignore: avoid_dynamic_calls
      itemsAny = (rawCart as dynamic).items;
    } catch (_) {
      final m = _mapAny(rawCart);
      itemsAny = m?['items'];
    }

    if (itemsAny is Map) {
      for (final e in itemsAny.entries) {
        final it = e.value;

        String title = '';
        String imageUrl = ''; // ✅ nullable で持たない（! や null 比較を不要にする）
        int qty = 0;
        int unit = 0;

        String productName = '';
        String size = '';
        String color = '';

        try {
          // ignore: avoid_dynamic_calls
          title = _s(it.title);
          // ignore: avoid_dynamic_calls
          imageUrl = _s(it.listImage);
          // ignore: avoid_dynamic_calls
          unit = _i(it.price);
          // ignore: avoid_dynamic_calls
          qty = _i(it.qty);

          // ignore: avoid_dynamic_calls
          productName = _s(it.productName);
          // ignore: avoid_dynamic_calls
          size = _s(it.size);
          // ignore: avoid_dynamic_calls
          color = _s(it.color);
        } catch (_) {
          final im = _mapAny(it) ?? <String, dynamic>{};
          title = _s(im['title']);
          imageUrl = _s(im['listImage']);
          if (imageUrl.isEmpty) imageUrl = _s(im['imageUrl']);
          unit = _i(im['price']);
          qty = _i(im['qty']);

          productName = _s(im['productName']);
          size = _s(im['size']);
          color = _s(im['color']);
        }

        if (title.isEmpty) title = '(no title)';
        if (qty <= 0) qty = 1;

        final lineTotal = unit * qty;
        if (lineTotal > 0) total += lineTotal;

        final subtitle = <String>[];
        if (productName.isNotEmpty) subtitle.add(productName);

        final sc = <String>[
          if (size.isNotEmpty) 'size: $size',
          if (color.isNotEmpty) 'color: $color',
          'qty: $qty',
        ].join(' / ');
        if (sc.trim().isNotEmpty) subtitle.add(sc);

        final cleanedImg = imageUrl.trim();

        items.add(
          CartItemVM(
            title: title,
            subtitleLines: subtitle,
            imageUrl: cleanedImg.isNotEmpty ? cleanedImg : null,
            trailingPrice: _yen(lineTotal),
          ),
        );
      }
    }

    final cartVM = CartCardVM(items: items, total: total);

    return PaymentPageVM(
      billing: billingVM,
      shipping: shippingVM,
      cart: cartVM,
    );
  }

  static String _yen(int n) {
    if (n <= 0) return '¥0';
    final s = n.toString();
    final buf = StringBuffer();
    for (int i = 0; i < s.length; i++) {
      final idxFromEnd = s.length - i;
      buf.write(s[i]);
      if (idxFromEnd > 1 && idxFromEnd % 3 == 1) {
        buf.write(',');
      }
    }
    return '¥${buf.toString()}';
  }
}

class ShippingCardVM {
  const ShippingCardVM({required this.lines});
  final List<String> lines;
  bool get isEmpty => lines.isEmpty;
}

class BillingCardVM {
  const BillingCardVM({
    required this.holderLine,
    required this.cardNumberLine,
    required this.isEmpty,
  });

  final String holderLine;
  final String cardNumberLine;
  final bool isEmpty;

  // ✅ use_payment.dart が保持している billingAddress(Map) からカード表示用VMを生成
  factory BillingCardVM.fromBillingAddressMap(Map<String, dynamic>? m) {
    if (m == null || m.isEmpty) {
      return const BillingCardVM(
        holderLine: '',
        cardNumberLine: '(empty)',
        isEmpty: true,
      );
    }

    String s(dynamic v) => (v ?? '').toString().trim();

    // use_payment.dart と一致するキー
    final holder = s(m['cardholderName']);
    final rawNumber = s(m['cardNumber']);

    // 将来/揺れ対策: サーバが last4 を直接返す場合にも対応
    final last4 = s(m['last4']);

    String maskCardNumber(String raw, {String last4Fallback = ''}) {
      final cleaned = raw.replaceAll(' ', '').trim();
      if (cleaned.isNotEmpty) {
        final last = cleaned.length >= 4
            ? cleaned.substring(cleaned.length - 4)
            : cleaned;
        return '**** **** **** $last';
      }
      if (last4Fallback.isNotEmpty) {
        return '**** **** **** $last4Fallback';
      }
      return '';
    }

    final masked = maskCardNumber(rawNumber, last4Fallback: last4);

    // カード情報があればカード表示を採用
    final hasCardInfo = holder.isNotEmpty || masked.isNotEmpty;
    if (hasCardInfo) {
      return BillingCardVM(
        isEmpty: false,
        holderLine: holder.isNotEmpty ? 'カード名義: $holder' : '',
        cardNumberLine: masked.isNotEmpty ? 'カード番号: $masked' : 'カード番号: (未設定)',
      );
    }

    // カード情報が無い場合は従来どおり住所lines表示へフォールバック
    final lines = PaymentPageVM._addressLines(m);
    if (lines.isEmpty) {
      return const BillingCardVM(
        holderLine: '',
        cardNumberLine: '(empty)',
        isEmpty: true,
      );
    }

    final holderLine = lines.first;
    final rest = lines.skip(1).join(' / ').trim();
    final line2 = rest.isEmpty ? '決済: (未設定 / 開発中)' : rest;

    return BillingCardVM(
      holderLine: holderLine,
      cardNumberLine: line2,
      isEmpty: false,
    );
  }

  // 互換: 既存の lines から生成するAPIも残す（他で参照されている可能性に備える）
  factory BillingCardVM.fromLines(List<String> lines) {
    if (lines.isEmpty) {
      return const BillingCardVM(
        holderLine: '',
        cardNumberLine: '(empty)',
        isEmpty: true,
      );
    }
    // “カード情報”はまだ無い想定なので、住所の先頭を holderLine に流用
    final holder = lines.first;
    final rest = lines.skip(1).join(' / ').trim();
    final line2 = rest.isEmpty ? '決済: (未設定 / 開発中)' : rest;
    return BillingCardVM(
      holderLine: holder,
      cardNumberLine: line2,
      isEmpty: false,
    );
  }
}

class CartCardVM {
  const CartCardVM({required this.items, required this.total});
  final List<CartItemVM> items;
  final int total;

  bool get isEmpty => items.isEmpty;

  String get totalLine {
    // PaymentPageVM の表示ロジックに合わせて ¥ + 3桁カンマ
    final s = total.toString();
    final buf = StringBuffer();
    for (int i = 0; i < s.length; i++) {
      final idxFromEnd = s.length - i;
      buf.write(s[i]);
      if (idxFromEnd > 1 && idxFromEnd % 3 == 1) {
        buf.write(',');
      }
    }
    return '¥${buf.toString()}';
  }
}

class CartItemVM {
  const CartItemVM({
    required this.title,
    required this.subtitleLines,
    required this.imageUrl,
    required this.trailingPrice,
  });

  final String title;
  final List<String> subtitleLines;
  final String? imageUrl;
  final String trailingPrice;
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
                  final img = it.imageUrl;
                  final hasImgUrl = (img ?? '').isNotEmpty;

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
