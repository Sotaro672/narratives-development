// frontend/sns/lib/features/home/presentation/hook/use_catalog.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../../../inventory/infrastructure/inventory_repository_http.dart';
import '../../../list/infrastructure/list_repository_http.dart';
import '../../../model/infrastructure/model_repository_http.dart';
import '../../../productBlueprint/infrastructure/product_blueprint_repository_http.dart';
import '../../../tokenBlueprint/infrastructure/token_blueprint_repository_http.dart';

/// ✅ state/logic holder for CatalogPage
class UseCatalog {
  UseCatalog({http.Client? client})
    : _catalogRepo = CatalogRepositoryHttp(client: client),
      _listRepo = ListRepositoryHttp(),
      _invRepo = InventoryRepositoryHttp(),
      _pbRepo = ProductBlueprintRepositoryHttp(),
      _modelRepo = ModelRepositoryHTTP(),
      _tbRepo = TokenBlueprintRepositoryHTTP();

  final CatalogRepositoryHttp _catalogRepo;

  final ListRepositoryHttp _listRepo;
  final InventoryRepositoryHttp _invRepo;
  final ProductBlueprintRepositoryHttp _pbRepo;
  final ModelRepositoryHTTP _modelRepo;

  // NOTE: TokenBlueprintRepositoryHTTP has no dispose() (same as ModelRepositoryHTTP).
  final TokenBlueprintRepositoryHTTP _tbRepo;

  void dispose() {
    _catalogRepo.dispose();
    _listRepo.dispose();
    _invRepo.dispose();
    _pbRepo.dispose();
    // ModelRepositoryHTTP: no dispose()
    // TokenBlueprintRepositoryHTTP: no dispose()
  }

  void _log(String msg) {
    // ignore: avoid_print
    print('[UseCatalog] $msg');
  }

  Future<CatalogState> load({required String listId}) async {
    final id = listId.trim();
    if (id.isEmpty) {
      throw Exception('catalog: listId is empty');
    }

    _log('load start listId=$id');

    // 1) try catalog endpoint first
    try {
      _log('try catalog endpoint: GET /sns/catalog/$id');
      final dto = await _catalogRepo.fetchCatalogByListId(id);

      _log(
        'catalog ok '
        'listId=${dto.list.id} '
        'inventoryId="${dto.list.inventoryId.trim()}" '
        'list.tbId="${(dto.list.tokenBlueprintId).trim()}" '
        'inv.tbId="${(dto.inventory?.tokenBlueprintId ?? '').trim()}"',
      );

      // ✅ tokenBlueprint patch (best-effort)
      // - inventory があれば inventory の tokenBlueprintId を優先
      // - なければ list.tokenBlueprintId を使う
      final resolvedTbId =
          (dto.inventory?.tokenBlueprintId ?? dto.list.tokenBlueprintId).trim();

      _log('resolved tokenBlueprintId="$resolvedTbId"');

      TokenBlueprintPatch? tbPatch;
      String? tbErr;

      if (resolvedTbId.isNotEmpty) {
        try {
          _log('fetchPatch start tokenBlueprintId=$resolvedTbId');
          tbPatch = await _tbRepo.fetchPatch(resolvedTbId);

          if (tbPatch == null) {
            tbErr = 'tokenBlueprint patch not found (404)';
            _log('fetchPatch result: null (404)');
          } else {
            _log(
              'fetchPatch ok '
              'name="${(tbPatch.name ?? '').trim()}" '
              'symbol="${(tbPatch.symbol ?? '').trim()}" '
              'brandId="${(tbPatch.brandId ?? '').trim()}" '
              'minted=${tbPatch.minted} '
              'hasIconUrl=${(tbPatch.iconUrl ?? '').trim().isNotEmpty}',
            );
          }
        } catch (e) {
          tbErr = e.toString();
          _log('fetchPatch error: $tbErr');
        }
      } else {
        tbErr = 'tokenBlueprintId is empty';
        _log('skip fetchPatch: tokenBlueprintId is empty');
      }

      final state = _buildState(
        list: dto.list,
        inventory: dto.inventory,
        inventoryError: dto.inventoryError,
        productBlueprint: dto.productBlueprint,
        productBlueprintError: dto.productBlueprintError,
        modelVariations: dto.modelVariations,
        modelVariationsError: dto.modelVariationsError,
        tokenBlueprintPatch: tbPatch,
        tokenBlueprintError: tbErr,
        // ✅ ここで resolvedTbId を渡す（inventory 無くてもIDが落ちない）
        resolvedTokenBlueprintId: resolvedTbId,
      );

      _log(
        'load done(catalog) '
        'state.tbId="${state.tokenBlueprintId}" '
        'state.tbPatch.name="${(state.tokenBlueprintPatch?.name ?? '').trim()}" '
        'state.tbErr="${state.tokenBlueprintError ?? ''}"',
      );

      return state;
    } catch (e) {
      _log('catalog endpoint failed -> fallback legacy. error=$e');
      // 2) fallback to legacy multi-fetch
      final state = await _loadLegacy(id);

      _log(
        'load done(legacy) '
        'state.tbId="${state.tokenBlueprintId}" '
        'state.tbPatch.name="${(state.tokenBlueprintPatch?.name ?? '').trim()}" '
        'state.tbErr="${state.tokenBlueprintError ?? ''}"',
      );

      return state;
    }
  }

  Future<CatalogState> _loadLegacy(String listId) async {
    _log('legacy start listId=$listId');

    final list = await _listRepo.fetchListById(listId);
    _log(
      'legacy list ok '
      'listId=${list.id} '
      'inventoryId="${list.inventoryId.trim()}" '
      'list.tbId="${list.tokenBlueprintId.trim()}"',
    );

    // inventory (must have inventoryId)
    final invId = list.inventoryId.trim();

    SnsInventoryResponse? inv;
    String? invErr;

    if (invId.isEmpty) {
      invErr = 'inventoryId is empty';
      _log('legacy inventory skip: inventoryId is empty');
    } else {
      try {
        _log('legacy fetch inventory start invId=$invId');
        inv = await _invRepo.fetchInventoryById(invId);
        _log(
          'legacy inventory ok '
          'pbId="${inv.productBlueprintId.trim()}" '
          'tbId="${inv.tokenBlueprintId.trim()}" '
          'stockKeys=${inv.stock.length}',
        );
      } catch (e) {
        invErr = e.toString();
        _log('legacy inventory error: $invErr');
      }
    }

    final pbId = (inv?.productBlueprintId ?? '').trim();

    // ✅ tokenBlueprintId は inventory が取れたらそれを優先し、無ければ list の tokenBlueprintId を利用
    final resolvedTbId = (inv?.tokenBlueprintId ?? list.tokenBlueprintId)
        .trim();
    _log('legacy resolved tokenBlueprintId="$resolvedTbId"');

    // product blueprint
    SnsProductBlueprintResponse? pb;
    String? pbErr;

    if (pbId.isNotEmpty) {
      try {
        _log('legacy fetch productBlueprint start pbId=$pbId');
        pb = await _pbRepo.fetchProductBlueprintById(pbId);
        _log('legacy productBlueprint ok productName="${pb.productName}"');
      } catch (e) {
        pbErr = e.toString();
        _log('legacy productBlueprint error: $pbErr');
      }
    } else {
      pbErr = 'productBlueprintId is unavailable (inventory not loaded)';
      _log('legacy productBlueprint skip: $pbErr');
    }

    // model variations
    List<ModelVariationDTO>? models;
    String? modelErr;

    if (pbId.isNotEmpty) {
      try {
        _log('legacy fetch model variations start pbId=$pbId');
        models = await _modelRepo.fetchModelVariationsByProductBlueprintId(
          pbId,
        );
        _log('legacy model variations ok count=${models.length}');
      } catch (e) {
        modelErr = e.toString();
        _log('legacy model variations error: $modelErr');
      }
    } else {
      modelErr = 'productBlueprintId is unavailable (skip model fetch)';
      _log('legacy model variations skip: $modelErr');
    }

    // token blueprint patch
    TokenBlueprintPatch? tbPatch;
    String? tbErr;

    if (resolvedTbId.isNotEmpty) {
      try {
        _log('legacy fetchPatch start tokenBlueprintId=$resolvedTbId');
        tbPatch = await _tbRepo.fetchPatch(resolvedTbId);

        if (tbPatch == null) {
          tbErr = 'tokenBlueprint patch not found (404)';
          _log('legacy fetchPatch result: null (404)');
        } else {
          _log(
            'legacy fetchPatch ok '
            'name="${(tbPatch.name ?? '').trim()}" '
            'symbol="${(tbPatch.symbol ?? '').trim()}" '
            'brandId="${(tbPatch.brandId ?? '').trim()}" '
            'minted=${tbPatch.minted} '
            'hasIconUrl=${(tbPatch.iconUrl ?? '').trim().isNotEmpty}',
          );
        }
      } catch (e) {
        tbErr = e.toString();
        _log('legacy fetchPatch error: $tbErr');
      }
    } else {
      tbErr = 'tokenBlueprintId is empty';
      _log('legacy fetchPatch skip: tokenBlueprintId is empty');
    }

    return _buildState(
      list: list,
      inventory: inv,
      inventoryError: invErr,
      productBlueprint: pb,
      productBlueprintError: pbErr,
      modelVariations: models,
      modelVariationsError: modelErr,
      tokenBlueprintPatch: tbPatch,
      tokenBlueprintError: tbErr,
      // ✅ legacy でも resolvedTbId を渡す
      resolvedTokenBlueprintId: resolvedTbId,
    );
  }

  CatalogState _buildState({
    required SnsListItem list,
    required SnsInventoryResponse? inventory,
    required String? inventoryError,
    required SnsProductBlueprintResponse? productBlueprint,
    required String? productBlueprintError,
    required List<ModelVariationDTO>? modelVariations,
    required String? modelVariationsError,
    required TokenBlueprintPatch? tokenBlueprintPatch,
    required String? tokenBlueprintError,

    // ✅ NEW: tokenBlueprintId を inventory だけに依存させないための引数
    required String resolvedTokenBlueprintId,
  }) {
    _log(
      '_buildState in '
      'listId=${list.id} '
      'inv?=${inventory != null} '
      'resolvedTbId="${resolvedTokenBlueprintId.trim()}" '
      'patch.name="${(tokenBlueprintPatch?.name ?? '').trim()}" '
      'patch.symbol="${(tokenBlueprintPatch?.symbol ?? '').trim()}" '
      'patch.brandId="${(tokenBlueprintPatch?.brandId ?? '').trim()}" '
      'patch.minted=${tokenBlueprintPatch?.minted} '
      'tbErr="${tokenBlueprintError ?? ''}"',
    );

    final imageUrl = list.image.trim();
    final hasImage = imageUrl.isNotEmpty;

    final priceText = _priceText(list.prices);

    // ✅ productBlueprintId は inventory 優先（list には基本無い/信頼しない方針）
    final pbId = (inventory?.productBlueprintId ?? '').trim();

    // ✅ tokenBlueprintId は resolved（inventory優先→list fallback）
    final tbId = resolvedTokenBlueprintId.trim();

    final totalStock = inventory != null ? _totalStock(inventory) : null;

    // modelId -> variation
    final modelMap = <String, ModelVariationDTO>{};
    if (modelVariations != null) {
      for (final v in modelVariations) {
        final id = v.id.trim();
        if (id.isNotEmpty) modelMap[id] = v;
      }
    }

    final modelStockRows = <CatalogModelStockRow>[];
    if (inventory != null) {
      for (final e in inventory.stock.entries) {
        final modelId = e.key.trim();
        final stock = e.value;
        final meta = modelMap[modelId];

        final label = meta != null
            ? _modelLabel(meta)
            : (modelId.isNotEmpty ? modelId : '(no model)');

        final count = _stockCount(stock);

        modelStockRows.add(
          CatalogModelStockRow(
            modelId: modelId,
            label: label,
            stockCount: count,
          ),
        );
      }
    }

    final tokenIconUrl = (tokenBlueprintPatch?.iconUrl ?? '').trim();

    final state = CatalogState(
      list: list,
      priceText: priceText,
      imageUrl: imageUrl,
      imageUrlEncoded: _safeUrl(imageUrl),
      hasImage: hasImage,

      inventory: inventory,
      inventoryError: _asNonEmptyString(inventoryError),
      productBlueprint: productBlueprint,
      productBlueprintError: _asNonEmptyString(productBlueprintError),
      modelVariations: modelVariations,
      modelVariationsError: _asNonEmptyString(modelVariationsError),

      productBlueprintId: pbId,
      tokenBlueprintId: tbId, // ✅ resolved を反映
      totalStock: totalStock,
      modelStockRows: modelStockRows,

      tokenBlueprintPatch: tokenBlueprintPatch,
      tokenBlueprintError: _asNonEmptyString(tokenBlueprintError),
      tokenIconUrlEncoded: tokenIconUrl.isNotEmpty
          ? _safeUrl(tokenIconUrl)
          : null,
    );

    _log(
      '_buildState out '
      'state.tbId="${state.tokenBlueprintId}" '
      'state.tbPatch.name="${(state.tokenBlueprintPatch?.name ?? '').trim()}" '
      'state.tbErr="${state.tokenBlueprintError ?? ''}" '
      'state.hasTokenIcon=${(state.tokenIconUrlEncoded ?? '').trim().isNotEmpty}',
    );

    return state;
  }

  static String? _asNonEmptyString(String? v) {
    final s = (v ?? '').trim();
    return s.isEmpty ? null : s;
  }

  static String _safeUrl(String raw) => Uri.encodeFull(raw.trim());

  static String _priceText(List<SnsListPriceRow> rows) {
    if (rows.isEmpty) return '';
    final prices = rows.map((e) => e.price).toList()..sort();
    final min = prices.first;
    final max = prices.last;
    if (min == max) return '¥$min';
    return '¥$min 〜 ¥$max';
  }

  static int _stockCount(SnsInventoryModelStock s) {
    if (s.products.isEmpty) return 0;
    var n = 0;
    for (final v in s.products.values) {
      if (v == true) n++;
    }
    return n;
  }

  static int _totalStock(SnsInventoryResponse inv) {
    var sum = 0;
    for (final v in inv.stock.values) {
      sum += _stockCount(v);
    }
    return sum;
  }

  static String _modelLabel(ModelVariationDTO v) {
    final parts = <String>[];
    if (v.modelNumber.trim().isNotEmpty) parts.add(v.modelNumber.trim());
    if (v.size.trim().isNotEmpty) parts.add(v.size.trim());
    final color = v.color.name.trim();
    if (color.isNotEmpty) parts.add(color);
    if (parts.isEmpty) return '(empty)';
    return parts.join(' / ');
  }
}

// ============================================================
// ViewModel for CatalogPage (UI-friendly)
// ============================================================

class CatalogState {
  const CatalogState({
    required this.list,
    required this.priceText,
    required this.imageUrl,
    required this.imageUrlEncoded,
    required this.hasImage,
    required this.inventory,
    required this.inventoryError,
    required this.productBlueprint,
    required this.productBlueprintError,
    required this.modelVariations,
    required this.modelVariationsError,
    required this.productBlueprintId,
    required this.tokenBlueprintId,
    required this.totalStock,
    required this.modelStockRows,
    required this.tokenBlueprintPatch,
    required this.tokenBlueprintError,
    required this.tokenIconUrlEncoded,
  });

  final SnsListItem list;

  final String priceText;

  final String imageUrl;
  final String imageUrlEncoded;
  final bool hasImage;

  final SnsInventoryResponse? inventory;
  final String? inventoryError;

  final SnsProductBlueprintResponse? productBlueprint;
  final String? productBlueprintError;

  final List<ModelVariationDTO>? modelVariations;
  final String? modelVariationsError;

  final String productBlueprintId;
  final String tokenBlueprintId;
  final int? totalStock;

  final List<CatalogModelStockRow> modelStockRows;

  final TokenBlueprintPatch? tokenBlueprintPatch;
  final String? tokenBlueprintError;
  final String? tokenIconUrlEncoded;
}

class CatalogModelStockRow {
  const CatalogModelStockRow({
    required this.modelId,
    required this.label,
    required this.stockCount,
  });

  final String modelId;
  final String label;
  final int stockCount;
}

// ============================================================
// CatalogRepositoryHttp + DTO (matches backend SNSCatalogDTO)
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
    const env = String.fromEnvironment('API_BASE');
    if (env.trim().isNotEmpty) return env.trim();
    throw Exception(
      'API_BASE is not set (use --dart-define=API_BASE=https://...)',
    );
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
