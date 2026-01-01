// frontend/sns/lib/features/cart/presentation/page/cart.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../infrastructure/cart_repository_http.dart';
import '../hook/use_cart.dart';

// ✅ routes
import '../../../../app/routing/routes.dart';

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

    // already initialized
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
    _ensureController();
  }

  @override
  void didUpdateWidget(covariant CartPage oldWidget) {
    super.didUpdateWidget(oldWidget);

    final prev = oldWidget.avatarId.trim();
    final next = _avatarId;

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

  CartDTO _emptyCart(String avatarId) {
    return CartDTO(
      avatarId: avatarId,
      items: const {},
      createdAt: null,
      updatedAt: null,
      expiresAt: null,
    );
  }

  bool _isCartNotFound(Object? err) {
    if (err is CartHttpException) {
      return err.statusCode == 404;
    }
    final s = err?.toString() ?? '';
    return s.contains('statusCode=404') || s.contains('HTTP 404');
  }

  void _goToPayment(BuildContext context, {required String avatarId}) {
    // ✅ 「戻る」用に現在地を from として URL で渡す（Uri で安全に組み立て）
    final from = Uri(
      path: AppRoutePath.cart,
      queryParameters: {AppQueryKey.avatarId: avatarId},
    ).toString();

    // ✅ URL に avatarId を持たせるため go_router で遷移
    context.goNamed(
      AppRouteName.payment,
      queryParameters: {AppQueryKey.avatarId: avatarId, AppQueryKey.from: from},
    );
  }

  @override
  Widget build(BuildContext context) {
    final avatarId = _avatarId;

    if (avatarId.isEmpty) {
      return const Center(child: Text('avatarId is required'));
    }

    _ensureController();

    void safeSetState(VoidCallback fn) {
      if (!mounted) return;
      setState(fn);
    }

    final ctl = _ctl;
    if (ctl == null) {
      return const Center(child: CircularProgressIndicator());
    }

    final vm = ctl.buildResult(safeSetState);

    return FutureBuilder<CartDTO>(
      future: vm.future,
      builder: (context, snap) {
        final isLoading =
            snap.connectionState == ConnectionState.waiting && !snap.hasData;

        final bool notFoundAsEmpty =
            snap.hasError && _isCartNotFound(snap.error);

        final CartDTO cart =
            snap.data ??
            (notFoundAsEmpty ? _emptyCart(avatarId) : _emptyCart(avatarId));

        return LayoutBuilder(
          builder: (context, constraints) {
            final content = Padding(
              padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  _HeaderCard(
                    avatarId: avatarId,
                    totalQty: cart.totalQty(),
                    expiresAt: cart.expiresAt,
                    onReload: vm.reload,
                    loading: isLoading,
                  ),
                  const SizedBox(height: 12),
                  if (isLoading)
                    const _LoadingCard()
                  else if (snap.hasError && !notFoundAsEmpty)
                    _ErrorCard(
                      errorText: snap.error.toString(),
                      onRetry: vm.reload,
                    )
                  else
                    Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        _ItemsCard(
                          cart: cart,
                          onInc: vm.inc,
                          onDec: vm.dec,
                          onRemove: vm.remove,
                          onClear: vm.clear,
                        ),
                        const SizedBox(height: 12),
                        _PurchaseBar(
                          enabled: cart.totalQty() > 0,
                          onPressed: () =>
                              _goToPayment(context, avatarId: avatarId),
                        ),
                      ],
                    ),
                ],
              ),
            );

            // ✅ bounded height のときだけスクロールを付与（unbounded だと白画面原因になり得る）
            final body = constraints.hasBoundedHeight
                ? SingleChildScrollView(child: content)
                : content;

            return Stack(
              children: [
                body,
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
    required this.onReload,
    required this.loading,
  });

  final String avatarId;
  final int totalQty;
  final DateTime? expiresAt;
  final Future<void> Function() onReload;
  final bool loading;

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
                  Text(
                    loading ? 'items: ...' : 'items: $totalQty',
                    style: t.bodySmall,
                  ),
                  const SizedBox(height: 4),
                  Text(
                    loading ? 'expiresAt: ...' : 'expiresAt: $expText',
                    style: t.bodySmall,
                  ),
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

  /// itemKey を受け取る
  final Future<void> Function(String itemKey) onInc;
  final Future<void> Function(String itemKey, int currentQty) onDec;
  final Future<void> Function(String itemKey) onRemove;

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
                  final itemKey = e.key;
                  final it = e.value; // CartItemDTO

                  final qty = it.qty;

                  return ListTile(
                    contentPadding: const EdgeInsets.symmetric(horizontal: 4),
                    title: Text(
                      it.modelId,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    subtitle: Text(
                      'inventoryId: ${it.inventoryId}\nlistId: ${it.listId}\nqty: $qty',
                      maxLines: 3,
                      overflow: TextOverflow.ellipsis,
                    ),
                    trailing: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        IconButton(
                          tooltip: 'Remove',
                          onPressed: () => onRemove(itemKey),
                          icon: const Icon(Icons.close),
                        ),
                        IconButton(
                          tooltip: '-',
                          onPressed: qty <= 1
                              ? () => onRemove(itemKey)
                              : () => onDec(itemKey, qty),
                          icon: const Icon(Icons.remove_circle_outline),
                        ),
                        Text('$qty', style: t.titleSmall),
                        IconButton(
                          tooltip: '+',
                          onPressed: () => onInc(itemKey),
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

class _PurchaseBar extends StatelessWidget {
  const _PurchaseBar({required this.enabled, required this.onPressed});

  final bool enabled;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      top: false,
      child: SizedBox(
        height: 48,
        child: FilledButton(
          onPressed: enabled ? onPressed : null,
          child: const Text('購入する'),
        ),
      ),
    );
  }
}
