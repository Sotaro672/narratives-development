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
  });

  final String avatarId;

  final Future<CartDTO> future;
  final bool busy;

  final Future<void> Function() reload;
  final Future<void> Function(String modelId) inc;
  final Future<void> Function(String modelId, int currentQty) dec;
  final Future<void> Function(String modelId) remove;
  final Future<void> Function() clear;
}

/// Hook-like controller for CartPage.
/// - Holds repository/client lifecycle
/// - Exposes state + handlers only
class UseCartController {
  UseCartController({required this.avatarId, required this.context});

  final String avatarId;
  final BuildContext context;

  late final CartRepositoryHttp _repo;

  Future<CartDTO> future = Future.value(
    CartDTO(
      avatarId: '',
      items: const {},
      createdAt: null,
      updatedAt: null,
      expiresAt: null,
      ordered: false,
    ),
  );

  bool busy = false;

  void init() {
    _repo = CartRepositoryHttp();
    future = _repo.fetchCart(avatarId: avatarId);
  }

  void dispose() {
    _repo.dispose();
  }

  Future<void> reload(SetStateFn setState) async {
    setState(() {
      future = _repo.fetchCart(avatarId: avatarId);
    });
  }

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

  Future<void> inc(SetStateFn setState, String modelId) async {
    await _withBusy(setState, () async {
      final c = await _repo.addItem(
        avatarId: avatarId,
        modelId: modelId,
        qty: 1,
      );
      if (!context.mounted) return;

      setState(() {
        future = Future.value(c);
      });
    });
  }

  Future<void> dec(SetStateFn setState, String modelId, int currentQty) async {
    await _withBusy(setState, () async {
      final next = currentQty - 1;
      final c = await _repo.setItemQty(
        avatarId: avatarId,
        modelId: modelId,
        qty: next,
      );
      if (!context.mounted) return;

      setState(() {
        future = Future.value(c);
      });
    });
  }

  Future<void> remove(SetStateFn setState, String modelId) async {
    await _withBusy(setState, () async {
      final c = await _repo.removeItem(avatarId: avatarId, modelId: modelId);
      if (!context.mounted) return;

      setState(() {
        future = Future.value(c);
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
      await reload(setState);
    });
  }

  UseCartResult buildResult(SetStateFn setState) {
    return UseCartResult(
      avatarId: avatarId,
      future: future,
      busy: busy,
      reload: () => reload(setState),
      inc: (modelId) => inc(setState, modelId),
      dec: (modelId, currentQty) => dec(setState, modelId, currentQty),
      remove: (modelId) => remove(setState, modelId),
      clear: () => clear(setState),
    );
  }
}
