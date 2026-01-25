// frontend/mall/lib/features/wallet/presentation/page/contents.dart
import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';

import '../hook/use_contents.dart';

/// Wallet token detail page (destination from TokenCard tap).
///
/// NOTE:
/// - header側が `?from=` を読んで戻るので、このWidget自身は戻るUIを持たない
/// - 取得/補完ロジックは hook 側に集約（このファイルは見た目＝スタイル中心）
class WalletContentsPage extends HookWidget {
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

    /// ✅ Preview 埋め込み時:
    /// - productName はリンクにしない
    /// - tokenName を押下可能にし、contents.dart（本ページ）へ遷移させる
    this.enableProductLink = true,
    this.enableTokenNameLink = false,
  });

  /// mint address (token identifier) - ✅ required
  /// NOTE: 画面には表示しない（保持のみ）
  final String mintAddress;

  /// resolved from backend
  /// NOTE: 画面には表示しない（保持のみ）
  final String? productId;
  final String? brandId;

  /// resolved names
  final String? brandName;
  final String? productName;

  /// metadata name
  final String? tokenName;

  /// metadata image url（互換: 既存ルートから渡される可能性あり）
  final String? imageUrl;

  /// optional return path (decoded, plain string)
  final String? from;

  /// productName をリンク（previewへ遷移）にするか
  final bool enableProductLink;

  /// tokenName をリンク（contentsへ遷移）にするか
  final bool enableTokenNameLink;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    final vm = useWalletContentsViewModel(
      mintAddress: mintAddress,
      productId: productId,
      brandId: brandId,
      brandName: brandName,
      productName: productName,
      tokenName: tokenName,
      imageUrl: imageUrl,
      from: from,
    );

    final children = <Widget>[];

    if (vm.loading) {
      children.add(
        const Padding(
          padding: EdgeInsets.only(bottom: 10),
          child: Row(
            children: [
              SizedBox(
                width: 18,
                height: 18,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
              SizedBox(width: 10),
              Text('読み込み中…'),
            ],
          ),
        ),
      );
    }

    final errText = vm.error.trim();
    if (errText.isNotEmpty) {
      children.add(
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: Colors.red.shade50,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: Colors.red.shade200),
          ),
          child: Text(errText, style: TextStyle(color: Colors.red.shade800)),
        ),
      );
      children.add(const SizedBox(height: 12));
    }

    // ✅ contents を「URL表示」ではなく「バケット上の実体を取得してプレビュー表示」
    // （URL文字列自体はUIに出さない）
    children.add(_ContentsArea(contentsUrl: vm.contentsUrl));
    children.add(const SizedBox(height: 12));

    children.add(
      _Card(
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // ✅ 推奨: imageUrl 互換ではなく iconUrl を表示
            _smallIcon(context, vm.iconUrl),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start, // ✅ 左寄せ
                children: [
                  if (vm.brandName.isNotEmpty)
                    Text(
                      vm.brandName,
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    )
                  else
                    Text(
                      '（brandName 未取得）',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        color: cs.onSurfaceVariant,
                      ),
                    ),
                  const SizedBox(height: 6),

                  // ✅ tokenName を押下可能（Preview埋め込み時の期待値）
                  if (vm.tokenName.isNotEmpty)
                    (enableTokenNameLink
                        ? TextButton(
                            onPressed: () => vm.openContents(context),
                            style: TextButton.styleFrom(
                              padding: EdgeInsets.zero,
                              minimumSize: const Size(0, 0),
                              tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                              alignment: Alignment.centerLeft,
                            ),
                            child: Text(
                              vm.tokenName,
                              style: Theme.of(context).textTheme.bodyLarge
                                  ?.copyWith(
                                    fontWeight: FontWeight.w700,
                                    decoration: TextDecoration.underline,
                                  ),
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                            ),
                          )
                        : Text(
                            vm.tokenName,
                            style: Theme.of(context).textTheme.bodyLarge
                                ?.copyWith(fontWeight: FontWeight.w700),
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                          ))
                  else
                    Text(
                      '（トークン名 未取得）',
                      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                        color: cs.onSurfaceVariant,
                      ),
                    ),

                  const SizedBox(height: 6),

                  // ✅ Preview埋め込み時は productName ボタンを出さない（期待値）
                  if (vm.productName.isNotEmpty)
                    (enableProductLink
                        ? TextButton(
                            onPressed: vm.productId.isEmpty
                                ? null
                                : () => vm.goPreviewByProductId(
                                    context,
                                    vm.productId,
                                  ),
                            style: TextButton.styleFrom(
                              padding: EdgeInsets.zero,
                              minimumSize: const Size(0, 0),
                              tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                              alignment: Alignment.centerLeft,
                            ),
                            child: Text(
                              vm.productName,
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                              style: Theme.of(context).textTheme.bodyMedium
                                  ?.copyWith(
                                    fontWeight: FontWeight.w700,
                                    decoration: TextDecoration.underline,
                                  ),
                            ),
                          )
                        : Text(
                            vm.productName,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: Theme.of(context).textTheme.bodyMedium
                                ?.copyWith(fontWeight: FontWeight.w700),
                          ))
                  else
                    Text(
                      '（productName 未取得）',
                      style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: cs.onSurfaceVariant,
                      ),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: children,
    );
  }

  Widget _smallIcon(BuildContext context, String url) {
    final cs = Theme.of(context).colorScheme;
    const double size = 56;

    final u = url.trim();
    return ClipRRect(
      borderRadius: BorderRadius.circular(999),
      child: Container(
        width: size,
        height: size,
        color: cs.surface,
        child: u.isEmpty
            ? Icon(
                Icons.image_not_supported_outlined,
                color: cs.onSurfaceVariant,
                size: 22,
              )
            : Image.network(
                u,
                fit: BoxFit.cover,
                errorBuilder: (_, __, ___) => Icon(
                  Icons.broken_image_outlined,
                  color: cs.onSurfaceVariant,
                  size: 22,
                ),
                loadingBuilder: (context, child, p) {
                  if (p == null) return child;
                  return const Center(
                    child: SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                  );
                },
              ),
      ),
    );
  }
}

class _ContentsArea extends StatelessWidget {
  const _ContentsArea({required this.contentsUrl});

  final String contentsUrl;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    final u = contentsUrl.trim();

    return _Card(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Contents',
            style: Theme.of(
              context,
            ).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w800),
          ),
          const SizedBox(height: 8),

          if (u.isEmpty)
            Text(
              '（contents 未取得）',
              style: Theme.of(
                context,
              ).textTheme.bodyMedium?.copyWith(color: cs.onSurfaceVariant),
            )
          else
            ClipRRect(
              borderRadius: BorderRadius.circular(12),
              child: Container(
                width: double.infinity,
                constraints: const BoxConstraints(minHeight: 160),
                color: cs.surface,
                child: Image.network(
                  u,
                  fit: BoxFit.contain,
                  // ✅ bucket上の実体へアクセスして描画（URL文字列はUIに表示しない）
                  loadingBuilder: (context, child, progress) {
                    if (progress == null) return child;
                    return const Center(
                      child: SizedBox(
                        width: 22,
                        height: 22,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      ),
                    );
                  },
                  errorBuilder: (_, __, ___) => _PreviewNotAvailable(),
                ),
              ),
            ),

          if (u.isNotEmpty) ...[
            const SizedBox(height: 8),
            Text(
              '（プレビュー不可の場合は、コンテンツ形式または空ファイルの可能性があります）',
              style: Theme.of(
                context,
              ).textTheme.bodySmall?.copyWith(color: cs.onSurfaceVariant),
            ),
          ],
        ],
      ),
    );
  }
}

class _PreviewNotAvailable extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.insert_drive_file_outlined,
              color: cs.onSurfaceVariant,
              size: 28,
            ),
            const SizedBox(height: 8),
            Text(
              'このコンテンツはプレビューできません',
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: cs.onSurfaceVariant,
                fontWeight: FontWeight.w700,
              ),
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
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
