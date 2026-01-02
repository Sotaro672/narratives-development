// frontend/sns/lib/features/cart/presentation/page/cart.dart
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/routing/routes.dart';
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

  CartQueryDTO _emptyCartQuery() {
    return CartQueryDTO(raw: const {}, cart: null, rows: const []);
  }

  bool _isCartNotFound(Object? err) {
    if (err is CartHttpException) {
      return err.statusCode == 404;
    }
    final s = err?.toString() ?? '';
    return s.contains('statusCode=404') || s.contains('HTTP 404');
  }

  bool _isQueryNotFound(Object? err) {
    if (err is CartHttpException) {
      return err.statusCode == 404;
    }
    final s = err?.toString() ?? '';
    return s.contains('statusCode=404') || s.contains('HTTP 404');
  }

  DateTime? _tryParseTime(dynamic v) {
    if (v == null) return null;

    if (v is DateTime) return v;

    if (v is String) {
      final s = v.trim();
      if (s.isEmpty) return null;
      return DateTime.tryParse(s);
    }

    if (v is Map) {
      final sec = v['seconds'];
      final nanos = v['nanos'] ?? 0;

      final s = (sec is int) ? sec : int.tryParse(sec?.toString() ?? '');
      final n = (nanos is int)
          ? nanos
          : (int.tryParse(nanos?.toString() ?? '0') ?? 0);

      if (s == null) return null;

      return DateTime.fromMillisecondsSinceEpoch(
        s * 1000 + (n ~/ 1000000),
        isUtc: true,
      );
    }

    final s = v.toString().trim();
    if (s.isEmpty) return null;
    return DateTime.tryParse(s);
  }

  /// ✅ CartQueryDTO に expiresAt が無い前提で、raw/cart から best-effort で引く
  DateTime? _expiresAtFromQuery(CartQueryDTO q) {
    try {
      final fromCart = q.cart?.expiresAt;
      if (fromCart != null) return fromCart;
    } catch (_) {
      // ignore
    }

    try {
      final v = q.raw['expiresAt'];
      final dt = _tryParseTime(v);
      if (dt != null) return dt;
    } catch (_) {
      // ignore
    }

    return null;
  }

  /// ✅ CartQueryDTO に totalQty() が無い前提で rows の qty を合算
  int _totalQtyFromQuery(CartQueryDTO q) {
    var sum = 0;
    for (final r in q.rows) {
      final n = r.qty;
      if (n > 0) sum += n;
    }
    return sum;
  }

  // ------------------------------------------------------------
  // ✅ preview navigation (from is base64url)
  // ------------------------------------------------------------

  String _encodeFrom(String raw) {
    final s = raw.trim();
    if (s.isEmpty) return '';
    return base64UrlEncode(utf8.encode(s));
  }

  String _currentCartUri(String avatarId) {
    return Uri(
      path: AppRoutePath.cart,
      queryParameters: {AppQueryKey.avatarId: avatarId},
    ).toString();
  }

  void _goPreview(BuildContext context, String avatarId) {
    final fromRaw = (widget.from ?? '').trim().isNotEmpty
        ? widget.from!.trim()
        : _currentCartUri(avatarId);

    final qp = <String, String>{AppQueryKey.avatarId: avatarId.trim()};

    final enc = _encodeFrom(fromRaw);
    if (enc.isNotEmpty) {
      qp[AppQueryKey.from] = enc;
    }

    context.goNamed(AppRouteName.preview, queryParameters: qp);
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

    // ✅ cart_query を「表示優先」にするため outer を CartQueryDTO にする
    return FutureBuilder<CartQueryDTO>(
      future: vm.cartQueryFuture,
      builder: (context, cqSnap) {
        final cqLoading =
            cqSnap.connectionState == ConnectionState.waiting &&
            !cqSnap.hasData;

        final bool cqNotFoundAsEmpty =
            cqSnap.hasError && _isQueryNotFound(cqSnap.error);

        final cartQ =
            cqSnap.data ??
            (cqNotFoundAsEmpty ? _emptyCartQuery() : _emptyCartQuery());

        final totalQtyFromQuery = _totalQtyFromQuery(cartQ);
        final expiresAtFromQuery = _expiresAtFromQuery(cartQ);

        return LayoutBuilder(
          builder: (context, constraints) {
            final content = Padding(
              padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  _HeaderCard(
                    avatarId: avatarId,
                    totalQty: totalQtyFromQuery,
                    expiresAt: expiresAtFromQuery,
                    onReload: () async {
                      // ✅ まとめて更新（legacy も含めて同期）
                      await vm.reload();
                      await vm.reloadCartQuery();
                      await vm.reloadPreview();
                    },
                    onPreview: totalQtyFromQuery <= 0
                        ? null
                        : () => _goPreview(context, avatarId),
                    loading: cqLoading,
                  ),
                  const SizedBox(height: 12),

                  if (cqLoading)
                    const _LoadingCard()
                  else if (cqSnap.hasError && !cqNotFoundAsEmpty)
                    _ErrorCard(
                      errorText: cqSnap.error.toString(),
                      onRetry: () async {
                        await vm.reloadCartQuery();
                      },
                    )
                  else
                    Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        _CartQueryItemsCard(
                          cartQuery: cartQ,
                          onInc: vm.inc,
                          onDec: vm.dec,
                          onRemove: vm.remove,
                          onClear: vm.clear,
                          onOpenPreview: totalQtyFromQuery <= 0
                              ? null
                              : () => _goPreview(context, avatarId),
                        ),
                        const SizedBox(height: 12),

                        // legacy cart は “デバッグ用途” として残したい場合だけ表示
                        FutureBuilder<CartDTO>(
                          future: vm.future,
                          builder: (context, snap) {
                            final isLoading =
                                snap.connectionState ==
                                    ConnectionState.waiting &&
                                !snap.hasData;

                            final bool notFoundAsEmpty =
                                snap.hasError && _isCartNotFound(snap.error);

                            final CartDTO cart =
                                snap.data ??
                                (notFoundAsEmpty
                                    ? _emptyCart(avatarId)
                                    : _emptyCart(avatarId));

                            if (isLoading) return const SizedBox.shrink();
                            if (snap.hasError && !notFoundAsEmpty) {
                              return const SizedBox.shrink();
                            }
                            return _LegacyCartDebugCard(cart: cart);
                          },
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
    this.onPreview,
  });

  final String avatarId;
  final int totalQty;
  final DateTime? expiresAt;
  final Future<void> Function() onReload;
  final bool loading;

  /// null の場合は無効
  final VoidCallback? onPreview;

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
            if (onPreview != null)
              IconButton(
                tooltip: 'Preview',
                onPressed: onPreview,
                icon: const Icon(Icons.receipt_long_outlined),
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

/// ✅ cart_query の rows を表示する Card
/// NOTE: CartQueryRowDTO のフィールドは「確実に存在する最小セット」だけ使う
class _CartQueryItemsCard extends StatelessWidget {
  const _CartQueryItemsCard({
    required this.cartQuery,
    required this.onInc,
    required this.onDec,
    required this.onRemove,
    required this.onClear,
    this.onOpenPreview,
  });

  final CartQueryDTO cartQuery;

  /// itemKey を受け取る
  final Future<void> Function(String itemKey) onInc;
  final Future<void> Function(String itemKey, int currentQty) onDec;
  final Future<void> Function(String itemKey) onRemove;

  final Future<void> Function() onClear;

  /// 行タップで preview へ（null の場合は無効）
  final VoidCallback? onOpenPreview;

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;

    final rows = cartQuery.rows.toList()
      ..sort((a, b) => a.itemKey.compareTo(b.itemKey));

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
                  onPressed: rows.isEmpty ? null : () => onClear(),
                  icon: const Icon(Icons.delete_outline),
                  label: const Text('Clear'),
                ),
              ],
            ),
            const SizedBox(height: 8),
            if (rows.isEmpty)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 14),
                child: Text('カートは空です', style: t.bodyMedium),
              )
            else
              ListView.separated(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                itemCount: rows.length,
                separatorBuilder: (_, __) => Divider(
                  height: 1,
                  thickness: 1,
                  color: Theme.of(context).dividerColor.withValues(alpha: 0.4),
                ),
                itemBuilder: (context, i) {
                  final r = rows[i];
                  final itemKey = r.itemKey;
                  final qty = r.qty;

                  // ✅ title は最低限 modelId を表示（title 等の拡張フィールドには依存しない）
                  final title = (r.modelId).trim().isNotEmpty
                      ? r.modelId
                      : itemKey;

                  return ListTile(
                    contentPadding: const EdgeInsets.symmetric(horizontal: 4),

                    // ✅ 「カート（行）を押下」→ preview へ
                    onTap: onOpenPreview,

                    title: Text(
                      title,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    subtitle: Text(
                      'inventoryId: ${r.inventoryId}\nlistId: ${r.listId}\nmodelId: ${r.modelId}\nqty: $qty',
                      maxLines: 6,
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

/// legacy cart の簡易デバッグ表示（不要なら消してOK）
class _LegacyCartDebugCard extends StatelessWidget {
  const _LegacyCartDebugCard({required this.cart});

  final CartDTO cart;

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;
    final entries = cart.items.entries.toList()
      ..sort((a, b) => a.key.compareTo(b.key));

    if (entries.isEmpty) return const SizedBox.shrink();

    return Card(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 10, 12, 10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Legacy (debug)', style: t.titleSmall),
            const SizedBox(height: 8),
            for (final e in entries.take(3))
              Padding(
                padding: const EdgeInsets.only(bottom: 6),
                child: Text(
                  '${e.key}: modelId=${e.value.modelId} qty=${e.value.qty}',
                  style: t.bodySmall,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            if (entries.length > 3)
              Text('... (${entries.length} items)', style: t.bodySmall),
          ],
        ),
      ),
    );
  }
}
