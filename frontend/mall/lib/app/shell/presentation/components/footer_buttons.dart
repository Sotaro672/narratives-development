// frontend\mall\lib\app\shell\presentation\components\footer_buttons.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:mall/features/cart/infrastructure/repository_http.dart';
import 'package:mall/features/payment/presentation/hook/use_payment.dart';

import 'package:mall/features/invoice/infrastructure/invoice_repository_http.dart';
// ✅ Case A（責務分離）: /mall/me/payments を叩く
import 'package:mall/features/payment/infrastructure/repository_http.dart';

/// ✅ /cart 用：購入する CTA（paymentへ遷移）
class GoToPaymentButton extends StatelessWidget {
  const GoToPaymentButton({
    super.key,
    required this.avatarId,
    required this.enabled,
  });

  final String avatarId;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    final aid = avatarId.trim();
    final canTap = enabled && aid.isNotEmpty;

    return SizedBox(
      height: 40,
      child: ElevatedButton.icon(
        icon: const Icon(Icons.shopping_bag_outlined, size: 20),
        label: const Text('購入する'),
        onPressed: !canTap
            ? null
            : () {
                final back = Uri(
                  path: '/cart',
                  queryParameters: {'avatarId': aid},
                ).toString();

                final uri = Uri(
                  path: '/payment',
                  queryParameters: {'avatarId': aid, 'from': back},
                );

                context.go(uri.toString());
              },
      ),
    );
  }
}

/// ✅ /payment 用：支払を確定する CTA（Order -> Invoice -> Payment を起票する）
class ConfirmPaymentButton extends StatefulWidget {
  const ConfirmPaymentButton({
    super.key,
    required this.avatarId,
    required this.enabled,
  });

  final String avatarId;
  final bool enabled;

  @override
  State<ConfirmPaymentButton> createState() => _ConfirmPaymentButtonState();
}

class _ConfirmPaymentButtonState extends State<ConfirmPaymentButton> {
  bool _loading = false;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  static String _pickBillingAddressId(Map<String, dynamic>? billingAddress) {
    if (billingAddress == null || billingAddress.isEmpty) return '';

    // backend / firestore の名揺れ吸収
    final candidates = [
      billingAddress['billingAddressId'],
      billingAddress['id'],
      billingAddress['billingId'],
    ];

    for (final c in candidates) {
      final s = _s(c);
      if (s.isNotEmpty) return s;
    }
    return '';
  }

  Future<void> _confirm() async {
    final aid = widget.avatarId.trim();
    final canTap = widget.enabled && !_loading && aid.isNotEmpty;
    if (!canTap) return;

    setState(() => _loading = true);

    // ✅ FIX: UsePaymentController は存在しないため、use_payment.dart の実体に合わせて UsePayment を使う
    final uc = UsePayment();

    final invRepo = InvoiceRepositoryHttp();
    final payRepo = PaymentRepositoryHttp();

    try {
      // ✅ Footer からでも確定できるように、必要データはここで再取得して Order 起票する
      final vm = await uc.load(qpAvatarId: aid);

      // ✅ confirmAndCreateOrder が orderId(String) を返す前提
      final orderId = (await uc.confirmAndCreateOrder(vm)).trim();
      if (orderId.isEmpty) {
        throw Exception('orderId is empty');
      }

      // ✅ billingAddressId は ctx のフィールドを最優先（payment.dart の payload に存在）
      var billingAddressId = '';
      try {
        // ignore: avoid_dynamic_calls
        billingAddressId = _s((vm.ctx as dynamic).billingAddressId);
      } catch (_) {
        billingAddressId = '';
      }
      if (billingAddressId.isEmpty) {
        // fallback（Map 内に id が入っている実装もあるため）
        billingAddressId = _pickBillingAddressId(vm.ctx.billingAddress);
      }
      if (billingAddressId.isEmpty) {
        throw Exception('billingAddressId is missing in payment context');
      }

      // ✅ backend が要求する prices を作る（各明細の line total を送る）
      final prices = <int>[];
      for (final e in vm.rawCart.items.entries) {
        final it = e.value;
        final p = it.price ?? 0;
        final q = it.qty;
        if (p > 0 && q > 0) {
          prices.add(p * q);
        }
      }
      if (prices.isEmpty) {
        throw Exception('prices is empty');
      }

      // ✅ tax / shipping は現状 0（将来 UI で入れるならここを更新）
      const tax = 0;
      const shipping = 0;

      // 1) /mall/me/invoices : invoice 起票
      // NOTE: 現状メソッド名は startCheckout のままでも OK（中身が POST /mall/me/invoices であれば）
      await invRepo.startCheckout(
        orderId: orderId,
        billingAddressId: billingAddressId,
        prices: prices,
        tax: tax,
        shipping: shipping,
      );

      // 2) /mall/me/payments : payment 起票（+ dev では自己 webhook trigger をここに寄せる想定）
      final amount = prices.fold<int>(0, (a, b) => a + b) + tax + shipping;

      await payRepo.startPayment(
        invoiceId: orderId, // invoice docId=orderId 前提
        billingAddressId: billingAddressId,
        amount: amount,
      );

      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('支払を確定しました（invoice/payment を起票しました）')),
      );
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('確定に失敗しました: $e')));
    } finally {
      invRepo.dispose();
      payRepo.dispose();
      uc.dispose();
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final aid = widget.avatarId.trim();
    final canTap = widget.enabled && !_loading && aid.isNotEmpty;

    return SizedBox(
      height: 40,
      child: ElevatedButton.icon(
        icon: _loading
            ? const SizedBox(
                width: 18,
                height: 18,
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            : const Icon(Icons.verified_outlined, size: 20),
        label: Text(_loading ? '確定中...' : '支払を確定する'),
        onPressed: !canTap ? null : _confirm,
      ),
    );
  }
}

/// ✅ catalog 用：カートに入れる CTA
class AddToCartButton extends StatefulWidget {
  const AddToCartButton({
    super.key,
    required this.from,
    required this.inventoryId,
    required this.listId,
    required this.avatarId,
    required this.enabled,
    required this.modelId,
    required this.stockCount,
  });

  final String from;

  final String inventoryId;
  final String listId;

  final String avatarId;

  final bool enabled;
  final String? modelId;
  final int? stockCount;

  @override
  State<AddToCartButton> createState() => _AddToCartButtonState();
}

class _AddToCartButtonState extends State<AddToCartButton> {
  bool _loading = false;

  Future<void> _addThenGoCart() async {
    final mid = (widget.modelId ?? '').trim();
    final sc = widget.stockCount ?? 0;
    final aid = widget.avatarId.trim();
    final invId = widget.inventoryId.trim();
    final listId = widget.listId.trim();

    final canTap =
        widget.enabled &&
        !_loading &&
        aid.isNotEmpty &&
        invId.isNotEmpty &&
        listId.isNotEmpty &&
        mid.isNotEmpty &&
        sc > 0;
    if (!canTap) return;

    setState(() => _loading = true);

    try {
      final repo = CartRepositoryHttp();
      try {
        await repo.addItem(
          avatarId: aid,
          inventoryId: invId,
          listId: listId,
          modelId: mid,
          qty: 1,
        );
      } finally {
        repo.dispose();
      }

      if (!mounted) return;

      final qp = <String, String>{'from': widget.from, 'avatarId': aid};
      final uri = Uri(path: '/cart', queryParameters: qp);

      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('カートに追加しました')));

      context.go(uri.toString());
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('追加に失敗しました: $e')));
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final mid = (widget.modelId ?? '').trim();
    final sc = widget.stockCount ?? 0;

    final invId = widget.inventoryId.trim();
    final listId = widget.listId.trim();
    final aid = widget.avatarId.trim();

    final canTap =
        widget.enabled &&
        !_loading &&
        aid.isNotEmpty &&
        invId.isNotEmpty &&
        listId.isNotEmpty &&
        mid.isNotEmpty &&
        sc > 0;

    final label = _loading
        ? '追加中...'
        : (mid.isNotEmpty && sc <= 0)
        ? '在庫なし'
        : 'カートに入れる';

    return SizedBox(
      height: 40,
      child: ElevatedButton.icon(
        icon: _loading
            ? const SizedBox(
                width: 18,
                height: 18,
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            : const Icon(Icons.add_shopping_cart_outlined, size: 20),
        label: Text(label),
        onPressed: (!canTap) ? null : _addThenGoCart,
      ),
    );
  }
}
