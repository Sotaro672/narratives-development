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

    // ✅ NEW: view model for Cart UI (title/price/listImage etc)
    required this.viewFuture,
    required this.reloadView,
  });

  final String avatarId;

  // legacy cart handler response
  final Future<CartDTO> future;

  // ✅ cart_query.go
  final Future<CartQueryDTO> cartQueryFuture;

  // ✅ preview_query.go
  final Future<PreviewQueryDTO> previewFuture;

  // ✅ Cart UI 用に整形済み（title/price/listImage を itemKey で引ける）
  final Future<CartViewDTO> viewFuture;

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

  // ✅ NEW: reload view model
  final Future<void> Function() reloadView;
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

  // ✅ Cart 画面にそのまま渡せる “表示用” view
  Future<CartViewDTO> viewFuture = Future.value(
    CartViewDTO(avatarId: '', items: const {}, raw: const {}),
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

    // ✅ view model (compose legacy + cart_query)
    viewFuture = _buildViewFuture();
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
      viewFuture = _buildViewFuture();
    });
  }

  Future<void> reloadCartQuery(SetStateFn setState) async {
    setState(() {
      cartQueryFuture = _repo.fetchCartQuery(avatarId: avatarId);
      viewFuture = _buildViewFuture();
    });
  }

  Future<void> reloadPreview(SetStateFn setState) async {
    setState(() {
      previewFuture = _repo.fetchPreview(avatarId: avatarId);
      // preview は Cart 画面表示に直結しないので view は触らない（必要なら reloadAll）
    });
  }

  Future<void> reloadView(SetStateFn setState) async {
    setState(() {
      viewFuture = _buildViewFuture();
    });
  }

  Future<void> reloadAll(SetStateFn setState) async {
    setState(() {
      future = _repo.fetchCart(avatarId: avatarId);
      cartQueryFuture = _repo.fetchCartQuery(avatarId: avatarId);
      previewFuture = _repo.fetchPreview(avatarId: avatarId);
      viewFuture = _buildViewFuture();
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

  // ✅ local int parser (do NOT rely on CartItemDTO._toInt (private in repository file))
  static int _toIntAny(dynamic v) {
    if (v == null) return 0;
    if (v is int) return v;
    if (v is double) return v.toInt();
    if (v is num) return v.toInt();
    final s = v.toString().trim();
    if (s.isEmpty) return 0;
    return int.tryParse(s) ?? 0;
  }

  // ----------------------------
  // View builder (Cart UI)
  // ----------------------------

  Future<CartViewDTO> _buildViewFuture() async {
    try {
      final results = await Future.wait<dynamic>([future, cartQueryFuture]);
      final CartDTO legacy = results[0] as CartDTO;
      final CartQueryDTO cq = results[1] as CartQueryDTO;

      final raw = cq.raw;

      // items map のあり得る場所を総当り（壊れにくく）
      Map<String, dynamic>? itemsMap;

      dynamic items0 = raw['items'];
      if (items0 is Map) itemsMap = items0.cast<String, dynamic>();

      if (itemsMap == null && raw['cart'] is Map) {
        final c = (raw['cart'] as Map).cast<String, dynamic>();
        final it = c['items'];
        if (it is Map) itemsMap = it.cast<String, dynamic>();
      }

      if (itemsMap == null && raw['data'] is Map) {
        final d = (raw['data'] as Map).cast<String, dynamic>();
        final it = d['items'];
        if (it is Map) itemsMap = it.cast<String, dynamic>();
        // さらに { data: { cart: { items: ... } } } もあり得る
        if (itemsMap == null && d['cart'] is Map) {
          final c = (d['cart'] as Map).cast<String, dynamic>();
          final it2 = c['items'];
          if (it2 is Map) itemsMap = it2.cast<String, dynamic>();
        }
      }

      final outItems = <String, CartViewItemDTO>{};

      // まず cart_query 側（表示フィールド）を基準に作る
      if (itemsMap != null) {
        for (final entry in itemsMap.entries) {
          final key = entry.key.toString().trim();
          if (key.isEmpty) continue;

          final v = entry.value;
          if (v is! Map) continue;
          final m = v.cast<String, dynamic>();

          String s(dynamic x) => (x ?? '').toString().trim();
          int? iN(dynamic x) {
            if (x == null) return null;
            final n = _toIntAny(x);
            return n == 0 ? null : n;
          }

          final title = s(m['title']);
          final listImage = s(m['listImage'] ?? m['imageId'] ?? m['imageID']);
          final price = iN(m['price']);
          final productName = s(m['productName']);
          final size = s(m['size']);
          final color = s(m['color']);
          final qty0 = _toIntAny(m['qty'] ?? m['quantity']);

          // identifiers (server が返していれば使う)
          final invId = s(m['inventoryId']);
          final listId = s(m['listId']);
          final modelId = s(m['modelId']);

          // legacy から補完
          final legacyIt = legacy.items[key];
          final inv = invId.isNotEmpty ? invId : (legacyIt?.inventoryId ?? '');
          final lid = listId.isNotEmpty ? listId : (legacyIt?.listId ?? '');
          final mid = modelId.isNotEmpty ? modelId : (legacyIt?.modelId ?? '');

          final qty = qty0 > 0 ? qty0 : (legacyIt?.qty ?? 0);

          outItems[key] = CartViewItemDTO(
            itemKey: key,
            inventoryId: inv,
            listId: lid,
            modelId: mid,
            title: title,
            listImage: listImage,
            price: price,
            productName: productName,
            size: size,
            color: color,
            qty: qty,
            raw: m,
          );
        }
      }

      // cart_query が items を返さない / 空の場合でも legacy は表示できるように補完
      if (outItems.isEmpty) {
        for (final entry in legacy.items.entries) {
          final key = entry.key.trim();
          if (key.isEmpty) continue;
          final it = entry.value;
          outItems[key] = CartViewItemDTO(
            itemKey: key,
            inventoryId: it.inventoryId,
            listId: it.listId,
            modelId: it.modelId,
            title: '',
            listImage: '',
            price: null,
            productName: '',
            size: '',
            color: '',
            qty: it.qty,
            raw: const {},
          );
        }
      }

      // avatarId
      final aid = legacy.avatarId.trim().isNotEmpty
          ? legacy.avatarId.trim()
          : avatarId;

      return CartViewDTO(avatarId: aid, items: outItems, raw: raw);
    } catch (_) {
      // worst-case: 画面が落ちないように空を返す
      return CartViewDTO(avatarId: avatarId, items: const {}, raw: const {});
    }
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
        viewFuture = _buildViewFuture();
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
        viewFuture = _buildViewFuture();
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
        viewFuture = _buildViewFuture();
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
      viewFuture: viewFuture,
      busy: busy,
      reload: () => reload(setState),
      reloadCartQuery: () => reloadCartQuery(setState),
      reloadPreview: () => reloadPreview(setState),
      reloadView: () => reloadView(setState),
      inc: (itemKey) => inc(setState, itemKey),
      dec: (itemKey, currentQty) => dec(setState, itemKey, currentQty),
      remove: (itemKey) => remove(setState, itemKey),
      clear: () => clear(setState),
    );
  }
}

// ----------------------------
// View Models (Cart UI)
// ----------------------------

class CartViewDTO {
  CartViewDTO({required this.avatarId, required this.items, required this.raw});

  final String avatarId;

  /// itemKey -> display item
  final Map<String, CartViewItemDTO> items;

  /// cart_query raw response (debug / fallback)
  final Map<String, dynamic> raw;
}

class CartViewItemDTO {
  CartViewItemDTO({
    required this.itemKey,
    required this.inventoryId,
    required this.listId,
    required this.modelId,
    required this.title,
    required this.listImage,
    required this.price,
    required this.productName,
    required this.size,
    required this.color,
    required this.qty,
    required this.raw,
  });

  final String itemKey;

  // identifiers (mutations)
  final String inventoryId;
  final String listId;
  final String modelId;

  // display fields
  final String title;
  final String listImage;
  final int? price;
  final String productName;
  final String size;
  final String color;

  final int qty;

  /// row/item raw map (best-effort)
  final Map<String, dynamic> raw;
}
