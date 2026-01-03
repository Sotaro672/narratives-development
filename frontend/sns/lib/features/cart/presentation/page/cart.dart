// frontend/sns/lib/features/cart/presentation/page/cart.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../hook/use_cart.dart';

class CartPage extends StatefulWidget {
  const CartPage({super.key});

  static const String pageName = 'cart';

  @override
  State<CartPage> createState() => _CartPageState();
}

class _CartPageState extends State<CartPage> {
  UseCartController? _ctrl;
  bool _booted = false;

  bool get _dbg => kDebugMode;

  void _log(String msg) {
    if (!_dbg) return;
    // ignore: avoid_print
    print('[CartPage] $msg');
  }

  String _prettyJson(dynamic v) {
    try {
      return const JsonEncoder.withIndent('  ').convert(v);
    } catch (_) {
      return v.toString();
    }
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_booted) return;
    _booted = true;

    final aid = _avatarIdFromRoute();
    if (aid.isEmpty) {
      _log('boot abort: avatarId missing');
      return;
    }

    _ctrl = UseCartController(avatarId: aid, context: context);
    _ctrl!.init();
  }

  @override
  void dispose() {
    _ctrl?.dispose();
    super.dispose();
  }

  String _avatarIdFromRoute() {
    final uri = GoRouterState.of(context).uri;
    return (uri.queryParameters['avatarId'] ?? '').trim();
  }

  String _mask(String s) {
    final t = s.trim();
    if (t.isEmpty) return '';
    if (t.length <= 8) return t;
    return '${t.substring(0, 4)}***${t.substring(t.length - 4)}';
  }

  // ✅ dynamic/nullableでも絶対落ちない
  String _s(dynamic v) => (v ?? '').toString().trim();

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

  Widget _thumb(String? url) {
    final u = (url ?? '').trim();
    if (u.isEmpty || !(u.startsWith('http://') || u.startsWith('https://'))) {
      return _thumbPlaceholder();
    }

    return ClipRRect(
      borderRadius: BorderRadius.circular(8),
      child: SizedBox(
        width: 56,
        height: 56,
        child: Image.network(
          u,
          fit: BoxFit.cover,
          errorBuilder: (_, __, ___) => _thumbPlaceholder(),
          loadingBuilder: (context, child, progress) {
            if (progress == null) return child;
            return Container(
              width: 56,
              height: 56,
              color: Colors.grey.shade200,
              child: const Center(
                child: SizedBox(
                  width: 18,
                  height: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
              ),
            );
          },
        ),
      ),
    );
  }

  Widget _buildItemCard({
    required String itemKey,
    required CartItemDTO it,
    required bool busy,
    required Future<void> Function(String itemKey) onRemove,
  }) {
    final title = _s(it.title);
    final productName = _s(it.productName);
    final size = _s(it.size);
    final color = _s(it.color);
    final listImage = _s(it.listImage);

    final displayTitle = productName.isNotEmpty
        ? productName
        : (title.isNotEmpty ? title : '商品');

    final lines = <String>[];
    if (title.isNotEmpty && productName.isNotEmpty) {
      lines.add(title);
    }

    // ✅ ラベルを日本語に
    lines.add(
      'サイズ: ${size.isEmpty ? '-' : size}  色: ${color.isEmpty ? '-' : color}',
    );
    lines.add('数量: ${it.qty}点');
    if (it.price != null) {
      lines.add('価格: ${it.price}円');
    }
    final subtitle = lines.join('\n');

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
          _thumb(listImage),
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
                Text(
                  subtitle,
                  style: TextStyle(color: Colors.grey.shade700, height: 1.35),
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          IconButton(
            onPressed: busy ? null : () => onRemove(itemKey),
            icon: const Icon(Icons.close),
            tooltip: '削除',
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    if (_ctrl == null) {
      return const Padding(
        padding: EdgeInsets.fromLTRB(16, 10, 16, 16),
        child: Text('avatarId がクエリにありません（?avatarId=...）'),
      );
    }

    final r = _ctrl!.buildResult(setState);

    return FutureBuilder<CartDTO>(
      future: r.future,
      builder: (context, snap) {
        final loading = snap.connectionState == ConnectionState.waiting;

        // ✅ “絶対に落ちない”：失敗/未完了でも current を使って描画可能にする
        final CartDTO cart = snap.data ?? r.current;
        final Object? err = snap.hasError ? snap.error : null;

        final entries = cart.items.entries.toList();

        if (_dbg) {
          _log(
            'render: loading=$loading err=${err != null} avatarId="${_mask(cart.avatarId)}" items=${entries.length}',
          );
          if (entries.isNotEmpty) {
            final e0 = entries.first;
            _log(
              'item0 key="${_mask(e0.key)}" val=${_prettyJson({"inventoryId": e0.value.inventoryId, "listId": e0.value.listId, "modelId": e0.value.modelId, "qty": e0.value.qty, "title": e0.value.title, "productName": e0.value.productName, "listImage": e0.value.listImage, "price": e0.value.price, "size": e0.value.size, "color": e0.value.color})}',
            );
          }
        }

        // ✅ タイトル/文言を日本語化 + Clear を右寄せ
        final children = <Widget>[
          Text(
            '商品数: ${entries.length}',
            style: TextStyle(
              fontSize: 22,
              fontWeight: FontWeight.w700,
              color: Colors.grey.shade900,
            ),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              const Spacer(),
              TextButton.icon(
                onPressed: (loading || r.busy || entries.isEmpty)
                    ? null
                    : r.clear,
                icon: const Icon(Icons.delete_outline, size: 18),
                label: const Text('空にする'),
              ),
            ],
          ),
          const SizedBox(height: 10),
        ];

        if (loading && entries.isEmpty) {
          children.add(
            const Padding(
              padding: EdgeInsets.only(top: 24),
              child: Center(child: CircularProgressIndicator()),
            ),
          );
        } else if (err != null && entries.isEmpty) {
          children.add(
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
          );
        } else if (entries.isEmpty) {
          children.add(
            const Padding(
              padding: EdgeInsets.only(top: 24),
              child: Center(child: Text('カートは空です')),
            ),
          );
        } else {
          for (var i = 0; i < entries.length; i++) {
            final e = entries[i];
            children.add(
              _buildItemCard(
                itemKey: e.key,
                it: e.value,
                busy: r.busy || loading,
                onRemove: r.remove,
              ),
            );
            if (i != entries.length - 1) {
              children.add(const SizedBox(height: 10));
            }
          }

          if (err != null) {
            children.add(const SizedBox(height: 14));
            children.add(
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: Colors.orange.shade50,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: Colors.orange.shade200),
                ),
                child: Text(
                  '最新取得に失敗（表示はキャッシュ/現状のまま）: ${err.toString()}',
                  style: TextStyle(color: Colors.orange.shade900),
                ),
              ),
            );
          }
        }

        return SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(16, 10, 16, 16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: children,
          ),
        );
      },
    );
  }
}
