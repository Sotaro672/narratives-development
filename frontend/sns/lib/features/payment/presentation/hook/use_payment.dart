// frontend/sns/lib/features/payment/presentation/hook/use_payment.dart
import '../../infrastructure/payment_repository_http.dart';
import '../../../cart/infrastructure/cart_repository_http.dart';

/// PaymentPage の “ロジック側” を集約（データ取得・整形・計算）
/// - payment.dart には UI/スタイル要素だけを残す
class UsePaymentController {
  UsePaymentController({
    PaymentRepositoryHttp? paymentRepo,
    CartRepositoryHttp? cartRepo,
  }) : _paymentRepo = paymentRepo ?? PaymentRepositoryHttp(),
       _cartRepo = cartRepo ?? CartRepositoryHttp();

  final PaymentRepositoryHttp _paymentRepo;
  final CartRepositoryHttp _cartRepo;

  void dispose() {
    _paymentRepo.dispose();
    _cartRepo.dispose();
  }

  Future<PaymentPageVM> load({required String qpAvatarId}) async {
    final ctx = await _paymentRepo.fetchPaymentContext();

    // ✅ Cart は「URLで渡している avatarId（現状 uid）」で引くのが正
    final qpId = qpAvatarId.trim();
    final cartKey = qpId.isNotEmpty
        ? qpId
        : (ctx.userId.trim().isNotEmpty ? ctx.userId.trim() : ctx.uid.trim());

    CartDTO rawCart;
    try {
      rawCart = await _cartRepo.fetchCart(avatarId: cartKey);
    } catch (e) {
      // 404 は空カート扱い
      if (e is CartHttpException && e.statusCode == 404) {
        rawCart = _emptyCart(cartKey);
      } else {
        rethrow;
      }
    }

    final shipping = _buildShippingVM(ctx.shippingAddress);
    final billing = _buildBillingVM(ctx.billingAddress);
    final cartVm = _buildCartVM(rawCart);

    return PaymentPageVM(
      ctx: ctx,
      rawCart: rawCart,
      cartKey: cartKey,
      shipping: shipping,
      billing: billing,
      cart: cartVm,
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

    // ✅ 「JP」は表示しない（国コードが JP の場合は出さない）
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

    // ✅ fullName 等は拾わない。cardholderName のみ。
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
// View Models (UI へ渡す “表示用” データ)
// ============================================================

class PaymentPageVM {
  PaymentPageVM({
    required this.ctx,
    required this.rawCart,
    required this.cartKey,
    required this.shipping,
    required this.billing,
    required this.cart,
  });

  // raw（必要なら後で使えるように残す）
  final PaymentContextDTO ctx;
  final CartDTO rawCart;

  /// どのキーで cart を引いたか（必要ならログ用途）
  final String cartKey;

  // view-ready
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
