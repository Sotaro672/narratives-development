// frontend/inspector/lib/screens/inspection_detail_screen.dart
import 'package:flutter/material.dart';
import 'dart:developer' as developer;

import '../services/product_api.dart';

class InspectionDetailScreen extends StatefulWidget {
  final String productId;

  const InspectionDetailScreen({super.key, required this.productId});

  @override
  State<InspectionDetailScreen> createState() => _InspectionDetailScreenState();
}

class _InspectionDetailScreenState extends State<InspectionDetailScreen> {
  late Future<InspectorProductDetail> _futureDetail;
  bool _submitting = false;

  @override
  void initState() {
    super.initState();
    _futureDetail = ProductApi.fetchInspectorDetail(widget.productId);
  }

  Future<void> _reload() async {
    setState(() {
      _futureDetail = ProductApi.fetchInspectorDetail(widget.productId);
    });
  }

  /// 合否（passed / failed）を送信
  ///
  /// - products テーブル: ProductApi.submitInspection()
  /// - inspections テーブル: ProductApi.updateInspectionBatch()
  Future<void> _submitResult(
    InspectorProductDetail detail,
    String result,
  ) async {
    if (_submitting) return;
    setState(() {
      _submitting = true;
    });

    try {
      // 1) products テーブル側を更新（/products/{productId}）
      await ProductApi.submitInspection(
        productId: detail.productId,
        result: result,
      );

      // 2) inspections テーブル側を更新（/products/inspections）
      await ProductApi.updateInspectionBatch(
        productionId: detail.productionId,
        productId: detail.productId,
        inspectionResult: result == 'passed' ? 'passed' : 'failed',
      );

      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('検品結果を送信しました（$result）')));

      await _reload();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('検品結果の送信に失敗しました: $e')));
    } finally {
      if (mounted) {
        setState(() {
          _submitting = false;
        });
      }
    }
  }

  /// 検品を完了する
  ///
  /// - Go 側ロジック前提:
  ///   - 該当 productionId の inspections のうち inspectionResult == "notYet" を
  ///     "notManufactured" に更新
  ///   - status を "inspected" に更新
  Future<void> _completeInspection(String productionId) async {
    if (_submitting) return;
    setState(() {
      _submitting = true;
    });

    try {
      await ProductApi.completeInspection(productionId: productionId);
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('検品を完了しました')));
      await _reload();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('検品完了処理に失敗しました: $e')));
    } finally {
      if (mounted) {
        setState(() {
          _submitting = false;
        });
      }
    }
  }

  /// 検品を続ける（カメラ画面に戻る）
  void _continueInspection() {
    Navigator.of(context).pop(); // スタックを 1 つ戻る → スキャナー画面へ
  }

  Widget _buildModelCard(InspectorProductDetail detail) {
    final entries = detail.measurements.entries.toList()
      ..sort((a, b) => a.key.compareTo(b.key));

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'モデル情報',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            Text('productId: ${detail.productId}'),
            Text('modelId: ${detail.modelId}'),
            Text('modelNumber: ${detail.modelNumber}'),
            if (detail.size.isNotEmpty) Text('サイズ: ${detail.size}'),
            const SizedBox(height: 8),
            Row(
              children: [
                const Text('カラー:'),
                const SizedBox(width: 8),
                Container(
                  width: 18,
                  height: 18,
                  decoration: BoxDecoration(
                    color: Color(detail.color.rgb),
                    borderRadius: BorderRadius.circular(4),
                    border: Border.all(color: Colors.grey.shade400),
                  ),
                ),
                const SizedBox(width: 8),
                Text(detail.color.name ?? ''),
              ],
            ),
            if (entries.isNotEmpty) ...[
              const SizedBox(height: 8),
              const Text('採寸値', style: TextStyle(fontWeight: FontWeight.bold)),
              const SizedBox(height: 4),
              Wrap(
                spacing: 8,
                runSpacing: 4,
                children: entries
                    .map((e) => Chip(label: Text('${e.key}: ${e.value}')))
                    .toList(),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildProductBlueprintCard(InspectorProductDetail detail) {
    final bp = detail.blueprint;
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              '商品設計情報 (ProductBlueprint)',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            Text('productBlueprintId: ${detail.productBlueprintId}'),
            Text('商品名: ${bp.productName}'),
            // ▼ ラベル＆プロパティ名を変更
            Text('ブランド名: ${bp.brandName}'),
            Text('会社名: ${bp.companyName}'),
            Text('アイテム種別: ${bp.itemType}'),
            if (bp.fit.isNotEmpty) Text('フィット: ${bp.fit}'),
            if (bp.material.isNotEmpty) Text('素材: ${bp.material}'),
            Text('重さ: ${bp.weight}'),
            const SizedBox(height: 8),
            if (bp.qualityAssurance.isNotEmpty) ...[
              const Text(
                '品質表示・注意事項',
                style: TextStyle(fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 4),
              Wrap(
                spacing: 8,
                runSpacing: 4,
                children: bp.qualityAssurance
                    .map((q) => Chip(label: Text(q)))
                    .toList(),
              ),
            ],
            const SizedBox(height: 8),
            Text('タグ種別: ${bp.productIdTagType}'),
            if (bp.assigneeId.isNotEmpty) Text('担当者ID: ${bp.assigneeId}'),
          ],
        ),
      ),
    );
  }

  Widget _buildInspectionList(InspectorProductDetail detail) {
    final inspections = detail.inspections;
    if (inspections.isEmpty) {
      return const Padding(
        padding: EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        child: Text('検品履歴はまだありません。'),
      );
    }

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              '検品結果一覧 (inspections)',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            ListView.separated(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              itemCount: inspections.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (context, index) {
                final item = inspections[index];
                return ListTile(
                  dense: true,
                  contentPadding: EdgeInsets.zero,
                  title: Text(
                    'productId: ${item.productId}',
                    style: const TextStyle(fontSize: 13),
                  ),
                  subtitle: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('結果: ${item.inspectionResult ?? '未検品'}'),
                      if (item.inspectedBy != null &&
                          item.inspectedBy!.isNotEmpty)
                        Text('検査者: ${item.inspectedBy}'),
                      if (item.inspectedAt != null)
                        Text('検査日時: ${item.inspectedAt}'),
                    ],
                  ),
                );
              },
            ),
          ],
        ),
      ),
    );
  }

  /// 合否ボタン + 「検品を続ける」「検品を完了する」
  Widget _buildActionButtons(InspectorProductDetail detail) {
    final nowStatus = detail.inspectionResult;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (nowStatus.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Text(
                '現在の検品ステータス: $nowStatus',
                style: const TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.bold,
                ),
              ),
            ),
          // 合否ボタン
          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: _submitting
                      ? null
                      : () => _submitResult(detail, 'failed'),
                  child: const Text('不合格'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: ElevatedButton(
                  onPressed: _submitting
                      ? null
                      : () => _submitResult(detail, 'passed'),
                  child: const Text('合格'),
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          // 検品を続ける / 検品を完了する
          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: _submitting ? null : _continueInspection,
                  child: const Text('検品を続ける'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: FilledButton(
                  onPressed: _submitting
                      ? null
                      : () => _completeInspection(detail.productionId),
                  child: const Text('検品を完了する'),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('検品詳細: ${widget.productId}')),
      body: FutureBuilder<InspectorProductDetail>(
        future: _futureDetail,
        builder: (context, snapshot) {
          if (snapshot.connectionState == ConnectionState.waiting &&
              !snapshot.hasData) {
            return const Center(child: CircularProgressIndicator());
          }

          if (snapshot.hasError) {
            return Center(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Text('データ取得に失敗しました: ${snapshot.error}'),
              ),
            );
          }

          final detail = snapshot.data!;

          // ▼ 取得したデータを表すログ（print → developer.log に変更）
          developer.log(
            '[InspectionDetailScreen] loaded detail: '
            'productId=${detail.productId}, '
            'modelId=${detail.modelId}, '
            'productionId=${detail.productionId}, '
            'modelNumber=${detail.modelNumber}, '
            'size=${detail.size}, '
            'brandName=${detail.blueprint.brandName}, '
            'companyName=${detail.blueprint.companyName}, '
            'inspectionResult=${detail.inspectionResult}, '
            'inspectionsCount=${detail.inspections.length}',
            name: 'InspectionDetailScreen',
          );

          return RefreshIndicator(
            onRefresh: _reload,
            child: ListView(
              children: [
                _buildModelCard(detail),
                _buildProductBlueprintCard(detail),
                _buildActionButtons(detail),
                _buildInspectionList(detail),
                const SizedBox(height: 16),
              ],
            ),
          );
        },
      ),
    );
  }
}
