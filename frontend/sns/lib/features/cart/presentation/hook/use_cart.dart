// frontend/sns/lib/features/cart/presentation/hook/use_cart.dart
import 'package:flutter/material.dart';

import '../../infrastructure/cart_repository_http.dart';

typedef SetStateFn = void Function(VoidCallback fn);

class UseCartResult {
  UseCartResult({
    required this.avatarId,
    required this.future,
    required this.busy,
    required this.reload,
    required this.inc,
    required this.dec,
    required this.remove,
    required this.clear,

    // ✅ NEW: cart_query / preview_query
    required this.cartQueryFuture,
    required this.previewFuture,
    required this.reloadCartQuery,
    required this.reloadPreview,
  });

  final String avatarId;

  // legacy cart handler response
  final Future<CartDTO> future;

  // ✅ cart_query.go
  final Future<CartQueryDTO> cartQueryFuture;

  // ✅ preview_query.go
  final Future<PreviewQueryDTO> previewFuture;

  final bool busy;

  /// itemKey を受け取る（items は itemKey -> CartItemDTO になったため）
  final Future<void> Function(String itemKey) inc;
  final Future<void> Function(String itemKey, int currentQty) dec;
  final Future<void> Function(String itemKey) remove;

  final Future<void> Function() reload; // legacy cart
  final Future<void> Function() clear;

  // ✅ NEW: reload query models
  final Future<void> Function() reloadCartQuery;
  final Future<void> Function() reloadPreview;
}

/// Hook-like controller for CartPage.
/// - Holds repository/client lifecycle
/// - Exposes state + handlers only
class UseCartController {
  UseCartController({required this.avatarId, required this.context});

  final String avatarId;
  final BuildContext context;

  late final CartRepositoryHttp _repo;

  // ----------------------------
  // State
  // ----------------------------

  Future<CartDTO> future = Future.value(
    CartDTO(
      avatarId: '',
      items: const {},
      createdAt: null,
      updatedAt: null,
      expiresAt: null,
    ),
  );

  // ✅ cart_query / preview_query state
  Future<CartQueryDTO> cartQueryFuture = Future.value(
    CartQueryDTO(raw: const {}, cart: null, rows: const []),
  );

  Future<PreviewQueryDTO> previewFuture = Future.value(
    PreviewQueryDTO(
      raw: const {},
      avatarId: '',
      cart: null,
      rows: const [],
      total: null,
      subtotal: null,
      shippingFee: null,
      tax: null,
    ),
  );

  bool busy = false;

  // ----------------------------
  // Lifecycle
  // ----------------------------

  void init() {
    _repo = CartRepositoryHttp();

    // legacy cart
    future = _repo.fetchCart(avatarId: avatarId);

    // ✅ read-models
    cartQueryFuture = _repo.fetchCartQuery(avatarId: avatarId);
    previewFuture = _repo.fetchPreview(avatarId: avatarId);
  }

  void dispose() {
    _repo.dispose();
  }

  // ----------------------------
  // Reloaders
  // ----------------------------

  Future<void> reload(SetStateFn setState) async {
    setState(() {
      future = _repo.fetchCart(avatarId: avatarId);
    });
  }

  Future<void> reloadCartQuery(SetStateFn setState) async {
    setState(() {
      cartQueryFuture = _repo.fetchCartQuery(avatarId: avatarId);
    });
  }

  Future<void> reloadPreview(SetStateFn setState) async {
    setState(() {
      previewFuture = _repo.fetchPreview(avatarId: avatarId);
    });
  }

  Future<void> reloadAll(SetStateFn setState) async {
    setState(() {
      future = _repo.fetchCart(avatarId: avatarId);
      cartQueryFuture = _repo.fetchCartQuery(avatarId: avatarId);
      previewFuture = _repo.fetchPreview(avatarId: avatarId);
    });
  }

  // ----------------------------
  // Busy wrapper
  // ----------------------------

  Future<void> _withBusy(
    SetStateFn setState,
    Future<void> Function() fn,
  ) async {
    if (busy) return;

    setState(() {
      busy = true;
    });

    try {
      await fn();
    } finally {
      if (context.mounted) {
        setState(() {
          busy = false;
        });
      }
    }
  }

  // ----------------------------
  // Helpers
  // ----------------------------

  CartItemDTO? _getItemFromKey(CartDTO c, String itemKey) {
    final key = itemKey.trim();
    if (key.isEmpty) return null;
    return c.items[key];
  }

  // For query/preview UIs: prefer rows, fallback to cart.items
  CartQueryRowDTO? _getRowFromKey(List<CartQueryRowDTO> rows, String itemKey) {
    final key = itemKey.trim();
    if (key.isEmpty) return null;
    for (final r in rows) {
      if (r.itemKey.trim() == key) return r;
    }
    return null;
  }

  // ----------------------------
  // Mutations
  // ----------------------------

  Future<void> inc(SetStateFn setState, String itemKey) async {
    await _withBusy(setState, () async {
      // まず legacy cart から引く（常に存在する前提）
      final current = await future;
      final it = _getItemFromKey(current, itemKey);

      // legacy が空の場合は cart_query rows から best-effort で拾う
      CartQueryRowDTO? row;
      if (it == null) {
        final cq = await cartQueryFuture;
        row = _getRowFromKey(cq.rows, itemKey);
        if (row == null) return;
      }

      final c = await _repo.addItem(
        avatarId: avatarId,
        inventoryId: it?.inventoryId ?? row!.inventoryId,
        listId: it?.listId ?? row!.listId,
        modelId: it?.modelId ?? row!.modelId,
        qty: 1,
      );
      if (!context.mounted) return;

      // ✅ mutation 後は 3つとも更新して揃える（UI がどれを参照しても破綻しない）
      setState(() {
        future = Future.value(c);
        cartQueryFuture = _repo.fetchCartQuery(avatarId: avatarId);
        previewFuture = _repo.fetchPreview(avatarId: avatarId);
      });
    });
  }

  Future<void> dec(SetStateFn setState, String itemKey, int currentQty) async {
    await _withBusy(setState, () async {
      final next = currentQty - 1;

      final current = await future;
      final it = _getItemFromKey(current, itemKey);

      CartQueryRowDTO? row;
      if (it == null) {
        final cq = await cartQueryFuture;
        row = _getRowFromKey(cq.rows, itemKey);
        if (row == null) return;
      }

      // qty<=0 は backend が remove 扱い
      final c = await _repo.setItemQty(
        avatarId: avatarId,
        inventoryId: it?.inventoryId ?? row!.inventoryId,
        listId: it?.listId ?? row!.listId,
        modelId: it?.modelId ?? row!.modelId,
        qty: next,
      );
      if (!context.mounted) return;

      setState(() {
        future = Future.value(c);
        cartQueryFuture = _repo.fetchCartQuery(avatarId: avatarId);
        previewFuture = _repo.fetchPreview(avatarId: avatarId);
      });
    });
  }

  Future<void> remove(SetStateFn setState, String itemKey) async {
    await _withBusy(setState, () async {
      final current = await future;
      final it = _getItemFromKey(current, itemKey);

      CartQueryRowDTO? row;
      if (it == null) {
        final cq = await cartQueryFuture;
        row = _getRowFromKey(cq.rows, itemKey);
        if (row == null) return;
      }

      final c = await _repo.removeItem(
        avatarId: avatarId,
        inventoryId: it?.inventoryId ?? row!.inventoryId,
        listId: it?.listId ?? row!.listId,
        modelId: it?.modelId ?? row!.modelId,
      );
      if (!context.mounted) return;

      setState(() {
        future = Future.value(c);
        cartQueryFuture = _repo.fetchCartQuery(avatarId: avatarId);
        previewFuture = _repo.fetchPreview(avatarId: avatarId);
      });
    });
  }

  Future<void> clear(SetStateFn setState) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('カートを空にしますか？'),
        content: const Text('この操作は元に戻せません。'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('キャンセル'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('空にする'),
          ),
        ],
      ),
    );

    if (ok != true) return;

    await _withBusy(setState, () async {
      await _repo.clearCart(avatarId: avatarId);
      if (!context.mounted) return;

      // ✅ clear 後は全部 reload
      await reloadAll(setState);
    });
  }

  // ----------------------------
  // Result builder
  // ----------------------------

  UseCartResult buildResult(SetStateFn setState) {
    return UseCartResult(
      avatarId: avatarId,
      future: future,
      cartQueryFuture: cartQueryFuture,
      previewFuture: previewFuture,
      busy: busy,
      reload: () => reload(setState),
      reloadCartQuery: () => reloadCartQuery(setState),
      reloadPreview: () => reloadPreview(setState),
      inc: (itemKey) => inc(setState, itemKey),
      dec: (itemKey, currentQty) => dec(setState, itemKey, currentQty),
      remove: (itemKey) => remove(setState, itemKey),
      clear: () => clear(setState),
    );
  }
}
