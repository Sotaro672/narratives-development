// frontend/mall/lib/features/wallet/presentation/component/token_card.dart
import 'package:flutter/material.dart';

import '../../infrastructure/token_metadata_dto.dart';
import '../../infrastructure/token_resolve_dto.dart';

class TokenCard extends StatelessWidget {
  const TokenCard({
    super.key,
    required this.mintAddress,
    this.resolved,
    this.metadata,
    this.isLoading = false,
  });

  final String mintAddress;
  final TokenResolveDTO? resolved;
  final TokenMetadataDTO? metadata;
  final bool isLoading;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    final resolvedOk = resolved != null;
    final metadataOk = metadata != null;

    final failed = !isLoading && (!resolvedOk || (resolvedOk && !metadataOk));

    // ✅ ID ではなく「name」を表示する（label は出さない）
    final productName = (resolved?.productName ?? '').trim();
    final brandName = (resolved?.brandName ?? '').trim();

    return Card(
      elevation: 0,
      color: cs.surfaceContainerHighest,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (isLoading) ...[
              const LinearProgressIndicator(),
            ] else if (failed) ...[
              Text(
                'データが取得できませんでした。',
                style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                  color: cs.error,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ] else ...[
              // -------------------------
              // resolve 側（productName / brandName）
              //  - value のみ表示（label 非表示）
              // -------------------------
              _valueOnly(
                context,
                value: productName.isEmpty ? '（空）' : productName,
              ),
              const SizedBox(height: 6),
              _valueOnly(context, value: brandName.isEmpty ? '（空）' : brandName),

              // metadataUri は非表示（要求により）
              // -------------------------
              // metadata 側（metadataUri の中身）
              // -------------------------
              if (metadata != null) ...[
                const SizedBox(height: 10),

                // name は label なしで value だけ
                if (metadata!.name.trim().isNotEmpty) ...[
                  _valueOnly(context, value: metadata!.name),
                  const SizedBox(height: 6),
                ],

                // ✅ image を画像として表示
                if (metadata!.image.trim().isNotEmpty) ...[
                  const SizedBox(height: 6),
                  _imageBox(context, metadata!.image),
                  const SizedBox(height: 6),
                ],
              ],
            ],
          ],
        ),
      ),
    );
  }

  Widget _imageBox(BuildContext context, String url) {
    final cs = Theme.of(context).colorScheme;
    final u = url.trim();

    return ClipRRect(
      borderRadius: BorderRadius.circular(10),
      child: AspectRatio(
        aspectRatio: 1, // 正方形（必要なら 16/9 等に変更）
        child: Image.network(
          u,
          fit: BoxFit.cover,
          // 読み込み中の表示
          loadingBuilder: (context, child, loadingProgress) {
            if (loadingProgress == null) return child;
            return Container(
              color: cs.surface,
              alignment: Alignment.center,
              child: const SizedBox(
                width: 20,
                height: 20,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            );
          },
          // 失敗時の表示
          errorBuilder: (context, error, stackTrace) {
            return Container(
              color: cs.surface,
              alignment: Alignment.center,
              padding: const EdgeInsets.all(12),
              child: Text(
                '画像を読み込めませんでした。',
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: cs.onSurfaceVariant,
                  fontWeight: FontWeight.w600,
                ),
              ),
            );
          },
        ),
      ),
    );
  }

  // ✅ label を出さず value だけ表示する表示部品
  Widget _valueOnly(BuildContext context, {required String value}) {
    final cs = Theme.of(context).colorScheme;
    final v = value.trim();

    return Text(
      v.isEmpty ? '（空）' : v,
      style: Theme.of(context).textTheme.bodySmall?.copyWith(
        color: cs.onSurface,
        fontWeight: FontWeight.w600,
      ),
    );
  }
}
