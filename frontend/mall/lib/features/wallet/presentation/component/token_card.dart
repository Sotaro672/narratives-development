// frontend/mall/lib/features/wallet/presentation/component/token_card.dart
import 'package:flutter/material.dart';

import '../../infrastructure/token_resolve_dto.dart';

class TokenCard extends StatelessWidget {
  const TokenCard({super.key, required this.mintAddress, this.resolved});

  /// On-chain mint address (base58)
  final String mintAddress;

  /// Resolved view from Firestore (tokens collection) by mintAddress.
  /// If null, the card shows mintAddress only.
  final TokenResolveDTO? resolved;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    final mint = mintAddress.trim();
    final productId = resolved?.productId.trim() ?? '';
    final brandId = resolved?.brandId.trim() ?? '';
    final metadataUri = resolved?.metadataUri.trim() ?? '';

    Widget kv({required String k, required String v, bool strong = false}) {
      if (v.trim().isEmpty) return const SizedBox.shrink();
      return Padding(
        padding: const EdgeInsets.only(top: 8),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              k,
              style: Theme.of(context).textTheme.labelMedium?.copyWith(
                color: cs.onSurfaceVariant,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 6),
            Text(
              v,
              style:
                  (strong
                          ? Theme.of(context).textTheme.bodyMedium
                          : Theme.of(context).textTheme.bodySmall)
                      ?.copyWith(
                        fontWeight: strong ? FontWeight.w700 : FontWeight.w500,
                        color: strong ? null : cs.onSurfaceVariant,
                      ),
            ),
          ],
        ),
      );
    }

    return Card(
      elevation: 0,
      color: cs.surfaceContainerHighest,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(Icons.local_offer_outlined, color: cs.onSurfaceVariant),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  kv(k: 'mintAddress', v: mint, strong: true),
                  kv(k: 'productId', v: productId),
                  kv(k: 'brandId', v: brandId),
                  kv(k: 'metadataUri', v: metadataUri),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
