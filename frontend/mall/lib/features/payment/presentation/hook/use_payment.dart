// frontend\mall\lib\features\payment\presentation\hook\use_payment.dart
import '../../infrastructure/payment_repository_http.dart';
import '../../../cart/infrastructure/cart_repository_http.dart';
import '../../../order/infrastructure/order_repository_http.dart';

/// PaymentPage の “ロジック側” を集約（データ取得・整形・計算）
class UsePaymentController {
  UsePaymentController({
    PaymentRepositoryHttp? paymentRepo,
    CartRepositoryHttp? cartRepo,
    OrderRepositoryHttp? orderRepo,
  }) : _paymentRepo = paymentRepo ?? PaymentRepositoryHttp(),
       _cartRepo = cartRepo ?? CartRepositoryHttp(),
       _orderRepo = orderRepo ?? OrderRepositoryHttp();

  final PaymentRepositoryHttp _paymentRepo;
  final CartRepositoryHttp _cartRepo;
  final OrderRepositoryHttp _orderRepo;

  void dispose() {
    _paymentRepo.dispose();
    _cartRepo.dispose();
    _orderRepo.dispose();
  }

  Future<PaymentPageVM> load({required String qpAvatarId}) async {
    final ctx = await _paymentRepo.fetchPaymentContext();

    final qpId = qpAvatarId.trim();

    // cart fetch key: まず avatarId を優先、無ければ従来互換で uid/userId にフォールバック
    final cartKey = qpId.isNotEmpty
        ? qpId
        : (ctx.userId.trim().isNotEmpty ? ctx.userId.trim() : ctx.uid.trim());

    CartDTO rawCart;
    try {
      rawCart = await _cartRepo.fetchCart(avatarId: cartKey);
    } catch (e) {
      if (e is CartHttpException && e.statusCode == 404) {
        rawCart = _emptyCart(cartKey);
      } else {
        rethrow;
      }
    }

    // ✅ Order 起票に渡す avatarId（原則は URL の qpAvatarId）
    final avatarId = qpId.isNotEmpty
        ? qpId
        : (rawCart.avatarId.trim().isNotEmpty
              ? rawCart.avatarId.trim()
              : cartKey);

    final shipping = _buildShippingVM(ctx.shippingAddress);
    final billing = _buildBillingVM(ctx.billingAddress);
    final cartVm = _buildCartVM(rawCart);

    return PaymentPageVM(
      ctx: ctx,
      rawCart: rawCart,
      cartKey: cartKey,
      avatarId: avatarId,
      shipping: shipping,
      billing: billing,
      cart: cartVm,
    );
  }

  /// ✅ 支払確定 = Order起票（/mall/orders）
  /// Items は snapshot: [modelId, inventoryId, qty, price]
  ///
  /// ✅ 方針変更:
  /// - Order はまず単独で起票する
  /// - invoiceId / paymentId をフロント側必須にしない（サーバ側で不要な形に寄せる）
  Future<Map<String, dynamic>> confirmAndCreateOrder(PaymentPageVM vm) async {
    if (vm.cart.isEmpty) {
      throw Exception('cart is empty');
    }
    if (vm.shipping.isEmpty) {
      throw Exception('shipping address is empty');
    }
    if (vm.billing.isEmpty) {
      throw Exception('billing is empty');
    }

    // ✅ avatarId は必須（バックエンドで Order.avatarId を保存するため）
    final avatarId = vm.avatarId.trim();
    if (avatarId.isEmpty) {
      throw Exception('avatarId is empty');
    }

    final userId = vm.ctx.userId.trim().isNotEmpty
        ? vm.ctx.userId.trim()
        : vm.ctx.uid.trim();
    if (userId.isEmpty) {
      throw Exception('userId/uid is empty');
    }

    final cartId = vm.cartKey.trim().isNotEmpty
        ? vm.cartKey.trim()
        : vm.rawCart.avatarId.trim();
    if (cartId.isEmpty) {
      throw Exception('cartId is empty');
    }

    // ✅ items snapshot を作る（key は inventoryId として運用している前提）
    final items = <Map<String, dynamic>>[];
    for (final e in vm.rawCart.items.entries) {
      final invId = e.key.trim();
      final it = e.value;

      final modelId = it.modelId.trim();
      final qty = it.qty;
      final price = it.price;

      if (invId.isEmpty) {
        throw Exception('inventoryId is empty in cart item key');
      }
      if (modelId.isEmpty) {
        throw Exception('modelId is empty (inventoryId=$invId)');
      }
      if (qty <= 0) {
        throw Exception('qty is invalid (inventoryId=$invId, qty=$qty)');
      }
      if (price == null) {
        throw Exception('price is missing (inventoryId=$invId)');
      }

      items.add({
        'modelId': modelId,
        'inventoryId': invId,
        'qty': qty,
        'price': price,
      });
    }
    if (items.isEmpty) {
      throw Exception('items is empty');
    }

    final ship = _buildShippingSnapshot(vm.ctx.shippingAddress);
    final bill = _buildBillingSnapshot(vm.ctx.billingAddress);

    // ✅ invoiceId/paymentId は “現段階では使わない”
    // （Order単独起票 → OrderId で Invoice/Payment を起票するフローに変更）
    return _orderRepo.createOrder(
      userId: userId,
      avatarId: avatarId, // ✅ 追加
      cartId: cartId,
      shippingSnapshot: ship,
      billingSnapshot: bill,
      items: items,
    );
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

  // ------------------------------------------------------------
  // Snapshots (for order create)
  // ------------------------------------------------------------

  Map<String, dynamic> _buildShippingSnapshot(Map<String, dynamic>? m) {
    if (m == null || m.isEmpty) {
      throw Exception('shippingAddress is missing');
    }
    return <String, dynamic>{
      'zipCode': _s(m['zipCode']),
      'state': _s(m['state']),
      'city': _s(m['city']),
      'street': _s(m['street']),
      'street2': _s(m['street2']),
      'country': _s(m['country']),
    };
  }

  Map<String, dynamic> _buildBillingSnapshot(Map<String, dynamic>? m) {
    if (m == null || m.isEmpty) {
      throw Exception('billingAddress is missing');
    }

    final holder = _s(m['cardholderName']);
    final cardNumber = _s(m['cardNumber']).replaceAll(' ', '');
    final last4 = cardNumber.isNotEmpty
        ? (cardNumber.length >= 4
              ? cardNumber.substring(cardNumber.length - 4)
              : cardNumber)
        : '';

    if (last4.isEmpty) {
      throw Exception('billing last4 is missing');
    }

    // cardHolderName は domain 的には任意だが、UI 的にはある方が良いのでできれば入れる
    return <String, dynamic>{'last4': last4, 'cardHolderName': holder};
  }

  // ------------------------------------------------------------
  // VM builders
  // ------------------------------------------------------------

  ShippingCardVM _buildShippingVM(Map<String, dynamic>? m) {
    if (m == null || m.isEmpty) {
      return const ShippingCardVM(isEmpty: true, lines: []);
    }

    final zip = _s(m['zipCode']);
    final state = _s(m['state']);
    final city = _s(m['city']);
    final street = _s(m['street']);
    final street2 = _s(m['street2']);
    final country = _s(m['country']);

    final line1 = zip.isNotEmpty ? '〒 $zip' : '';
    final line2 = [
      state,
      city,
      street,
      street2,
    ].where((e) => e.trim().isNotEmpty).join(' ');

    final line3 = (country.toUpperCase() == 'JP') ? '' : country;

    final lines = <String>[
      if (line1.trim().isNotEmpty) line1,
      if (line2.trim().isNotEmpty) line2,
      if (line3.trim().isNotEmpty) line3,
    ];

    return ShippingCardVM(isEmpty: lines.isEmpty, lines: lines);
  }

  BillingCardVM _buildBillingVM(Map<String, dynamic>? m) {
    if (m == null || m.isEmpty) {
      return const BillingCardVM(
        isEmpty: true,
        holderLine: '',
        cardNumberLine: '',
      );
    }

    final holder = _s(m['cardholderName']);
    final cardNumberMasked = _maskCardNumber(_s(m['cardNumber']));

    return BillingCardVM(
      isEmpty: false,
      holderLine: holder.isNotEmpty ? 'カード名義: $holder' : '',
      cardNumberLine: 'カード番号: $cardNumberMasked',
    );
  }

  CartCardVM _buildCartVM(CartDTO cart) {
    final entries = cart.items.entries.toList()
      ..sort((a, b) => a.key.compareTo(b.key));

    if (entries.isEmpty) {
      return const CartCardVM(isEmpty: true, items: [], totalLine: '¥0');
    }

    final items = <CartLineVM>[];
    var sum = 0;

    for (final e in entries) {
      final it = e.value;

      final title = _pickTitle(it);
      final variant = _pickVariantLine(it);

      final p = it.price;
      final q = it.qty;
      if ((p ?? 0) > 0 && q > 0) sum += (p ?? 0) * q;

      final priceText = _yen(p);
      final trailing = (p == null) ? '' : priceText;

      final subtitle = <String>[
        if (variant.isNotEmpty) variant,
        '数量: $q',
        '価格: $priceText',
      ];

      items.add(
        CartLineVM(
          title: title,
          subtitleLines: subtitle,
          trailingPrice: trailing,
          imageUrl: _looksLikeUrl(it.listImage) ? it.listImage!.trim() : null,
        ),
      );
    }

    return CartCardVM(isEmpty: false, items: items, totalLine: _yen0(sum));
  }

  // ------------------------------------------------------------
  // formatting helpers
  // ------------------------------------------------------------

  static String _s(dynamic v) => (v ?? '').toString().trim();

  static String _yen(int? v) {
    if (v == null) return '-';
    return '¥$v';
  }

  static String _yen0(int v) => '¥$v';

  static bool _looksLikeUrl(String? s) {
    final t = (s ?? '').trim();
    if (t.isEmpty) return false;
    return t.startsWith('http://') || t.startsWith('https://');
  }

  static String _pickTitle(CartItemDTO it) {
    final a = (it.title ?? '').trim();
    if (a.isNotEmpty) return a;

    final b = (it.productName ?? '').trim();
    if (b.isNotEmpty) return b;

    return it.modelId.trim().isNotEmpty ? it.modelId : '-';
  }

  static String _pickVariantLine(CartItemDTO it) {
    final size = (it.size ?? '').trim();
    final color = (it.color ?? '').trim();
    final parts = <String>[
      if (size.isNotEmpty) 'サイズ: $size',
      if (color.isNotEmpty) 'カラー: $color',
    ];
    return parts.join(' / ');
  }

  static String _maskCardNumber(String raw) {
    final s = raw.replaceAll(' ', '').trim();
    if (s.isEmpty) return '-';
    final last4 = s.length >= 4 ? s.substring(s.length - 4) : s;
    return '**** **** **** $last4';
  }
}

// ============================================================
// View Models
// ============================================================

class PaymentPageVM {
  PaymentPageVM({
    required this.ctx,
    required this.rawCart,
    required this.cartKey,
    required this.avatarId,
    required this.shipping,
    required this.billing,
    required this.cart,
  });

  final PaymentContextDTO ctx;
  final CartDTO rawCart;
  final String cartKey;

  // ✅ 追加: backend へ渡す avatarId
  final String avatarId;

  final ShippingCardVM shipping;
  final BillingCardVM billing;
  final CartCardVM cart;
}

class ShippingCardVM {
  const ShippingCardVM({required this.isEmpty, required this.lines});
  final bool isEmpty;
  final List<String> lines;
}

class BillingCardVM {
  const BillingCardVM({
    required this.isEmpty,
    required this.holderLine,
    required this.cardNumberLine,
  });

  final bool isEmpty;
  final String holderLine;
  final String cardNumberLine;
}

class CartCardVM {
  const CartCardVM({
    required this.isEmpty,
    required this.items,
    required this.totalLine,
  });

  final bool isEmpty;
  final List<CartLineVM> items;
  final String totalLine;
}

class CartLineVM {
  const CartLineVM({
    required this.title,
    required this.subtitleLines,
    required this.trailingPrice,
    required this.imageUrl,
  });

  final String title;
  final List<String> subtitleLines;
  final String trailingPrice;
  final String? imageUrl;
}
