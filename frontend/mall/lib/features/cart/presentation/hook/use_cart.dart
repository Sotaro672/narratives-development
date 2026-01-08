//frontend\sns\lib\features\cart\presentation\hook\use_cart.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import '../../infrastructure/cart_repository_http.dart';

// ✅ cart.dart から DTO 型を使えるようにする（repository を直接 import しなくてよくなる）
// NOTE: export は「型だけ」に限定（HTTP 実装や例外型を不用意に public にしない）
export '../../infrastructure/cart_repository_http.dart'
    show CartDTO, CartItemDTO;

typedef SetStateFn = void Function(VoidCallback fn);

/// ✅ Cart hook result (CartDTO / CartItemDTO only)
class UseCartResult {
  UseCartResult({
    required this.avatarId,
    required this.future,
    required this.current,
    required this.busy,
    required this.reload,
    required this.inc,
    required this.dec,
    required this.remove,
    required this.clear,
  });

  final String avatarId;

  /// Source of truth: GET /mall/cart?avatarId=...
  final Future<CartDTO> future;

  /// ✅ “今のカート” を常に保持（FutureBuilder を使わない UI でも落ちない）
  /// - init 時点では empty cart
  /// - reload / CRUD 後はここが更新される
  final CartDTO current;

  final bool busy;

  /// Reload cart (re-fetch)
  final Future<void> Function() reload;

  /// itemKey を受け取る（items は itemKey -> CartItemDTO）
  final Future<void> Function(String itemKey) inc;
  final Future<void> Function(String itemKey, int currentQty) dec;
  final Future<void> Function(String itemKey) remove;

  final Future<void> Function() clear;
}

/// Hook-like controller for CartPage.
/// - Holds repository/client lifecycle
/// - Exposes state + handlers only
///
/// ✅ This file handles ONLY CartDTO / CartItemDTO.
class UseCartController {
  UseCartController({required this.avatarId, required this.context});

  final String avatarId;
  final BuildContext context;

  late final CartRepositoryHttp _repo;

  bool get _dbg => kDebugMode;

  void _log(String msg) {
    if (!_dbg) return;
    // ignore: avoid_print
    print('[UseCart] $msg');
  }

  String _mask(String s) {
    final t = s.trim();
    if (t.isEmpty) return '';
    if (t.length <= 6) return t;
    return '${t.substring(0, 3)}***${t.substring(t.length - 3)}';
  }

  String _prettyJson(dynamic v) {
    try {
      return const JsonEncoder.withIndent('  ').convert(v);
    } catch (_) {
      return v.toString();
    }
  }

  CartDTO _emptyCart() => CartDTO(
    avatarId: avatarId.trim(),
    items: const {},
    createdAt: null,
    updatedAt: null,
    expiresAt: null,
  );

  /// ✅ 常に “最後に成功したカート” を保持（UIが Future に依存しないで済む）
  CartDTO current = CartDTO(
    avatarId: '',
    items: const {},
    createdAt: null,
    updatedAt: null,
    expiresAt: null,
  );

  Future<CartDTO> future = Future.value(
    CartDTO(
      avatarId: '',
      items: const {},
      createdAt: null,
      updatedAt: null,
      expiresAt: null,
    ),
  );

  bool busy = false;

  void init() {
    _repo = CartRepositoryHttp();

    final aid = avatarId.trim();
    current = aid.isEmpty ? _emptyCart() : _emptyCart();

    _log('init avatarId="${_mask(avatarId)}"');

    // ✅ future をセット（成功したら current にも反映）
    future = _repo
        .fetchCart(avatarId: avatarId)
        .then((c) {
          current = c;
          return c;
        })
        .catchError((_) {
          // ✅ 失敗しても UI を落とさない（current は empty のまま）
          return current;
        });

    _attachDebugLogging();
  }

  void dispose() {
    _log('dispose');
    _repo.dispose();
  }

  void _attachDebugLogging() {
    future
        .then((c) {
          _log(
            'cart OK: avatarId="${_mask(c.avatarId)}" items=${c.items.length} '
            'createdAt=${c.createdAt?.toIso8601String()} '
            'updatedAt=${c.updatedAt?.toIso8601String()} '
            'expiresAt=${c.expiresAt?.toIso8601String()}',
          );

          if (c.items.isNotEmpty) {
            final k0 = c.items.keys.first;
            final it0 = c.items[k0]!;
            _log(
              'cart item0: key="$k0" inv="${it0.inventoryId}" list="${it0.listId}" model="${it0.modelId}" '
              'qty=${it0.qty} title="${it0.title}" size="${it0.size}" color="${it0.color}"',
            );
          } else {
            _log('cart items empty');
          }

          if (_dbg) {
            final sample = <String, dynamic>{
              'avatarId': c.avatarId,
              'itemsCount': c.items.length,
              'itemsSample': c.items.isEmpty
                  ? null
                  : {
                      c.items.keys.first: {
                        'inventoryId': c.items.values.first.inventoryId,
                        'listId': c.items.values.first.listId,
                        'modelId': c.items.values.first.modelId,
                        'qty': c.items.values.first.qty,
                        'title': c.items.values.first.title,
                        'size': c.items.values.first.size,
                        'color': c.items.values.first.color,
                        // 追加フィールドも来ている前提（nullでも落ちない）
                        'listImage': c.items.values.first.listImage,
                        'price': c.items.values.first.price,
                        'productName': c.items.values.first.productName,
                      },
                    },
            };
            _log('cart sample(json)=\n${_prettyJson(sample)}');
          }
        })
        .catchError((e, st) {
          _log('cart ERROR: $e');
          _log('stack:\n$st');
          return null;
        });
  }

  Future<void> reload(SetStateFn setState) async {
    _log('reload');

    setState(() {
      future = _repo
          .fetchCart(avatarId: avatarId)
          .then((c) {
            current = c;
            return c;
          })
          .catchError((_) {
            // ✅ 失敗しても落とさない（current は維持）
            return current;
          });
    });

    _attachDebugLogging();
  }

  Future<void> _withBusy(
    SetStateFn setState,
    Future<void> Function() fn,
  ) async {
    if (busy) {
      _log('_withBusy: already busy -> skip');
      return;
    }

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

  CartItemDTO? _getItemFromKey(CartDTO c, String itemKey) {
    final key = itemKey.trim();
    if (key.isEmpty) return null;
    return c.items[key];
  }

  Future<void> inc(SetStateFn setState, String itemKey) async {
    await _withBusy(setState, () async {
      final base = await future; // ✅ 失敗しても future は current を返す
      final it = _getItemFromKey(base, itemKey);
      if (it == null) return;

      final c = await _repo.addItem(
        avatarId: avatarId,
        inventoryId: it.inventoryId,
        listId: it.listId,
        modelId: it.modelId,
        qty: 1,
      );
      if (!context.mounted) return;

      setState(() {
        current = c;
        future = Future.value(c);
      });

      _attachDebugLogging();
    });
  }

  Future<void> dec(SetStateFn setState, String itemKey, int currentQty) async {
    await _withBusy(setState, () async {
      final next = currentQty - 1;

      final base = await future;
      final it = _getItemFromKey(base, itemKey);
      if (it == null) return;

      final c = await _repo.setItemQty(
        avatarId: avatarId,
        inventoryId: it.inventoryId,
        listId: it.listId,
        modelId: it.modelId,
        qty: next,
      );
      if (!context.mounted) return;

      setState(() {
        current = c;
        future = Future.value(c);
      });

      _attachDebugLogging();
    });
  }

  Future<void> remove(SetStateFn setState, String itemKey) async {
    await _withBusy(setState, () async {
      final base = await future;
      final it = _getItemFromKey(base, itemKey);
      if (it == null) return;

      final c = await _repo.removeItem(
        avatarId: avatarId,
        inventoryId: it.inventoryId,
        listId: it.listId,
        modelId: it.modelId,
      );
      if (!context.mounted) return;

      setState(() {
        current = c;
        future = Future.value(c);
      });

      _attachDebugLogging();
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
      await reload(setState);
    });
  }

  UseCartResult buildResult(SetStateFn setState) {
    return UseCartResult(
      avatarId: avatarId,
      future: future,
      current: current,
      busy: busy,
      reload: () => reload(setState),
      inc: (itemKey) => inc(setState, itemKey),
      dec: (itemKey, currentQty) => dec(setState, itemKey, currentQty),
      remove: (itemKey) => remove(setState, itemKey),
      clear: () => clear(setState),
    );
  }
}
