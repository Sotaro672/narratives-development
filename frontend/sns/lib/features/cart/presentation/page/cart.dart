import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../infrastructure/cart_repository_http.dart';

class CartPage extends StatefulWidget {
  const CartPage({super.key});

  static const String pageName = 'cart';

  @override
  State<CartPage> createState() => _CartPageState();
}

class _CartPageState extends State<CartPage> {
  late final CartRepositoryHttp _repo;
  late Future<CartQueryDTO> _future;

  bool _booted = false;

  @override
  void initState() {
    super.initState();
    _repo = CartRepositoryHttp();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_booted) return;
    _booted = true;

    // ✅ Hash URL (#/cart?... ) の query は Uri.base では拾えないため、
    //   GoRouterState から読む前提でここで初期化する
    _future = _fetch();
  }

  @override
  void dispose() {
    _repo.dispose();
    super.dispose();
  }

  // ----------------------------
  // routing helpers (go_router)
  // ----------------------------

  String _avatarIdFromRoute() {
    final qp = GoRouterState.of(context).uri.queryParameters;
    return (qp['avatarId'] ?? '').trim();
  }

  // from は現状この page では使っていないが、必要なら同様に取れる
  // String _fromFromRoute() {
  //   final qp = GoRouterState.of(context).uri.queryParameters;
  //   return (qp['from'] ?? '').trim();
  // }

  Future<CartQueryDTO> _fetch() async {
    final aid = _avatarIdFromRoute();
    if (aid.isEmpty) {
      throw StateError('avatarId is missing in query (?avatarId=...)');
    }
    return _repo.fetchCartQuery(avatarId: aid);
  }

  Future<void> _reload() async {
    setState(() {
      _future = _fetch();
    });
  }

  // ----------------------------
  // actions (rows-based)
  // ----------------------------

  Future<void> _remove(CartQueryRowDTO row) async {
    final aid = _avatarIdFromRoute();
    if (aid.isEmpty) return;

    final invId = row.inventoryId.trim();
    final listId = row.listId.trim();
    final modelId = row.modelId.trim();

    if (invId.isEmpty || listId.isEmpty || modelId.isEmpty) return;

    await _repo.removeItem(
      avatarId: aid,
      inventoryId: invId,
      listId: listId,
      modelId: modelId,
    );

    await _reload();
  }

  Future<void> _clear() async {
    final aid = _avatarIdFromRoute();
    if (aid.isEmpty) return;

    await _repo.clearCart(avatarId: aid);
    await _reload();
  }

  // ----------------------------
  // ui helpers
  // ----------------------------

  String _mask(String s) {
    final t = s.trim();
    if (t.length <= 6) return t;
    return '${t.substring(0, 3)}***${t.substring(t.length - 3)}';
  }

  String _fmtPrice(int? price) {
    if (price == null) return '-';
    return '¥$price';
  }

  Widget _buildThumb(String? listImage) {
    final v = (listImage ?? '').trim();

    if (v.startsWith('http://') || v.startsWith('https://')) {
      return ClipRRect(
        borderRadius: BorderRadius.circular(8),
        child: Image.network(
          v,
          width: 56,
          height: 56,
          fit: BoxFit.cover,
          errorBuilder: (_, __, ___) => _thumbPlaceholder(),
        ),
      );
    }

    return _thumbPlaceholder();
  }

  Widget _thumbPlaceholder() {
    return Container(
      width: 56,
      height: 56,
      decoration: BoxDecoration(
        color: Colors.grey.shade200,
        borderRadius: BorderRadius.circular(8),
      ),
      child: const Icon(Icons.image_not_supported_outlined, size: 22),
    );
  }

  // ----------------------------
  // build
  // ----------------------------

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<CartQueryDTO>(
      future: _future,
      builder: (context, snap) {
        final loading = snap.connectionState == ConnectionState.waiting;
        final err = snap.hasError ? snap.error : null;

        final dto = snap.data;
        final rows = dto?.rows ?? const <CartQueryRowDTO>[];

        final avatarId = _avatarIdFromRoute();
        final expiresAt = dto?.cart?.expiresAt;

        return Padding(
          padding: const EdgeInsets.fromLTRB(16, 10, 16, 16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Cart',
                style: TextStyle(
                  fontSize: 22,
                  fontWeight: FontWeight.w700,
                  color: Colors.grey.shade900,
                ),
              ),
              const SizedBox(height: 8),
              Text('avatarId: ${_mask(avatarId)}'),
              Text('items: ${rows.length}'),
              if (expiresAt != null)
                Text('expiresAt: ${expiresAt.toIso8601String()}'),
              const SizedBox(height: 18),

              Row(
                children: [
                  const Text(
                    'Items',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
                  ),
                  const Spacer(),
                  TextButton.icon(
                    onPressed: (loading || rows.isEmpty) ? null : _clear,
                    icon: const Icon(Icons.delete_outline, size: 18),
                    label: const Text('Clear'),
                  ),
                ],
              ),

              const SizedBox(height: 10),

              if (loading) ...[
                const Padding(
                  padding: EdgeInsets.only(top: 24),
                  child: Center(child: CircularProgressIndicator()),
                ),
              ] else if (err != null) ...[
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.red.shade50,
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: Colors.red.shade200),
                  ),
                  child: Text(
                    err.toString(),
                    style: TextStyle(color: Colors.red.shade800),
                  ),
                ),
              ] else if (rows.isEmpty) ...[
                const Padding(
                  padding: EdgeInsets.only(top: 24),
                  child: Center(child: Text('No items in cart.')),
                ),
              ] else ...[
                Expanded(
                  child: ListView.separated(
                    itemCount: rows.length,
                    separatorBuilder: (_, __) => const SizedBox(height: 10),
                    itemBuilder: (context, i) {
                      final r = rows[i];

                      final title = (r.title ?? '').trim();
                      final productName = (r.productName ?? '').trim();
                      final size = (r.size ?? '').trim();
                      final color = (r.color ?? '').trim();

                      final displayTitle = title.isNotEmpty
                          ? title
                          : (productName.isNotEmpty ? productName : 'Item');

                      final qty = r.qty;

                      return Container(
                        padding: const EdgeInsets.all(12),
                        decoration: BoxDecoration(
                          color: Colors.white,
                          borderRadius: BorderRadius.circular(14),
                          border: Border.all(color: Colors.grey.shade200),
                          boxShadow: [
                            BoxShadow(
                              blurRadius: 10,
                              offset: const Offset(0, 4),
                              color: Colors.black.withValues(alpha: 0.04),
                            ),
                          ],
                        ),
                        child: Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            _buildThumb(r.listImage),
                            const SizedBox(width: 12),
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(
                                    displayTitle,
                                    style: const TextStyle(
                                      fontSize: 15,
                                      fontWeight: FontWeight.w600,
                                    ),
                                  ),
                                  const SizedBox(height: 6),
                                  Text('price: ${_fmtPrice(r.price)}'),
                                  const SizedBox(height: 4),
                                  Text(
                                    'size: ${size.isEmpty ? '-' : size} / color: ${color.isEmpty ? '-' : color}',
                                    style: TextStyle(
                                      color: Colors.grey.shade700,
                                    ),
                                  ),
                                  const SizedBox(height: 6),
                                  Text(
                                    'qty: $qty',
                                    style: TextStyle(
                                      color: Colors.grey.shade700,
                                    ),
                                  ),
                                ],
                              ),
                            ),
                            const SizedBox(width: 8),
                            IconButton(
                              onPressed: () => _remove(r),
                              icon: const Icon(Icons.close),
                              tooltip: 'Remove',
                            ),
                          ],
                        ),
                      );
                    },
                  ),
                ),
              ],
            ],
          ),
        );
      },
    );
  }
}
