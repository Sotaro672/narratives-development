//frontend\mall\lib\features\cart\presentation\hook\use_cart.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import '../../../../app/routing/navigation.dart';
import '../../infrastructure/repository_http.dart';

// ✅ cart.dart から DTO 型を使えるようにする（repository を直接 import しなくてよくなる）
// NOTE: export は「型だけ」に限定（HTTP 実装や例外型を不用意に public にしない）
export '../../infrastructure/repository_http.dart' show CartDTO, CartItemDTO;

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

  /// ✅ Source of truth: AvatarIdStore（URL からは読まない）
  final String avatarId;

  /// Source of truth: GET /mall/me/cart?avatarId=...
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
/// ✅ avatarId は URL ではなく AvatarIdStore から解決する。
class UseCartController {
  UseCartController({required this.context, String? avatarId})
    : _initialAvatarId = (avatarId ?? '').trim();

  final BuildContext context;

  /// 任意：呼び出し側が明示的に渡したい場合のみ使用（URL由来は不可推奨）
  final String _initialAvatarId;

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

  /// ✅ 実際に API 呼び出しに使う avatarId（AvatarIdStore で確定した値）
  String _avatarId = '';
  String get avatarId => _avatarId;

  CartDTO _emptyCart(String aid) => CartDTO(
    avatarId: aid.trim(),
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

  /// ✅ AvatarIdStore から avatarId を確定する（URLは使わない）
  Future<String> _ensureAvatarId() async {
    // 1) 呼び出し側から明示的に渡されたもの（URL由来でない前提）
    final a0 = _initialAvatarId.trim();
    if (a0.isNotEmpty) {
      AvatarIdStore.I.set(a0);
      return a0;
    }

    // 2) 既に store にあるならそれ
    final storeId = AvatarIdStore.I.avatarId.trim();
    if (storeId.isNotEmpty) return storeId;

    // 3) サーバで /mall/me/avatar を叩いて解決
    final resolved = await AvatarIdStore.I.resolveMyAvatarId();
    final a1 = (resolved ?? '').trim();
    if (a1.isNotEmpty) {
      AvatarIdStore.I.set(a1);
      return a1;
    }

    return '';
  }

  void init() {
    _repo = CartRepositoryHttp();

    // init 時点では empty を入れておく（落ちない）
    current = _emptyCart('');
    future = Future.value(current);

    _log('init (avatarId will be resolved from AvatarIdStore)');

    // ✅ future を「avatarId 解決 -> fetchCart」チェーンにする
    future = _ensureAvatarId()
        .then((aid) async {
          _avatarId = aid.trim();
          if (_avatarId.isEmpty) {
            current = _emptyCart('');
            return current;
          }

          // 初回取得
          final c = await _repo.fetchCart(avatarId: _avatarId);
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

  Future<void> _ensureReadyIfNeeded(SetStateFn setState) async {
    if (_avatarId.trim().isNotEmpty) return;

    final aid = await _ensureAvatarId();
    if (!context.mounted) return;

    if (aid.trim().isEmpty) {
      // 取れない場合は empty のまま
      setState(() {
        _avatarId = '';
        current = _emptyCart('');
        future = Future.value(current);
      });
      return;
    }

    setState(() {
      _avatarId = aid.trim();
    });
  }

  Future<void> reload(SetStateFn setState) async {
    _log('reload');

    await _ensureReadyIfNeeded(setState);
    final aid = _avatarId.trim();
    if (aid.isEmpty) return;

    setState(() {
      future = _repo
          .fetchCart(avatarId: aid)
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
      await _ensureReadyIfNeeded(setState);
      final aid = _avatarId.trim();
      if (aid.isEmpty) return;

      final base = await future; // ✅ 失敗しても future は current を返す
      final it = _getItemFromKey(base, itemKey);
      if (it == null) return;

      final c = await _repo.addItem(
        avatarId: aid,
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
      await _ensureReadyIfNeeded(setState);
      final aid = _avatarId.trim();
      if (aid.isEmpty) return;

      final next = currentQty - 1;

      final base = await future;
      final it = _getItemFromKey(base, itemKey);
      if (it == null) return;

      final c = await _repo.setItemQty(
        avatarId: aid,
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
      await _ensureReadyIfNeeded(setState);
      final aid = _avatarId.trim();
      if (aid.isEmpty) return;

      final base = await future;
      final it = _getItemFromKey(base, itemKey);
      if (it == null) return;

      final c = await _repo.removeItem(
        avatarId: aid,
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
      await _ensureReadyIfNeeded(setState);
      final aid = _avatarId.trim();
      if (aid.isEmpty) return;

      await _repo.clearCart(avatarId: aid);
      if (!context.mounted) return;
      await reload(setState);
    });
  }

  UseCartResult buildResult(SetStateFn setState) {
    final aid = _avatarId.trim().isNotEmpty
        ? _avatarId.trim()
        : AvatarIdStore.I.avatarId.trim();

    return UseCartResult(
      avatarId: aid,
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
