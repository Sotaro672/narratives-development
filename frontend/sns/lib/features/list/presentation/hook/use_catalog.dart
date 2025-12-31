// frontend/sns/lib/features/home/presentation/hook/use_catalog.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../../../inventory/infrastructure/inventory_repository_http.dart';
import '../../infrastructure/list_repository_http.dart';
import '../../../productBlueprint/infrastructure/product_blueprint_repository_http.dart';
import '../../../tokenBlueprint/infrastructure/token_blueprint_repository_http.dart';
import 'use_catalog_inventory.dart';
import 'use_catalog_product.dart';
import 'use_catalog_token.dart';

/// ✅ state/logic holder for CatalogPage
class UseCatalog {
  UseCatalog({http.Client? client})
    : _catalogRepo = CatalogRepositoryHttp(client: client),
      _listRepo = ListRepositoryHttp(),
      _invRepo = InventoryRepositoryHttp(),
      _inventory = const UseCatalogInventory(),
      _product = UseCatalogProduct(),
      _token = UseCatalogToken();

  final CatalogRepositoryHttp _catalogRepo;

  final ListRepositoryHttp _listRepo;
  final InventoryRepositoryHttp _invRepo;

  // ✅ inventory hook（モデル取得 + 計算）
  final UseCatalogInventory _inventory;

  // ✅ product / token
  final UseCatalogProduct _product;
  final UseCatalogToken _token;

  void dispose() {
    _catalogRepo.dispose();
    _listRepo.dispose();
    _invRepo.dispose();
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

      // ✅ models（catalog DTO優先 / 無ければ /sns/models で補完）
      final modelsRes = await _inventory.loadModels(
        invRepo: _invRepo,
        productBlueprintId: pbId,
        initial: dto.modelVariations,
        initialError: dto.modelVariationsError,
      );

      final state = _buildState(
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
      );

      return state;
    } catch (e) {
      _log('catalog endpoint failed -> fallback legacy. error=$e');
      return _loadLegacy(id);
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

    // inventory
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

    // ✅ tokenBlueprintId resolve
    final resolvedTbId = (inv?.tokenBlueprintId ?? list.tokenBlueprintId)
        .trim();
    _log('legacy resolved tokenBlueprintId="$resolvedTbId"');

    // ✅ product
    final prod = await _product.load(
      productBlueprintId: pbId,
      initial: null,
      initialError: null,
    );

    // ✅ models (/sns/models)
    final modelsRes = await _inventory.loadModels(
      invRepo: _invRepo,
      productBlueprintId: pbId,
      initial: null,
      initialError: null,
    );

    // ✅ token patch
    final token = await _token.load(resolvedTokenBlueprintId: resolvedTbId);

    return _buildState(
      list: list,
      inventory: inv,
      inventoryError: invErr,
      productBlueprint: prod.productBlueprint,
      productBlueprintError: prod.productBlueprintError,
      modelVariations: modelsRes.models,
      modelVariationsError: modelsRes.error,
      tokenBlueprintPatch: token.patch,
      tokenBlueprintError: token.error,
      resolvedTokenBlueprintId: resolvedTbId,
    );
  }

  CatalogState _buildState({
    required SnsListItem list,
    required SnsInventoryResponse? inventory,
    required String? inventoryError,
    required SnsProductBlueprintResponse? productBlueprint,
    required String? productBlueprintError,
    required List<SnsModelVariationDTO>? modelVariations,
    required String? modelVariationsError,
    required TokenBlueprintPatch? tokenBlueprintPatch,
    required String? tokenBlueprintError,
    required String resolvedTokenBlueprintId,
  }) {
    final imageUrl = list.image.trim();
    final hasImage = imageUrl.isNotEmpty;

    final priceText = _priceText(list.prices);

    // ✅ productBlueprintId は inventory 優先（list には基本無い/信頼しない方針）
    final pbId = (inventory?.productBlueprintId ?? '').trim();

    // ✅ tokenBlueprintId は resolved
    final tbId = resolvedTokenBlueprintId.trim();

    // ✅ inventory計算（モデル一覧をベースに stock を追記）
    // ✅ FIX: prices を渡す（missing_required_argument 対策）
    final invComputed = _inventory.compute(
      inventory: inventory,
      modelVariations: modelVariations,
      prices: list.prices,
    );

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
    );
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

  final List<SnsModelVariationDTO>? modelVariations;
  final String? modelVariationsError;

  final String productBlueprintId;
  final String tokenBlueprintId;
  final int? totalStock;

  final List<CatalogModelStockRow> modelStockRows;

  final TokenBlueprintPatch? tokenBlueprintPatch;
  final String? tokenBlueprintError;
  final String? tokenIconUrlEncoded;
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

  final List<SnsModelVariationDTO>? modelVariations;
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
                  (e) =>
                      SnsModelVariationDTO.fromJson(e.cast<String, dynamic>()),
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

  /// ✅ inventory_repository_http.dart の解決ロジックを使う（重複排除）
  static Uri _buildUri(String path) {
    // resolveSnsApiBase() is defined in inventory_repository_http.dart
    final base = resolveSnsApiBase().replaceAll(RegExp(r'\/+$'), '');
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
