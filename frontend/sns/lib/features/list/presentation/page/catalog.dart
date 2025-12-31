// frontend/sns/lib/features/home/presentation/page/catalog.dart
import 'package:flutter/material.dart';

import 'package:sns/features/list/presentation/components/catalog_inventory.dart';
import '../../infrastructure/list_repository_http.dart';
import '../components/catalog_product.dart';
import '../components/catalog_token.dart';
import '../hook/use_catalog.dart';

class CatalogPage extends StatefulWidget {
  const CatalogPage({super.key, required this.listId, this.initialItem});

  final String listId;
  final SnsListItem? initialItem;

  @override
  State<CatalogPage> createState() => _CatalogPageState();
}

class _CatalogPageState extends State<CatalogPage> {
  late final UseCatalog _uc;
  late Future<CatalogState> _future;

  @override
  void initState() {
    super.initState();
    _uc = UseCatalog();
    _future = _uc.load(listId: widget.listId);
  }

  @override
  void dispose() {
    _uc.dispose();
    super.dispose();
  }

  Future<void> _reload() async {
    setState(() {
      _future = _uc.load(listId: widget.listId);
    });
  }

  @override
  Widget build(BuildContext context) {
    final initial = widget.initialItem;

    // ✅ AppShell が Scaffold / Scroll を持つ前提なので、ここは「中身だけ」
    return FutureBuilder<CatalogState>(
      future: _future,
      builder: (context, snap) {
        final vm = snap.data;
        final list = vm?.list ?? initial;

        if (snap.connectionState == ConnectionState.waiting && list == null) {
          return const Center(child: CircularProgressIndicator());
        }
        if (snap.hasError && list == null) {
          return _ErrorView(error: snap.error, onRetry: _reload);
        }
        if (list == null) {
          return const Center(child: Text('No data'));
        }

        final priceText = vm?.priceText ?? '';
        final hasImage = vm?.hasImage ?? list.image.trim().isNotEmpty;

        // vm に imageUrlEncoded が無い場合でも list.image を使って出せるようにする
        final imageUrlEncoded =
            (vm?.imageUrlEncoded ??
            (list.image.trim().isNotEmpty
                ? Uri.encodeFull(list.image.trim())
                : null));

        final pbId = vm?.productBlueprintId ?? '';
        final tbId = vm?.tokenBlueprintId ?? '';

        final inv = vm?.inventory;
        final invErr = vm?.inventoryError;

        final pb = vm?.productBlueprint;
        final pbErr = vm?.productBlueprintError;

        final totalStock = vm?.totalStock;

        final tbPatch = vm?.tokenBlueprintPatch;
        final tbErr = vm?.tokenBlueprintError;
        final tokenIconUrlEncoded = vm?.tokenIconUrlEncoded;

        // ✅ リフレッシュボタン行を削除（ヘッダーのような行スペースも消える）
        return Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Card(
                clipBehavior: Clip.antiAlias,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    AspectRatio(
                      aspectRatio: 16 / 9,
                      child: hasImage && imageUrlEncoded != null
                          ? Image.network(
                              imageUrlEncoded,
                              fit: BoxFit.cover,
                              errorBuilder: (context, err, st) {
                                return _ImageFallback(
                                  label: 'image failed',
                                  detail: err.toString(),
                                );
                              },
                              loadingBuilder: (context, child, progress) {
                                if (progress == null) return child;
                                return const Center(
                                  child: CircularProgressIndicator(),
                                );
                              },
                            )
                          : const _ImageFallback(label: 'no image'),
                    ),
                    Padding(
                      padding: const EdgeInsets.all(12),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            list.title.isNotEmpty ? list.title : '(no title)',
                            style: Theme.of(context).textTheme.titleLarge,
                          ),
                          const SizedBox(height: 8),
                          if (priceText.isNotEmpty)
                            Text(
                              priceText,
                              style: Theme.of(context).textTheme.titleMedium,
                            ),
                          const SizedBox(height: 10),
                          if (list.description.trim().isNotEmpty)
                            Text(
                              list.description.trim(),
                              style: Theme.of(context).textTheme.bodyMedium,
                            ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 12),
              CatalogTokenCard(
                tokenBlueprintId: tbId,
                patch: tbPatch,
                error: tbErr,
                iconUrlEncoded: tokenIconUrlEncoded,
              ),
              const SizedBox(height: 12),
              CatalogInventoryCard(
                productBlueprintId: pbId,
                tokenBlueprintId: tbId,
                totalStock: totalStock,
                inventory: inv,
                inventoryError: invErr,
                modelStockRows: vm?.modelStockRows,
              ),
              const SizedBox(height: 12),
              CatalogProductCard(
                productBlueprintId: pbId,
                productBlueprint: pb,
                error: pbErr,
              ),
              const SizedBox(height: 24),
            ],
          ),
        );
      },
    );
  }
}

// ============================================================

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

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});

  final Object? error;
  final Future<void> Function() onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.error_outline, size: 40),
            const SizedBox(height: 12),
            Text(
              'Failed to load',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              error?.toString() ?? 'unknown error',
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 12),
            ElevatedButton(
              onPressed: () => onRetry(),
              child: const Text('Retry'),
            ),
          ],
        ),
      ),
    );
  }
}
