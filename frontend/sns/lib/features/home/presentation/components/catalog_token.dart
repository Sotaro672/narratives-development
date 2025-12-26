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

  void _log(String msg) {
    // ignore: avoid_print
    print('[TokenCard] $msg');
  }

  String _s(String? v, {String fallback = '(empty)'}) {
    final t = (v ?? '').trim();
    return t.isNotEmpty ? t : fallback;
  }

  @override
  Widget build(BuildContext context) {
    final tbId = tokenBlueprintId.trim();
    final p = patch;

    // ✅ このカードが「実際に受け取った patch」をログで確認
    _log(
      'build tbId="${tbId.isNotEmpty ? tbId : '(empty)'}" '
      'patch?=${p != null} '
      'name="${p?.name ?? ''}" symbol="${p?.symbol ?? ''}" brandId="${p?.brandId ?? ''}" '
      'minted=${p?.minted} '
      'hasIcon=${(iconUrlEncoded ?? '').trim().isNotEmpty} '
      'err="${(error ?? '').trim()}"',
    );

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Token', style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            _KeyValueRow(
              label: 'tokenBlueprintId',
              value: tbId.isNotEmpty ? tbId : '(unknown)',
            ),
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
                        label: 'token icon failed',
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
              _KeyValueRow(label: 'name', value: _s(p.name)),
              const SizedBox(height: 6),
              _KeyValueRow(label: 'symbol', value: _s(p.symbol)),
              const SizedBox(height: 6),
              _KeyValueRow(label: 'brandId', value: _s(p.brandId)),
              const SizedBox(height: 6),
              _KeyValueRow(label: 'brandName', value: _s(p.brandName)),
              const SizedBox(height: 6),
              _KeyValueRow(
                label: 'minted',
                value: p.minted == null
                    ? '(unknown)'
                    : (p.minted! ? 'true' : 'false'),
              ),
              const SizedBox(height: 10),
              if ((p.description ?? '').trim().isNotEmpty)
                Text(
                  p.description!.trim(),
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
            ] else ...[
              if (error != null && error!.trim().isNotEmpty)
                Text(
                  'token error: ${error!.trim()}',
                  style: Theme.of(context).textTheme.labelSmall,
                )
              else
                Text(
                  'token is not loaded',
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
