// frontend/inspector/lib/screens/inspection_detail/inspection_detail_actions.dart
import 'package:flutter/material.dart';

import '../../models/inspector_product_detail.dart';
import '../../services/product_api.dart';
import 'utils/inspection_formatters.dart';

class InspectionDetailActions {
  InspectionDetailActions();

  Future<InspectorProductDetail> fetchDetail(String productId) {
    return ProductApi.fetchInspectorDetail(productId);
  }

  Future<void> submitResult({
    required BuildContext context,
    required VoidCallback setSubmittingTrue,
    required VoidCallback setSubmittingFalse,
    required Future<void> Function() reload,
    required InspectorProductDetail detail,
    required String result, // 'passed', 'failed', or 'notManufactured'
    required bool submitting,
  }) async {
    if (submitting) return;

    if (result != 'passed' &&
        result != 'failed' &&
        result != 'notManufactured') {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('送信できる検品結果ではありません')));
      return;
    }

    setSubmittingTrue();

    try {
      await ProductApi.updateInspectionBatch(
        productionId: detail.productionId,
        productId: detail.productId,
        inspectionResult: result,
      );

      if (!context.mounted) return;

      final resultLabel = formatInspectionResultLabel(result);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('検品結果を送信しました（$resultLabel）')));

      await reload();
    } catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('検品結果の送信に失敗しました: $e')));
    } finally {
      if (context.mounted) {
        setSubmittingFalse();
      }
    }
  }

  Future<void> completeInspection({
    required BuildContext context,
    required VoidCallback setSubmittingTrue,
    required VoidCallback setSubmittingFalse,
    required Future<void> Function() reload,
    required String productionId,
    required bool submitting,
  }) async {
    if (submitting) return;
    setSubmittingTrue();

    debugPrint(
      '[InspectionDetailScreen] completeInspection requested: productionId=$productionId',
    );

    try {
      await ProductApi.completeInspection(productionId: productionId);
      if (!context.mounted) return;

      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('検品を完了しました')));

      await reload();
    } catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('検品完了処理に失敗しました: $e')));
    } finally {
      if (context.mounted) {
        setSubmittingFalse();
      }
    }
  }
}
