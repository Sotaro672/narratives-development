// frontend/sns/lib/features/home/presentation/page/catalog.dart
import 'package:flutter/material.dart';

import '../../../list/infrastructure/list_repository_http.dart'; // SnsListItem, SnsListPriceRow
import '../../../tokenBlueprint/infrastructure/token_blueprint_repository_http.dart'
    show TokenBlueprintPatch;
import '../hook/use_catalog.dart';

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
  late final UseCatalog _uc;
  late Future<CatalogState> _future;

  // ✅ Render時に「画面へ渡っているか」を確認するログ（スパム抑制）
  String _lastVmLogKey = '';

  void _log(String msg) {
    // ignore: avoid_print
    print('[CatalogPage] $msg');
  }

  void _logVmOnce(CatalogState vm) {
    final key = [
      vm.list.id,
      vm.inventory != null ? 'inv' : 'noinv',
      vm.productBlueprintId,
      vm.tokenBlueprintId,
      vm.tokenBlueprintPatch?.name ?? '',
      vm.tokenBlueprintPatch?.symbol ?? '',
      vm.tokenBlueprintPatch?.brandId ?? '',
      (vm.tokenBlueprintPatch?.minted ?? '').toString(),
      vm.tokenBlueprintError ?? '',
      (vm.tokenIconUrlEncoded ?? '').isNotEmpty ? 'icon' : 'noicon',
    ].join('|');

    if (key == _lastVmLogKey) return;
    _lastVmLogKey = key;

    _log(
      'vm received '
      'listId=${vm.list.id} '
      'inv?=${vm.inventory != null} '
      'pbId="${vm.productBlueprintId}" '
      'tbId="${vm.tokenBlueprintId}" '
      'patch.name="${vm.tokenBlueprintPatch?.name ?? ''}" '
      'patch.symbol="${vm.tokenBlueprintPatch?.symbol ?? ''}" '
      'patch.brandId="${vm.tokenBlueprintPatch?.brandId ?? ''}" '
      'patch.minted=${vm.tokenBlueprintPatch?.minted} '
      'tbErr="${vm.tokenBlueprintError ?? ''}" '
      'hasTokenIcon=${(vm.tokenIconUrlEncoded ?? '').trim().isNotEmpty}',
    );
  }

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
      body: FutureBuilder<CatalogState>(
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

          // ✅ ここで「画面へ渡っている vm」をログで確認
          if (vm != null) {
            _logVmOnce(vm);
          } else {
            _log(
              'vm is null (initial only) listId=${list.id} title="${list.title}"',
            );
          }

          final priceText = vm?.priceText ?? '';
          final hasImage = vm?.hasImage ?? list.image.trim().isNotEmpty;

          final imageUrlEncoded = vm?.imageUrlEncoded;
          final pbId = vm?.productBlueprintId ?? '';
          final tbId = vm?.tokenBlueprintId ?? '';

          final inv = vm?.inventory;
          final invErr = vm?.inventoryError;

          final pb = vm?.productBlueprint;
          final pbErr = vm?.productBlueprintError;

          final models = vm?.modelVariations;
          final modelErr = vm?.modelVariationsError;

          final totalStock = vm?.totalStock;

          final tbPatch = vm?.tokenBlueprintPatch;
          final tbErr = vm?.tokenBlueprintError;
          final tokenIconUrlEncoded = vm?.tokenIconUrlEncoded;

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

              // ✅ token blueprint card (patch)
              _TokenBlueprintCard(
                tokenBlueprintId: tbId,
                patch: tbPatch,
                error: tbErr,
                iconUrlEncoded: tokenIconUrlEncoded,
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
                        value: totalStock != null
                            ? totalStock.toString()
                            : '(not loaded)',
                      ),
                      if (inv != null) ...[
                        const SizedBox(height: 12),
                        Text(
                          'By model',
                          style: Theme.of(context).textTheme.titleSmall,
                        ),
                        const SizedBox(height: 6),
                        ...vm!.modelStockRows.map((r) {
                          final modelId = r.modelId;
                          final count = r.stockCount;

                          return Padding(
                            padding: const EdgeInsets.symmetric(vertical: 6),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  r.label,
                                  style: Theme.of(context).textTheme.bodyMedium,
                                ),
                                const SizedBox(height: 2),
                                Text(
                                  'modelId: ${modelId.isNotEmpty ? modelId : '(empty)'}   stock: $count',
                                  style: Theme.of(context).textTheme.labelSmall,
                                ),
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

              // -------- product blueprint --------
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

              const SizedBox(height: 12),

              // -------- model variations --------
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Model',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 8),
                      _KeyValueRow(
                        label: 'productBlueprintId',
                        value: pbId.isNotEmpty ? pbId : '(unknown)',
                      ),
                      const SizedBox(height: 10),
                      if (models != null) ...[
                        if (models.isEmpty)
                          Text(
                            '(empty)',
                            style: Theme.of(context).textTheme.bodyMedium,
                          )
                        else
                          ...models.map((v) {
                            final mId = v.id.trim();
                            return Padding(
                              padding: const EdgeInsets.symmetric(vertical: 8),
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(
                                    [
                                      v.modelNumber.trim(),
                                      v.size.trim(),
                                      v.color.name.trim(),
                                    ].where((s) => s.isNotEmpty).join(' / '),
                                    style: Theme.of(
                                      context,
                                    ).textTheme.bodyLarge,
                                  ),
                                  const SizedBox(height: 4),
                                  Text(
                                    'modelId: ${mId.isNotEmpty ? mId : '(empty)'}',
                                    style: Theme.of(
                                      context,
                                    ).textTheme.labelSmall,
                                  ),
                                  if (v.measurements.isNotEmpty) ...[
                                    const SizedBox(height: 6),
                                    Wrap(
                                      spacing: 8,
                                      runSpacing: 8,
                                      children: v.measurements.entries.map((e) {
                                        return Chip(
                                          label: Text('${e.key}: ${e.value}'),
                                          visualDensity: VisualDensity.compact,
                                        );
                                      }).toList(),
                                    ),
                                  ],
                                ],
                              ),
                            );
                          }),
                      ] else ...[
                        if (modelErr != null && modelErr.trim().isNotEmpty)
                          Text(
                            'model error: $modelErr',
                            style: Theme.of(context).textTheme.labelSmall,
                          )
                        else
                          Text(
                            'model is not loaded',
                            style: Theme.of(context).textTheme.labelSmall,
                          ),
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

// ============================================================
// UI components (style-only)
// ============================================================

class _TokenBlueprintCard extends StatelessWidget {
  const _TokenBlueprintCard({
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
