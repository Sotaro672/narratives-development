//frontend\mall\lib\features\wallet\presentation\page\contents.dart
import 'package:flutter/material.dart';

/// Wallet token detail page (destination from TokenCard tap).
class WalletContentsPage extends StatelessWidget {
  const WalletContentsPage({
    super.key,
    required this.mintAddress,
    this.productId,
    this.brandId,
    this.brandName,
    this.productName,
    this.tokenName,
    this.imageUrl,
    this.from,
  });

  /// mint address (token identifier) - ✅ required
  final String mintAddress;

  /// resolved from backend
  final String? productId;
  final String? brandId;

  /// resolved names
  final String? brandName;
  final String? productName;

  /// metadata name
  final String? tokenName;

  /// metadata image url
  final String? imageUrl;

  /// optional return path (decoded, plain string)
  ///
  /// NOTE:
  /// - header側が `?from=` を読んで戻るので、このWidget自身は戻るUIを持たない
  /// - ここに値を持っていてもOKだが、現状は display/debug 用に残すだけ
  final String? from;

  String _s(String? v) => (v ?? '').trim();

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    final mint = mintAddress.trim();
    final pid = _s(productId);
    final bid = _s(brandId);
    final bname = _s(brandName);
    final pname = _s(productName);
    final tname = _s(tokenName);
    final img = _s(imageUrl);

    // ✅ AppShell の main 領域に載せる “中身だけ”
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _Card(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (bname.isNotEmpty)
                Text(
                  bname,
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                )
              else
                Text(
                  '（brandName 未取得）',
                  style: Theme.of(
                    context,
                  ).textTheme.titleMedium?.copyWith(color: cs.onSurfaceVariant),
                ),
              const SizedBox(height: 10),

              if (img.isNotEmpty)
                ClipRRect(
                  borderRadius: BorderRadius.circular(14),
                  child: AspectRatio(
                    aspectRatio: 1,
                    child: Image.network(
                      img,
                      fit: BoxFit.cover,
                      loadingBuilder: (context, child, p) {
                        if (p == null) return child;
                        return Container(
                          color: cs.surfaceContainerHighest,
                          alignment: Alignment.center,
                          child: const SizedBox(
                            width: 22,
                            height: 22,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          ),
                        );
                      },
                      errorBuilder: (context, error, st) {
                        return Container(
                          color: cs.surfaceContainerHighest,
                          alignment: Alignment.center,
                          padding: const EdgeInsets.all(12),
                          child: Text(
                            '画像を読み込めませんでした。',
                            style: Theme.of(context).textTheme.bodySmall
                                ?.copyWith(
                                  color: cs.onSurfaceVariant,
                                  fontWeight: FontWeight.w600,
                                ),
                          ),
                        );
                      },
                    ),
                  ),
                )
              else
                Container(
                  height: 220,
                  decoration: BoxDecoration(
                    color: cs.surfaceContainerHighest,
                    borderRadius: BorderRadius.circular(14),
                  ),
                  alignment: Alignment.center,
                  child: Text(
                    '（画像なし）',
                    style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                      color: cs.onSurfaceVariant,
                    ),
                  ),
                ),

              const SizedBox(height: 10),

              if (tname.isNotEmpty)
                Text(
                  tname,
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  textAlign: TextAlign.center,
                )
              else
                Text(
                  '（トークン名 未取得）',
                  style: Theme.of(
                    context,
                  ).textTheme.bodyMedium?.copyWith(color: cs.onSurfaceVariant),
                  textAlign: TextAlign.center,
                ),

              const SizedBox(height: 8),

              if (pname.isNotEmpty)
                Text(
                  pname,
                  style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: cs.onSurfaceVariant,
                    fontWeight: FontWeight.w600,
                  ),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  textAlign: TextAlign.center,
                )
              else
                Text(
                  '（productName 未取得）',
                  style: Theme.of(
                    context,
                  ).textTheme.bodySmall?.copyWith(color: cs.onSurfaceVariant),
                  textAlign: TextAlign.center,
                ),
            ],
          ),
        ),

        const SizedBox(height: 12),

        _Card(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _row(
                context,
                label: 'productId',
                value: pid.isEmpty ? '（未取得）' : pid,
              ),
              const Divider(height: 18),
              _row(
                context,
                label: 'brandId',
                value: bid.isEmpty ? '（未取得）' : bid,
              ),
              const Divider(height: 18),
              _row(
                context,
                label: 'mintAddress',
                value: mint.isEmpty ? '（未取得）' : mint,
                mono: true,
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _row(
    BuildContext context, {
    required String label,
    required String value,
    bool mono = false,
  }) {
    final cs = Theme.of(context).colorScheme;
    final v = value.trim();

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 110,
          child: Text(
            label,
            style: Theme.of(context).textTheme.labelMedium?.copyWith(
              color: cs.onSurfaceVariant,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: Text(
            v,
            style:
                (mono
                        ? Theme.of(context).textTheme.bodySmall
                        : Theme.of(context).textTheme.bodyMedium)
                    ?.copyWith(
                      color: cs.onSurface,
                      fontWeight: FontWeight.w600,
                      fontFamily: mono ? 'monospace' : null,
                    ),
          ),
        ),
      ],
    );
  }
}

class _Card extends StatelessWidget {
  const _Card({required this.child});
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return Card(
      elevation: 0,
      color: cs.surfaceContainerHighest,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
      child: Padding(padding: const EdgeInsets.all(14), child: child),
    );
  }
}
