//frontend\sns\lib\features\home\presentation\components\catalog_token.dart
import 'package:flutter/material.dart';

import '../../../tokenBlueprint/infrastructure/token_blueprint_repository_http.dart'
    show TokenBlueprintPatch;

class CatalogTokenCard extends StatelessWidget {
  const CatalogTokenCard({
    super.key,
    required this.tokenBlueprintId,
    required this.patch,
    required this.error,
    required this.iconUrlEncoded,
  });

  final String tokenBlueprintId;
  final TokenBlueprintPatch? patch;
  final String? error;
  final String? iconUrlEncoded;

  String _s(String? v, {String fallback = '(未設定)'}) {
    final t = (v ?? '').trim();
    return t.isNotEmpty ? t : fallback;
  }

  @override
  Widget build(BuildContext context) {
    final p = patch;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('トークン', style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 10),
            if (p != null) ...[
              if ((iconUrlEncoded ?? '').trim().isNotEmpty) ...[
                AspectRatio(
                  aspectRatio: 1,
                  child: Image.network(
                    iconUrlEncoded!,
                    fit: BoxFit.cover,
                    errorBuilder: (context, err, st) {
                      return _ImageFallback(
                        label: 'トークン画像の読み込みに失敗しました',
                        detail: err.toString(),
                      );
                    },
                    loadingBuilder: (context, child, progress) {
                      if (progress == null) return child;
                      return const Center(child: CircularProgressIndicator());
                    },
                  ),
                ),
                const SizedBox(height: 10),
              ],
              _KeyValueRow(label: 'トークン名', value: _s(p.name)),
              const SizedBox(height: 6),
              _KeyValueRow(label: 'シンボル', value: _s(p.symbol)),
              const SizedBox(height: 6),

              // ✅ brandId / companyId / minted / tokenBlueprintId は表示しない
              _KeyValueRow(label: 'ブランド名', value: _s(p.brandName)),
              const SizedBox(height: 6),
              _KeyValueRow(label: '会社名', value: _s(p.companyName)),
              const SizedBox(height: 10),

              if ((p.description ?? '').trim().isNotEmpty)
                Text(
                  p.description!.trim(),
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
            ] else ...[
              if (error != null && error!.trim().isNotEmpty)
                Text(
                  'トークン取得エラー: ${error!.trim()}',
                  style: Theme.of(context).textTheme.labelSmall,
                )
              else
                Text(
                  'トークン情報が未取得です',
                  style: Theme.of(context).textTheme.labelSmall,
                ),
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

class _ImageFallback extends StatelessWidget {
  const _ImageFallback({required this.label, this.detail});

  final String label;
  final String? detail;

  @override
  Widget build(BuildContext context) {
    return Container(
      color: Theme.of(context).colorScheme.surfaceContainerHighest,
      padding: const EdgeInsets.all(12),
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.image_not_supported_outlined, size: 36),
            const SizedBox(height: 8),
            Text(label),
            if (detail != null) ...[
              const SizedBox(height: 6),
              Text(
                detail!,
                textAlign: TextAlign.center,
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
                style: Theme.of(context).textTheme.labelSmall,
              ),
            ],
          ],
        ),
      ),
    );
  }
}
