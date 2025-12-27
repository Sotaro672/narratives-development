//frontend\sns\lib\features\home\presentation\components\catalog_product.dart
import 'package:flutter/material.dart';

/// ProductBlueprint の型定義がこのファイルから直接参照できないため、
/// 受け取りは dynamic にしています（CatalogPage 側から vm.productBlueprint をそのまま渡す想定）。
class CatalogProductCard extends StatelessWidget {
  const CatalogProductCard({
    super.key,
    required this.productBlueprintId,
    required this.productBlueprint,
    required this.error,
  });

  final String productBlueprintId;
  final dynamic
  productBlueprint; // ProductBlueprint DTO/entity (from use_catalog)
  final String? error;

  String _s(String? v, {String fallback = '(empty)'}) {
    final t = (v ?? '').trim();
    return t.isNotEmpty ? t : fallback;
  }

  // ✅ productIdTag の type を best-effort で取り出す（Map / class / 既存string すべて対応）
  String _productIdTagType(dynamic pb) {
    if (pb == null) return '';

    // 1) 既存: pb.productIdTagType が string の場合
    try {
      final v = pb.productIdTagType;
      if (v != null) {
        final s = v.toString().trim();
        if (s.isNotEmpty) return s;
      }
    } catch (_) {}

    // 2) pb.productIdTag が Map の場合（jsonDecode由来）
    try {
      final tag = pb.productIdTag;
      if (tag is Map) {
        final t = tag['type'] ?? tag['Type'];
        if (t != null) {
          final s = t.toString().trim();
          if (s.isNotEmpty) return s;
        }
      }
    } catch (_) {}

    // 3) pb.productIdTag が class の場合（tag.type）
    try {
      final tag = pb.productIdTag;
      if (tag != null) {
        final t = tag.type;
        if (t != null) {
          final s = t.toString().trim();
          if (s.isNotEmpty) return s;
        }
      }
    } catch (_) {}

    return '';
  }

  @override
  Widget build(BuildContext context) {
    final pbId = productBlueprintId.trim();
    final pb = productBlueprint;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('商品', style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            if (pb != null) ...[
              _KeyValueRow(label: '商品名', value: _s(pb.productName)),
              const SizedBox(height: 6),

              // ✅ 名前解決結果（SNS側で付与される想定）
              _KeyValueRow(label: '会社名', value: _s(pb.companyName)),
              const SizedBox(height: 6),
              _KeyValueRow(label: 'ブランド名', value: _s(pb.brandName)),
              const SizedBox(height: 6),

              _KeyValueRow(label: 'カテゴリ', value: _s(pb.itemType)),
              const SizedBox(height: 6),
              _KeyValueRow(label: 'フィット', value: _s(pb.fit)),
              const SizedBox(height: 6),
              _KeyValueRow(label: '素材', value: _s(pb.material)),
              const SizedBox(height: 6),
              _KeyValueRow(
                label: '重量',
                value: pb.weight != null ? '${pb.weight}' : '(empty)',
              ),

              const SizedBox(height: 12),
              Text('品質保証', style: Theme.of(context).textTheme.titleSmall),
              const SizedBox(height: 6),
              if ((pb.qualityAssurance ?? const <dynamic>[]).isEmpty)
                Text('(empty)', style: Theme.of(context).textTheme.bodyMedium)
              else
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: (pb.qualityAssurance as List)
                      .map(
                        (s) => Chip(
                          label: Text(s.toString()),
                          visualDensity: VisualDensity.compact,
                        ),
                      )
                      .toList(),
                ),

              const SizedBox(height: 12),
              Text('商品IDタグ', style: Theme.of(context).textTheme.titleSmall),
              const SizedBox(height: 6),
              _KeyValueRow(label: 'タグ種別', value: _s(_productIdTagType(pb))),
            ] else ...[
              _KeyValueRow(
                label: '商品ブループリントID',
                value: pbId.isNotEmpty ? pbId : '(unknown)',
              ),
              if (error != null && error!.trim().isNotEmpty) ...[
                const SizedBox(height: 10),
                Text(
                  '商品エラー: ${error!.trim()}',
                  style: Theme.of(context).textTheme.labelSmall,
                ),
              ] else ...[
                const SizedBox(height: 10),
                Text(
                  '商品が読み込まれていません',
                  style: Theme.of(context).textTheme.labelSmall,
                ),
              ],
            ],
          ],
        ),
      ),
    );
  }
}

class _KeyValueRow extends StatelessWidget {
  const _KeyValueRow({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        SizedBox(
          width: 160,
          child: Text(label, style: Theme.of(context).textTheme.labelMedium),
        ),
        Expanded(child: Text(value)),
      ],
    );
  }
}
