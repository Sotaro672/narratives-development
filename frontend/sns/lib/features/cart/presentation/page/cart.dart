// frontend/sns/lib/features/cart/presentation/page/cart.dart
import 'package:flutter/material.dart';

import '../../infrastructure/cart_repository_http.dart';
import '../hook/use_cart.dart';

/// Cart page (buyer-facing).
///
/// ✅ style only: state/logic is delegated to UseCartController
class CartPage extends StatefulWidget {
  const CartPage({super.key, required this.avatarId, this.from});

  final String avatarId;
  final String? from;

  @override
  State<CartPage> createState() => _CartPageState();
}

class _CartPageState extends State<CartPage> {
  UseCartController? _ctl;

  String get _avatarId => widget.avatarId.trim();

  void _ensureController() {
    final avatarId = _avatarId;
    if (avatarId.isEmpty) return;

    // 既に初期化済みなら何もしない
    if (_ctl != null) return;

    _ctl = UseCartController(avatarId: avatarId, context: context)..init();
  }

  void _disposeController() {
    _ctl?.dispose();
    _ctl = null;
  }

  @override
  void initState() {
    super.initState();
    // ✅ avatarId が空のときは init しない（空でAPIを叩くとUncaught Errorの原因）
    _ensureController();
  }

  @override
  void didUpdateWidget(covariant CartPage oldWidget) {
    super.didUpdateWidget(oldWidget);

    final prev = (oldWidget.avatarId).trim();
    final next = _avatarId;

    // avatarId が変わったら作り直す
    if (prev != next) {
      _disposeController();
      _ensureController();
      if (mounted) setState(() {});
    }
  }

  @override
  void dispose() {
    _disposeController();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final avatarId = _avatarId;

    // ✅ avatarId が無ければ controller を作らず、APIも叩かない
    if (avatarId.isEmpty) {
      return const Center(child: Text('avatarId is required'));
    }

    // ここに来た時点で avatarId はあるはずなので、念のため初期化
    _ensureController();

    // ✅ SetStateFn: void Function(VoidCallback fn)
    void safeSetState(VoidCallback fn) {
      if (!mounted) return;
      setState(fn);
    }

    final ctl = _ctl;
    if (ctl == null) {
      // 念のため（通常ここには来ない）
      return const Center(child: CircularProgressIndicator());
    }

    final vm = ctl.buildResult(safeSetState);

    return FutureBuilder<CartDTO>(
      future: vm.future,
      builder: (context, snap) {
        final cart = snap.data;

        return Stack(
          children: [
            ListView(
              padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
              children: [
                _HeaderCard(
                  avatarId: avatarId,
                  totalQty: cart?.totalQty() ?? 0,
                  expiresAt: cart?.expiresAt,
                  ordered: cart?.ordered ?? false,
                  onReload: vm.reload,
                ),
                const SizedBox(height: 12),

                if (snap.connectionState == ConnectionState.waiting &&
                    cart == null)
                  const _LoadingCard()
                else if (snap.hasError)
                  _ErrorCard(
                    errorText: snap.error.toString(),
                    onRetry: vm.reload,
                  )
                else
                  _ItemsCard(
                    cart:
                        cart ??
                        CartDTO(
                          avatarId: avatarId,
                          items: const {},
                          createdAt: null,
                          updatedAt: null,
                          expiresAt: null,
                          ordered: false,
                        ),
                    onInc: vm.inc,
                    onDec: vm.dec,
                    onRemove: vm.remove,
                    onClear: vm.clear,
                  ),
              ],
            ),

            if (vm.busy)
              Positioned.fill(
                child: IgnorePointer(
                  ignoring: true,
                  child: Container(
                    color: Colors.black.withValues(alpha: 0.06),
                    child: const Center(child: CircularProgressIndicator()),
                  ),
                ),
              ),
          ],
        );
      },
    );
  }
}

// ------------------------------------------------------------
// UI parts (style only)
// ------------------------------------------------------------

class _HeaderCard extends StatelessWidget {
  const _HeaderCard({
    required this.avatarId,
    required this.totalQty,
    required this.expiresAt,
    required this.ordered,
    required this.onReload,
  });

  final String avatarId;
  final int totalQty;
  final DateTime? expiresAt;
  final bool ordered;
  final Future<void> Function() onReload;

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;
    final exp = expiresAt;
    final expText = exp == null ? '-' : exp.toLocal().toString();

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Cart', style: t.titleMedium),
                  const SizedBox(height: 6),
                  Text('avatarId: $avatarId', style: t.bodySmall),
                  const SizedBox(height: 4),
                  Text('items: $totalQty', style: t.bodySmall),
                  const SizedBox(height: 4),
                  Text('expiresAt: $expText', style: t.bodySmall),
                  if (ordered) ...[
                    const SizedBox(height: 6),
                    Text('ordered: true', style: t.bodySmall),
                  ],
                ],
              ),
            ),
            IconButton(
              tooltip: 'Reload',
              onPressed: () => onReload(),
              icon: const Icon(Icons.refresh),
            ),
          ],
        ),
      ),
    );
  }
}

class _LoadingCard extends StatelessWidget {
  const _LoadingCard();

  @override
  Widget build(BuildContext context) {
    return const Card(
      child: Padding(
        padding: EdgeInsets.all(18),
        child: Center(child: CircularProgressIndicator()),
      ),
    );
  }
}

class _ErrorCard extends StatelessWidget {
  const _ErrorCard({required this.errorText, required this.onRetry});

  final String errorText;
  final Future<void> Function() onRetry;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('Error'),
            const SizedBox(height: 6),
            Text(errorText),
            const SizedBox(height: 10),
            Align(
              alignment: Alignment.centerRight,
              child: OutlinedButton(
                onPressed: () => onRetry(),
                child: const Text('Retry'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ItemsCard extends StatelessWidget {
  const _ItemsCard({
    required this.cart,
    required this.onInc,
    required this.onDec,
    required this.onRemove,
    required this.onClear,
  });

  final CartDTO cart;

  final Future<void> Function(String modelId) onInc;
  final Future<void> Function(String modelId, int currentQty) onDec;
  final Future<void> Function(String modelId) onRemove;
  final Future<void> Function() onClear;

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;

    final entries = cart.items.entries.toList()
      ..sort((a, b) => a.key.compareTo(b.key));

    return Card(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 8),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text('Items', style: t.titleMedium),
                const Spacer(),
                TextButton.icon(
                  onPressed: entries.isEmpty ? null : () => onClear(),
                  icon: const Icon(Icons.delete_outline),
                  label: const Text('Clear'),
                ),
              ],
            ),
            const SizedBox(height: 8),

            if (entries.isEmpty)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 14),
                child: Text('カートは空です', style: t.bodyMedium),
              )
            else
              ListView.separated(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                itemCount: entries.length,
                separatorBuilder: (_, __) => Divider(
                  height: 1,
                  thickness: 1,
                  color: Theme.of(context).dividerColor.withValues(alpha: 0.4),
                ),
                itemBuilder: (context, i) {
                  final e = entries[i];
                  final modelId = e.key;
                  final qty = e.value;

                  return ListTile(
                    contentPadding: const EdgeInsets.symmetric(horizontal: 4),
                    title: Text(
                      modelId,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    subtitle: Text('qty: $qty'),
                    trailing: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        IconButton(
                          tooltip: 'Remove',
                          onPressed: () => onRemove(modelId),
                          icon: const Icon(Icons.close),
                        ),
                        IconButton(
                          tooltip: '-',
                          onPressed: qty <= 1
                              ? () => onRemove(modelId)
                              : () => onDec(modelId, qty),
                          icon: const Icon(Icons.remove_circle_outline),
                        ),
                        Text('$qty', style: t.titleSmall),
                        IconButton(
                          tooltip: '+',
                          onPressed: () => onInc(modelId),
                          icon: const Icon(Icons.add_circle_outline),
                        ),
                      ],
                    ),
                  );
                },
              ),
            const SizedBox(height: 10),
          ],
        ),
      ),
    );
  }
}
