// frontend\mall\lib\app\shell\presentation\state\catalog_selection_store.dart
import 'package:flutter/foundation.dart';

@immutable
class CatalogSelection {
  const CatalogSelection({
    this.listId = '',
    this.inventoryId = '',
    this.modelId,
    this.stockCount,
  });

  final String listId;

  /// ✅ catalog の実体（在庫）を特定するID
  /// - cart へ渡すために保持する
  final String inventoryId;

  final String? modelId;

  /// ✅ 選択された model の在庫数（未確定/未選択なら null）
  final int? stockCount;
}

class CatalogSelectionStore {
  CatalogSelectionStore._();

  static final ValueNotifier<CatalogSelection> notifier = ValueNotifier(
    const CatalogSelection(),
  );

  static void setSelection({
    required String listId,
    String? inventoryId,
    String? modelId,
    int? stockCount,
  }) {
    final lid = listId.trim();
    final prev = notifier.value;

    // ✅ inventoryId が未指定(null)なら「保持」する
    // ただし list が変わった場合は、未指定ならリセットする
    final String inv = (() {
      final nextInv = inventoryId?.trim();
      if (nextInv != null) return nextInv;

      // inventoryId を渡さずに呼ばれた場合
      if (prev.listId.trim() == lid) {
        return prev.inventoryId.trim();
      }
      return '';
    })();

    final midRaw = (modelId ?? '').trim();
    final int? sc = stockCount;

    // ✅ 追加（原因特定用）
    // ignore: avoid_print
    print(
      '[CatalogSelectionStore] list="$lid" inv="$inv" model="$midRaw" stock="$sc" (prevInv="${prev.inventoryId}")',
    );

    notifier.value = CatalogSelection(
      listId: lid,
      inventoryId: inv,
      modelId: midRaw.isEmpty ? null : midRaw,
      stockCount: midRaw.isEmpty ? null : sc,
    );
  }

  static void clearIfList(String listId) {
    if (notifier.value.listId.trim() == listId.trim()) {
      notifier.value = const CatalogSelection();
    }
  }
}
