// frontend/sns/lib/features/home/presentation/page/catalog.dart
import 'package:flutter/material.dart';

import '../../../inventory/infrastructure/inventory_repository_http.dart';
import '../../../list/infrastructure/list_repository_http.dart';

// ✅ NEW: productBlueprint を引く
import '../../../productBlueprint/infrastructure/product_blueprint_repository_http.dart';

/// 商品詳細ページ（buyer-facing）
/// - list 詳細: GET /sns/lists/{listId}
/// - inventory: GET /sns/inventories/{id} OR GET /sns/inventories?pb&tb
/// - productBlueprint: GET /sns/product-blueprints/{productBlueprintId}
class CatalogPage extends StatefulWidget {
  const CatalogPage({super.key, required this.listId, this.initialItem});

  final String listId;
  final SnsListItem? initialItem;

  static Route<void> route({required String listId, SnsListItem? initialItem}) {
    return MaterialPageRoute(
      builder: (_) => CatalogPage(listId: listId, initialItem: initialItem),
    );
  }

  @override
  State<CatalogPage> createState() => _CatalogPageState();
}

class _CatalogPageState extends State<CatalogPage> {
  late final ListRepositoryHttp _listRepo;
  late final InventoryRepositoryHttp _invRepo;

  // ✅ NEW
  late final ProductBlueprintRepositoryHttp _pbRepo;

  late Future<_CatalogPayload> _future;

  @override
  void initState() {
    super.initState();
    _listRepo = ListRepositoryHttp();
    _invRepo = InventoryRepositoryHttp();
    _pbRepo = ProductBlueprintRepositoryHttp();
    _future = _load();
  }

  @override
  void dispose() {
    _listRepo.dispose();
    _invRepo.dispose();
    _pbRepo.dispose();
    super.dispose();
  }

  Future<_CatalogPayload> _load() async {
    final list = await _listRepo.fetchListById(widget.listId);

    // ✅ inventory linkage (優先順位)
    // 1) inventoryId があれば /sns/inventories/{id}
    // 2) productBlueprintId + tokenBlueprintId があれば /sns/inventories?pb&tb
    final invId = list.inventoryId.trim();
    final listPbId = list.productBlueprintId.trim();
    final listTbId = list.tokenBlueprintId.trim();

    SnsInventoryResponse? inv;
    String? invErr;

    if (invId.isEmpty && (listPbId.isEmpty || listTbId.isEmpty)) {
      invErr = 'inventory linkage is missing (inventoryId or pb/tb)';
    } else {
      try {
        inv = invId.isNotEmpty
            ? await _invRepo.fetchInventoryById(invId)
            : await _invRepo.fetchInventoryByQuery(
                productBlueprintId: listPbId,
                tokenBlueprintId: listTbId,
              );
      } catch (e) {
        invErr = e.toString();
      }
    }

    // ✅ productBlueprintId は inventory 側が取れたらそちらを優先
    final pbId = (inv?.productBlueprintId ?? list.productBlueprintId).trim();

    SnsProductBlueprintResponse? pb;
    String? pbErr;

    if (pbId.isNotEmpty) {
      try {
        pb = await _pbRepo.fetchProductBlueprintById(pbId);
      } catch (e) {
        pbErr = e.toString();
      }
    } else {
      pbErr = 'productBlueprintId is empty';
    }

    return _CatalogPayload(
      list: list,
      inventory: inv,
      inventoryError: invErr,
      productBlueprint: pb,
      productBlueprintError: pbErr,
    );
  }

  Future<void> _reload() async {
    setState(() {
      _future = _load();
    });
  }

  String _safeUrl(String raw) => Uri.encodeFull(raw.trim());

  String _priceText(List<SnsListPriceRow> rows) {
    if (rows.isEmpty) return '';
    final prices = rows.map((e) => e.price).toList()..sort();
    final min = prices.first;
    final max = prices.last;
    if (min == max) return '¥$min';
    return '¥$min 〜 ¥$max';
  }

  int _totalStock(SnsInventoryResponse inv) {
    var sum = 0;
    for (final v in inv.stock.values) {
      sum += v.accumulation;
    }
    return sum;
  }

  @override
  Widget build(BuildContext context) {
    final initial = widget.initialItem;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Catalog'),
        actions: [
          IconButton(
            onPressed: _reload,
            icon: const Icon(Icons.refresh),
            tooltip: 'Reload',
          ),
        ],
      ),
      body: FutureBuilder<_CatalogPayload>(
        future: _future,
        builder: (context, snap) {
          final payload = snap.data;
          final list = payload?.list ?? initial;

          if (snap.connectionState == ConnectionState.waiting && list == null) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snap.hasError && list == null) {
            return _ErrorView(error: snap.error, onRetry: _reload);
          }
          if (list == null) {
            return const Center(child: Text('No data'));
          }

          final inv = payload?.inventory;
          final invErr = payload?.inventoryError;

          final pb = payload?.productBlueprint;
          final pbErr = payload?.productBlueprintError;

          final imageUrl = list.image.trim();
          final hasImage = imageUrl.isNotEmpty;
          final price = _priceText(list.prices);

          final total = inv != null ? _totalStock(inv) : null;

          // 表示用（inventory 側が取れたらそれを優先）
          final pbId = (inv?.productBlueprintId ?? list.productBlueprintId)
              .trim();
          final tbId = (inv?.tokenBlueprintId ?? list.tokenBlueprintId).trim();

          return ListView(
            padding: const EdgeInsets.all(12),
            children: [
              // -------- list --------
              Card(
                clipBehavior: Clip.antiAlias,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    AspectRatio(
                      aspectRatio: 16 / 9,
                      child: hasImage
                          ? Image.network(
                              _safeUrl(imageUrl),
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
                          if (price.isNotEmpty)
                            Text(
                              price,
                              style: Theme.of(context).textTheme.titleMedium,
                            ),
                          const SizedBox(height: 10),
                          if (list.description.trim().isNotEmpty)
                            Text(
                              list.description.trim(),
                              style: Theme.of(context).textTheme.bodyMedium,
                            ),
                          const SizedBox(height: 12),
                          Text(
                            'listId: ${list.id}',
                            style: Theme.of(context).textTheme.labelSmall,
                          ),
                          Text(
                            'inventoryId: ${list.inventoryId.isNotEmpty ? list.inventoryId : '(empty)'}',
                            style: Theme.of(context).textTheme.labelSmall,
                          ),
                          Text(
                            'productBlueprintId: ${pbId.isNotEmpty ? pbId : '(empty)'}',
                            style: Theme.of(context).textTheme.labelSmall,
                          ),
                          Text(
                            'tokenBlueprintId: ${tbId.isNotEmpty ? tbId : '(empty)'}',
                            style: Theme.of(context).textTheme.labelSmall,
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 12),

              // -------- inventory --------
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Inventory',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 8),
                      _KeyValueRow(
                        label: 'productBlueprintId',
                        value: pbId.isNotEmpty ? pbId : '(unknown)',
                      ),
                      const SizedBox(height: 6),
                      _KeyValueRow(
                        label: 'tokenBlueprintId',
                        value: tbId.isNotEmpty ? tbId : '(unknown)',
                      ),
                      const SizedBox(height: 6),
                      _KeyValueRow(
                        label: 'total stock',
                        value: total != null
                            ? total.toString()
                            : '(not loaded)',
                      ),
                      if (inv != null) ...[
                        const SizedBox(height: 12),
                        Text(
                          'By model',
                          style: Theme.of(context).textTheme.titleSmall,
                        ),
                        const SizedBox(height: 6),
                        ...inv.stock.entries.map((e) {
                          final modelId = e.key;
                          final stock = e.value;
                          return Padding(
                            padding: const EdgeInsets.symmetric(vertical: 6),
                            child: Row(
                              children: [
                                Expanded(
                                  child: Text(
                                    modelId.isNotEmpty ? modelId : '(no model)',
                                  ),
                                ),
                                Text('stock: ${stock.accumulation}'),
                              ],
                            ),
                          );
                        }),
                      ] else ...[
                        if (invErr != null && invErr.trim().isNotEmpty) ...[
                          const SizedBox(height: 10),
                          Text(
                            'inventory error: $invErr',
                            style: Theme.of(context).textTheme.labelSmall,
                          ),
                        ],
                      ],
                    ],
                  ),
                ),
              ),

              const SizedBox(height: 12),

              // -------- product blueprint (NEW) --------
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Product',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 8),

                      // ✅ 要件:
                      // - productBlueprintId から情報表示
                      // - assigneeId / createdAt / createdBy / updatedAt / updatedBy は表示しない
                      // - deletedAt / deletedBy も拾わない（＝表示しない）
                      if (pb != null) ...[
                        _KeyValueRow(
                          label: 'productName',
                          value: pb.productName.isNotEmpty
                              ? pb.productName
                              : '(empty)',
                        ),
                        const SizedBox(height: 6),
                        _KeyValueRow(
                          label: 'brandId',
                          value: pb.brandId.isNotEmpty ? pb.brandId : '(empty)',
                        ),
                        const SizedBox(height: 6),
                        _KeyValueRow(
                          label: 'companyId',
                          value: pb.companyId.isNotEmpty
                              ? pb.companyId
                              : '(empty)',
                        ),
                        const SizedBox(height: 6),
                        _KeyValueRow(
                          label: 'itemType',
                          value: pb.itemType.isNotEmpty
                              ? pb.itemType
                              : '(empty)',
                        ),
                        const SizedBox(height: 6),
                        _KeyValueRow(
                          label: 'fit',
                          value: pb.fit.isNotEmpty ? pb.fit : '(empty)',
                        ),
                        const SizedBox(height: 6),
                        _KeyValueRow(
                          label: 'material',
                          value: pb.material.isNotEmpty
                              ? pb.material
                              : '(empty)',
                        ),
                        const SizedBox(height: 6),
                        _KeyValueRow(
                          label: 'weight',
                          value: pb.weight != null ? '${pb.weight}' : '(empty)',
                        ),
                        const SizedBox(height: 6),
                        _KeyValueRow(
                          label: 'printed',
                          value: pb.printed == true ? 'true' : 'false',
                        ),
                        const SizedBox(height: 12),
                        Text(
                          'Quality assurance',
                          style: Theme.of(context).textTheme.titleSmall,
                        ),
                        const SizedBox(height: 6),
                        if (pb.qualityAssurance.isEmpty)
                          Text(
                            '(empty)',
                            style: Theme.of(context).textTheme.bodyMedium,
                          )
                        else
                          Wrap(
                            spacing: 8,
                            runSpacing: 8,
                            children: pb.qualityAssurance
                                .map(
                                  (s) => Chip(
                                    label: Text(s),
                                    visualDensity: VisualDensity.compact,
                                  ),
                                )
                                .toList(),
                          ),
                        const SizedBox(height: 12),
                        Text(
                          'ProductId tag',
                          style: Theme.of(context).textTheme.titleSmall,
                        ),
                        const SizedBox(height: 6),
                        _KeyValueRow(
                          label: 'type',
                          value: pb.productIdTagType.isNotEmpty
                              ? pb.productIdTagType
                              : '(empty)',
                        ),
                      ] else ...[
                        _KeyValueRow(
                          label: 'productBlueprintId',
                          value: pbId.isNotEmpty ? pbId : '(unknown)',
                        ),
                        if (pbErr != null && pbErr.trim().isNotEmpty) ...[
                          const SizedBox(height: 10),
                          Text(
                            'product error: $pbErr',
                            style: Theme.of(context).textTheme.labelSmall,
                          ),
                        ] else ...[
                          const SizedBox(height: 10),
                          Text(
                            'product is not loaded',
                            style: Theme.of(context).textTheme.labelSmall,
                          ),
                        ],
                      ],
                    ],
                  ),
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

class _CatalogPayload {
  const _CatalogPayload({
    required this.list,
    required this.inventory,
    required this.inventoryError,
    required this.productBlueprint,
    required this.productBlueprintError,
  });

  final SnsListItem list;
  final SnsInventoryResponse? inventory;
  final String? inventoryError;

  // ✅ NEW
  final SnsProductBlueprintResponse? productBlueprint;
  final String? productBlueprintError;
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
