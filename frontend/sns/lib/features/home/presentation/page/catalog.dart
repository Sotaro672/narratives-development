// frontend/sns/lib/features/home/presentation/page/catalog.dart
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

import '../../../inventory/infrastructure/inventory_repository_http.dart';
import '../../../list/infrastructure/list_repository_http.dart';

// ✅ productBlueprint
import '../../../productBlueprint/infrastructure/product_blueprint_repository_http.dart';

// ✅ model
import '../../../model/infrastructure/model_repository_http.dart';

/// 商品詳細ページ（buyer-facing）
///
/// ✅ 推奨（新）:
/// - catalog DTO: GET /sns/catalog/{listId}
///
/// fallback（旧）:
/// - list 詳細: GET /sns/lists/{listId}
/// - inventory: GET /sns/inventories/{id} OR GET /sns/inventories?pb&tb
/// - productBlueprint: GET /sns/product-blueprints/{productBlueprintId}
/// - model variations: GET /sns/models?productBlueprintId=...
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
  // ✅ NEW: catalog endpoint
  late final CatalogRepositoryHttp _catalogRepo;

  // fallback legacy repos
  late final ListRepositoryHttp _listRepo;
  late final InventoryRepositoryHttp _invRepo;
  late final ProductBlueprintRepositoryHttp _pbRepo;
  late final ModelRepositoryHTTP _modelRepo;

  late Future<_CatalogPayload> _future;

  @override
  void initState() {
    super.initState();
    _catalogRepo = CatalogRepositoryHttp();

    _listRepo = ListRepositoryHttp();
    _invRepo = InventoryRepositoryHttp();
    _pbRepo = ProductBlueprintRepositoryHttp();
    _modelRepo = ModelRepositoryHTTP();

    _future = _load();
  }

  @override
  void dispose() {
    _catalogRepo.dispose();

    _listRepo.dispose();
    _invRepo.dispose();
    _pbRepo.dispose();
    // NOTE: ModelRepositoryHTTP currently has no dispose(). Keep it as-is.
    super.dispose();
  }

  Future<_CatalogPayload> _load() async {
    // 1) try catalog endpoint first
    try {
      final dto = await _catalogRepo.fetchCatalogByListId(widget.listId);

      return _CatalogPayload(
        list: dto.list,
        inventory: dto.inventory,
        inventoryError: dto.inventoryError,
        productBlueprint: dto.productBlueprint,
        productBlueprintError: dto.productBlueprintError,
        modelVariations: dto.modelVariations,
        modelVariationsError: dto.modelVariationsError,
      );
    } catch (_) {
      // 2) fallback to legacy multi-fetch (keeps the app usable while backend is being wired)
      return _loadLegacy();
    }
  }

  Future<_CatalogPayload> _loadLegacy() async {
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

    // ✅ model variations（productBlueprintId が取れたら取得）
    List<ModelVariationDTO>? models;
    String? modelErr;

    if (pbId.isNotEmpty) {
      try {
        models = await _modelRepo.fetchModelVariationsByProductBlueprintId(
          pbId,
        );
      } catch (e) {
        modelErr = e.toString();
      }
    } else {
      modelErr = 'productBlueprintId is empty (skip model fetch)';
    }

    return _CatalogPayload(
      list: list,
      inventory: inv,
      inventoryError: invErr,
      productBlueprint: pb,
      productBlueprintError: pbErr,
      modelVariations: models,
      modelVariationsError: modelErr,
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

  String _modelLabel(ModelVariationDTO v) {
    final parts = <String>[];
    if (v.modelNumber.trim().isNotEmpty) parts.add(v.modelNumber.trim());
    if (v.size.trim().isNotEmpty) parts.add(v.size.trim());
    final color = v.color.name.trim();
    if (color.isNotEmpty) parts.add(color);
    if (parts.isEmpty) return '(empty)';
    return parts.join(' / ');
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

          final models = payload?.modelVariations;
          final modelErr = payload?.modelVariationsError;

          final imageUrl = list.image.trim();
          final hasImage = imageUrl.isNotEmpty;
          final price = _priceText(list.prices);

          final total = inv != null ? _totalStock(inv) : null;

          // 表示用（inventory 側が取れたらそれを優先）
          final pbId = (inv?.productBlueprintId ?? list.productBlueprintId)
              .trim();
          final tbId = (inv?.tokenBlueprintId ?? list.tokenBlueprintId).trim();

          // modelId -> metadata (variation)
          final modelMap = <String, ModelVariationDTO>{};
          if (models != null) {
            for (final v in models) {
              final id = v.id.trim();
              if (id.isNotEmpty) modelMap[id] = v;
            }
          }

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
                          final meta = modelMap[modelId.trim()];

                          return Padding(
                            padding: const EdgeInsets.symmetric(vertical: 6),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  meta != null
                                      ? _modelLabel(meta)
                                      : (modelId.isNotEmpty
                                            ? modelId
                                            : '(no model)'),
                                  style: Theme.of(context).textTheme.bodyMedium,
                                ),
                                const SizedBox(height: 2),
                                Text(
                                  'modelId: ${modelId.isNotEmpty ? modelId : '(empty)'}   stock: ${stock.accumulation}',
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
                        else ...[
                          ...models.map((v) {
                            final mId = v.id.trim();
                            return Padding(
                              padding: const EdgeInsets.symmetric(vertical: 8),
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(
                                    _modelLabel(v),
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
                        ],
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

class _CatalogPayload {
  const _CatalogPayload({
    required this.list,
    required this.inventory,
    required this.inventoryError,
    required this.productBlueprint,
    required this.productBlueprintError,
    required this.modelVariations,
    required this.modelVariationsError,
  });

  final SnsListItem list;
  final SnsInventoryResponse? inventory;
  final String? inventoryError;

  final SnsProductBlueprintResponse? productBlueprint;
  final String? productBlueprintError;

  final List<ModelVariationDTO>? modelVariations;
  final String? modelVariationsError;
}

// ============================================================
// ✅ NEW: CatalogRepositoryHttp + DTO (matches backend SNSCatalogDTO)
// ============================================================

class SnsCatalogDTO {
  const SnsCatalogDTO({
    required this.list,
    required this.inventory,
    required this.inventoryError,
    required this.productBlueprint,
    required this.productBlueprintError,
    required this.modelVariations,
    required this.modelVariationsError,
  });

  final SnsListItem list;

  final SnsInventoryResponse? inventory;
  final String? inventoryError;

  final SnsProductBlueprintResponse? productBlueprint;
  final String? productBlueprintError;

  final List<ModelVariationDTO>? modelVariations;
  final String? modelVariationsError;

  static String? _asNonEmptyString(dynamic v) {
    final s = (v ?? '').toString().trim();
    return s.isEmpty ? null : s;
  }

  factory SnsCatalogDTO.fromJson(Map<String, dynamic> json) {
    // NOTE:
    // - list / inventory / productBlueprint / modelVariations are nested objects
    // - errors are strings (empty string means "no error" from backend)
    final listJson =
        (json['list'] as Map?)?.cast<String, dynamic>() ?? const {};
    final invJson = (json['inventory'] as Map?)?.cast<String, dynamic>();
    final pbJson = (json['productBlueprint'] as Map?)?.cast<String, dynamic>();
    final mvJson = json['modelVariations'];

    return SnsCatalogDTO(
      list: SnsListItem.fromJson(listJson),
      inventory: invJson != null
          ? SnsInventoryResponse.fromJson(invJson)
          : null,
      inventoryError: _asNonEmptyString(json['inventoryError']),
      productBlueprint: pbJson != null
          ? SnsProductBlueprintResponse.fromJson(pbJson)
          : null,
      productBlueprintError: _asNonEmptyString(json['productBlueprintError']),
      modelVariations: (mvJson is List)
          ? mvJson
                .whereType<Map>()
                .map(
                  (e) => ModelVariationDTO.fromJson(e.cast<String, dynamic>()),
                )
                .toList()
          : null,
      modelVariationsError: _asNonEmptyString(json['modelVariationsError']),
    );
  }
}

class CatalogRepositoryHttp {
  CatalogRepositoryHttp({http.Client? client})
    : _client = client ?? http.Client();

  final http.Client _client;

  void dispose() {
    _client.close();
  }

  static String _resolveApiBase() {
    // flutter run --dart-define=API_BASE=https://...
    const env = String.fromEnvironment('API_BASE');
    if (env.trim().isNotEmpty) return env.trim();

    // fallback (Cloud Run)
    return 'https://narratives-backend-871263659099.asia-northeast1.run.app';
  }

  static Uri _buildUri(String path) {
    final base = _resolveApiBase().replaceAll(RegExp(r'\/+$'), '');
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$base$p');
  }

  Future<SnsCatalogDTO> fetchCatalogByListId(String listId) async {
    final id = listId.trim();
    if (id.isEmpty) {
      throw Exception('catalog: listId is empty');
    }

    final uri = _buildUri('/sns/catalog/$id');
    final res = await _client.get(uri, headers: {'accept': 'application/json'});

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw Exception('catalog: http ${res.statusCode} body=${res.body}');
    }

    final jsonObj = jsonDecode(res.body);
    if (jsonObj is! Map) {
      throw Exception('catalog: invalid json (not an object)');
    }

    return SnsCatalogDTO.fromJson(jsonObj.cast<String, dynamic>());
  }
}

// ============================================================
// UI components
// ============================================================

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
