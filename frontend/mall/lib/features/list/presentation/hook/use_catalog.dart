// frontend/mall/lib/features/list/presentation/hook/use_catalog.dart
import 'dart:convert';
import 'package:http/http.dart' as http;

import '../../../inventory/infrastructure/inventory_repository_http.dart';
import '../../infrastructure/list_repository_http.dart';
import '../../../productBlueprint/infrastructure/product_blueprint_repository_http.dart';
import '../../../tokenBlueprint/infrastructure/token_blueprint_repository_http.dart';
import '../../../../app/config/api_base.dart';

import 'use_catalog_inventory.dart';
import 'use_catalog_product.dart';
import 'use_catalog_token.dart';
import 'use_catalog_measurement.dart';
import '../../../../app/shell/presentation/state/catalog_selection_store.dart';

// ============================================================
// ✅ NEW: listImages DTO (for catalog.dart carousel)
// ============================================================

class CatalogListImage {
  const CatalogListImage({
    required this.id,
    required this.url,
    required this.objectPath,
    required this.fileName,
    required this.size,
    required this.displayOrder,
  });

  final String id;
  final String url;
  final String objectPath;
  final String fileName;
  final int? size;

  /// 1..N (unknown: null)
  final int? displayOrder;

  factory CatalogListImage.fromJson(Map<String, dynamic> j) {
    String s(dynamic v) => (v ?? '').toString().trim();

    int? i(dynamic v) {
      if (v is int) return v;
      if (v is num) return v.toInt();
      final t = (v ?? '').toString().trim();
      if (t.isEmpty) return null;
      return int.tryParse(t);
    }

    return CatalogListImage(
      id: s(j['id']),
      url: s(j['url']),
      objectPath: s(j['objectPath']),
      fileName: s(j['fileName']),
      size: (j['size'] is int)
          ? (j['size'] as int)
          : (j['size'] is num)
          ? (j['size'] as num).toInt()
          : i(j['size']),
      displayOrder: i(j['displayOrder']),
    );
  }
}

/// ✅ state/logic holder for CatalogPage
class UseCatalog {
  UseCatalog({http.Client? client})
    : _catalogRepo = CatalogRepositoryHttp(client: client),
      _invRepo = InventoryRepositoryHttp(),
      _inventory = const UseCatalogInventory(),
      _measurement = const UseCatalogMeasurement(),
      _product = UseCatalogProduct(),
      _token = UseCatalogToken();

  final CatalogRepositoryHttp _catalogRepo;

  // legacy list/inventory repos are removed
  final InventoryRepositoryHttp _invRepo;

  // ✅ inventory hook（モデル取得 + 計算）
  final UseCatalogInventory _inventory;

  // ✅ measurement hook（採寸テーブル計算）
  final UseCatalogMeasurement _measurement;

  // ✅ product / token
  final UseCatalogProduct _product;
  final UseCatalogToken _token;

  void dispose() {
    _catalogRepo.dispose();
    _invRepo.dispose();
    _measurement.dispose();
    _product.dispose();
    _token.dispose();
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

    // ✅ 先に listId だけ入れておく（画面初期化）
    CatalogSelectionStore.setSelection(listId: id);

    // ✅ ONLY: catalog endpoint
    _log('GET /mall/catalog/$id');
    final dto = await _catalogRepo.fetchCatalogByListId(id);

    _log(
      'catalog ok '
      'listId=${dto.list.id} '
      'inventoryId="${dto.list.inventoryId.trim()}" '
      'list.tbId="${(dto.list.tokenBlueprintId).trim()}" '
      'inv.tbId="${(dto.inventory?.tokenBlueprintId ?? '').trim()}"',
    );

    // ✅ inventoryId を SelectionStore に流し込む
    final invId = dto.list.inventoryId.trim();
    if (invId.isNotEmpty) {
      CatalogSelectionStore.setSelection(
        listId: dto.list.id.trim().isNotEmpty ? dto.list.id.trim() : id,
        inventoryId: invId,
        // modelId/stockCount は触らない
      );
    }

    // ✅ NEW: listImages debug logs（拾えているかの可視化）
    _log(
      'listImages raw '
      'count=${dto.listImages.length} '
      'err="${(dto.listImagesError ?? '').trim()}"',
    );
    if (dto.listImages.isNotEmpty) {
      final sorted = [...dto.listImages]
        ..sort((a, b) {
          final ao = a.displayOrder ?? 1 << 30;
          final bo = b.displayOrder ?? 1 << 30;
          final c = ao.compareTo(bo);
          if (c != 0) return c;
          return a.id.compareTo(b.id);
        });

      for (var idx = 0; idx < sorted.length; idx++) {
        final it = sorted[idx];
        _log(
          'listImages[$idx] '
          'id="${it.id}" '
          'order=${it.displayOrder ?? 'null'} '
          'url.len=${it.url.length} '
          'file="${it.fileName}" '
          'size=${it.size?.toString() ?? 'null'} '
          'objectPath="${it.objectPath}"',
        );
      }

      // 同一URL/同一objectPathの重複チェック（表示が1枚に見える原因切り分け用）
      final urlSet = <String>{};
      var dupUrl = 0;
      for (final it in dto.listImages) {
        final u = it.url.trim();
        if (u.isEmpty) continue;
        if (!urlSet.add(u)) dupUrl++;
      }
      if (dupUrl > 0) {
        _log('listImages warning: duplicated url count=$dupUrl');
      }

      final pathSet = <String>{};
      var dupPath = 0;
      for (final it in dto.listImages) {
        final p = it.objectPath.trim();
        if (p.isEmpty) continue;
        if (!pathSet.add(p)) dupPath++;
      }
      if (dupPath > 0) {
        _log('listImages warning: duplicated objectPath count=$dupPath');
      }
    }

    // ✅ tokenBlueprintId resolve (inventory優先 → list fallback)
    final resolvedTbId =
        (dto.inventory?.tokenBlueprintId ?? dto.list.tokenBlueprintId).trim();
    _log('resolved tokenBlueprintId="$resolvedTbId"');

    // ✅ token patch
    final token = await _token.load(resolvedTokenBlueprintId: resolvedTbId);

    // ✅ product
    final pbId = (dto.inventory?.productBlueprintId ?? '').trim();
    final prod = await _product.load(
      productBlueprintId: pbId,
      initial: dto.productBlueprint,
      initialError: dto.productBlueprintError,
    );

    // ✅ models（catalog DTO優先 / 無ければ /mall/models で補完）
    // ✅ productBlueprint.modelRefs があればそれを優先して並び替える
    final modelsRes = await _inventory.loadModels(
      invRepo: _invRepo,
      productBlueprintId: pbId,
      initial: dto.modelVariations,
      initialError: dto.modelVariationsError,
      modelRefs: prod.productBlueprint?.modelRefs,
    );

    return _buildState(
      list: dto.list,
      inventory: dto.inventory,
      inventoryError: dto.inventoryError,
      productBlueprint: prod.productBlueprint,
      productBlueprintError: prod.productBlueprintError,
      modelVariations: modelsRes.models,
      modelVariationsError: modelsRes.error,
      tokenBlueprintPatch: token.patch,
      tokenBlueprintError: token.error,
      resolvedTokenBlueprintId: resolvedTbId,
      displayOrderByModelId: modelsRes.displayOrderByModelId,

      // ✅ NEW: listImages
      listImages: dto.listImages,
      listImagesError: dto.listImagesError,
    );
  }

  CatalogState _buildState({
    required MallListItem list,
    required MallInventoryResponse? inventory,
    required String? inventoryError,
    required MallProductBlueprintResponse? productBlueprint,
    required String? productBlueprintError,
    required List<MallModelVariationDTO>? modelVariations,
    required String? modelVariationsError,
    required TokenBlueprintPatch? tokenBlueprintPatch,
    required String? tokenBlueprintError,
    required String resolvedTokenBlueprintId,
    required Map<String, int> displayOrderByModelId,

    // ✅ NEW
    required List<CatalogListImage> listImages,
    required String? listImagesError,
  }) {
    final imageUrl = list.image.trim();
    final hasImage = imageUrl.isNotEmpty;

    final priceText = _priceText(list.prices);

    // ✅ productBlueprintId は inventory 優先（list には基本無い/信頼しない方針）
    final pbId = (inventory?.productBlueprintId ?? '').trim();

    // ✅ tokenBlueprintId は resolved
    final tbId = resolvedTokenBlueprintId.trim();

    // ✅ inventory計算（モデル一覧をベースに stock を追記）
    final invComputed = _inventory.compute(
      inventory: inventory,
      modelVariations: modelVariations,
      prices: list.prices,
      displayOrderByModelId: displayOrderByModelId,
    );

    // ✅ measurements table（サイズ×採寸キー）
    final measTable = _measurement.compute(models: modelVariations);

    final tokenIconUrl = (tokenBlueprintPatch?.iconUrl ?? '').trim();

    return CatalogState(
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
      tokenBlueprintId: tbId,
      totalStock: invComputed.totalStock,
      modelStockRows: invComputed.modelStockRows,
      tokenBlueprintPatch: tokenBlueprintPatch,
      tokenBlueprintError: _asNonEmptyString(tokenBlueprintError),
      tokenIconUrlEncoded: tokenIconUrl.isNotEmpty
          ? _safeUrl(tokenIconUrl)
          : null,
      measurementTable: measTable,

      // ✅ NEW
      listImages: listImages,
      listImagesError: _asNonEmptyString(listImagesError),
    );
  }

  static String? _asNonEmptyString(String? v) {
    final s = (v ?? '').trim();
    return s.isEmpty ? null : s;
  }

  static String _safeUrl(String raw) => Uri.encodeFull(raw.trim());

  static String _priceText(List<MallListPriceRow> rows) {
    if (rows.isEmpty) return '';
    final prices = rows.map((e) => e.price).toList()..sort();
    final min = prices.first;
    final max = prices.last;
    if (min == max) return '¥$min';
    return '¥$min 〜 ¥$max';
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
    required this.measurementTable,

    // ✅ NEW
    required this.listImages,
    required this.listImagesError,
  });

  final MallListItem list;

  final String priceText;

  final String imageUrl;
  final String imageUrlEncoded;
  final bool hasImage;

  final MallInventoryResponse? inventory;
  final String? inventoryError;

  final MallProductBlueprintResponse? productBlueprint;
  final String? productBlueprintError;

  final List<MallModelVariationDTO>? modelVariations;
  final String? modelVariationsError;

  final String productBlueprintId;
  final String tokenBlueprintId;
  final int? totalStock;

  final List<CatalogModelStockRow> modelStockRows;

  final TokenBlueprintPatch? tokenBlueprintPatch;
  final String? tokenBlueprintError;
  final String? tokenIconUrlEncoded;

  // ✅ measurements table
  final CatalogMeasurementTable measurementTable;

  // ✅ NEW: list images for carousel
  final List<CatalogListImage> listImages;
  final String? listImagesError;
}

// ============================================================
// CatalogRepositoryHttp + DTO (matches backend MallCatalogDTO)
// ============================================================

class MallCatalogDTO {
  const MallCatalogDTO({
    required this.list,
    required this.inventory,
    required this.inventoryError,
    required this.productBlueprint,
    required this.productBlueprintError,
    required this.modelVariations,
    required this.modelVariationsError,

    // ✅ NEW
    required this.listImages,
    required this.listImagesError,
  });

  final MallListItem list;

  final MallInventoryResponse? inventory;
  final String? inventoryError;

  final MallProductBlueprintResponse? productBlueprint;
  final String? productBlueprintError;

  final List<MallModelVariationDTO>? modelVariations;
  final String? modelVariationsError;

  // ✅ NEW
  final List<CatalogListImage> listImages;
  final String? listImagesError;

  static String? _asNonEmptyString(dynamic v) {
    final s = (v ?? '').toString().trim();
    return s.isEmpty ? null : s;
  }

  factory MallCatalogDTO.fromJson(Map<String, dynamic> json) {
    final listJson =
        (json['list'] as Map?)?.cast<String, dynamic>() ?? const {};
    final invJson = (json['inventory'] as Map?)?.cast<String, dynamic>();
    final pbJson = (json['productBlueprint'] as Map?)?.cast<String, dynamic>();
    final mvJson = json['modelVariations'];

    // ✅ NEW: listImages
    final liJson = json['listImages'];
    final listImages = (liJson is List)
        ? liJson
              .whereType<Map>()
              .map((e) => CatalogListImage.fromJson(e.cast<String, dynamic>()))
              .toList()
        : <CatalogListImage>[];

    return MallCatalogDTO(
      list: MallListItem.fromJson(listJson),
      inventory: invJson != null
          ? MallInventoryResponse.fromJson(invJson)
          : null,
      inventoryError: _asNonEmptyString(json['inventoryError']),
      productBlueprint: pbJson != null
          ? MallProductBlueprintResponse.fromJson(pbJson)
          : null,
      productBlueprintError: _asNonEmptyString(json['productBlueprintError']),
      modelVariations: (mvJson is List)
          ? mvJson
                .whereType<Map>()
                .map(
                  (e) =>
                      MallModelVariationDTO.fromJson(e.cast<String, dynamic>()),
                )
                .toList()
          : null,
      modelVariationsError: _asNonEmptyString(json['modelVariationsError']),

      // ✅ NEW
      listImages: listImages,
      listImagesError: _asNonEmptyString(json['listImagesError']),
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

  /// ✅ app/config/api_base.dart の解決ロジックを使う（重複排除）
  static Uri _buildUri(String path) {
    final base = resolveApiBase().replaceAll(RegExp(r'\/+$'), '');
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$base$p');
  }

  Future<MallCatalogDTO> fetchCatalogByListId(String listId) async {
    final id = listId.trim();
    if (id.isEmpty) {
      throw Exception('catalog: listId is empty');
    }

    final uri = _buildUri('/mall/catalog/$id');
    final res = await _client.get(uri, headers: {'accept': 'application/json'});

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw Exception('catalog: http ${res.statusCode} body=${res.body}');
    }

    final jsonObj = jsonDecode(res.body);
    if (jsonObj is! Map) {
      throw Exception('catalog: invalid json (not an object)');
    }

    // wrapper 吸収: {data:{...}} を許容
    final root = jsonObj.cast<String, dynamic>();
    final data = (root['data'] is Map)
        ? (root['data'] as Map).cast<String, dynamic>()
        : root;

    return MallCatalogDTO.fromJson(data);
  }
}
