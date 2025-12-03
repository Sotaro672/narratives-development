// ignore_for_file: deprecated_member_use, avoid_web_libraries_in_flutter
import 'package:flutter/material.dart';
import 'dart:html' as html; // Chrome コンソール出力用

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

  // 検査結果ラベル変換
  String formatResult(String? raw) {
    switch (raw) {
      case 'passed':
        return '合格';
      case 'failed':
        return '不合格';
      case 'notYet':
      case null:
      default:
        return '未検査';
    }
  }

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

  Future<void> _submitResult(
    InspectorProductDetail detail,
    String result,
  ) async {
    if (_submitting) return;
    setState(() {
      _submitting = true;
    });

    try {
      // products テーブルの検品結果更新
      await ProductApi.submitInspection(
        productId: detail.productId,
        result: result,
      );

      // inspections テーブルの検品結果更新
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

  Future<void> _completeInspection(String productionId) async {
    if (_submitting) return;
    setState(() {
      _submitting = true;
    });

    // ★ 検品完了リクエスト時に渡す情報をログ出力
    html.window.console.log(
      '[InspectionDetailScreen] completeInspection requested: '
      'productionId=$productionId',
    );

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

  void _continueInspection() {
    Navigator.of(context).pop();
  }

  // ----------------------------------------------------------
  // モデル情報カード（productId / modelId 非表示）
  // ----------------------------------------------------------
  Widget _buildModelCard(InspectorProductDetail detail) {
    final entries = detail.measurements.entries.toList()
      ..sort((a, b) => a.key.compareTo(b.key));

    // rgb(int) → Flutter Color 変換（RGB だけの場合も Alpha=0xFF を補う）
    final colorInt = (() {
      final v = detail.color.rgb;
      // 上位 8bit が 0 の場合は alpha が無い想定なので 0xFF を付与
      if ((v & 0xFF000000) == 0) {
        return 0xFF000000 | v;
      }
      return v;
    })();

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
                    color: Color(colorInt),
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

  // ----------------------------------------------------------
  // 商品設計情報カード
  // ----------------------------------------------------------
  Widget _buildProductBlueprintCard(InspectorProductDetail detail) {
    final bp = detail.productBlueprint;
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              '商品設計情報',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            Text('商品名: ${bp.productName}'),
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

  // ----------------------------------------------------------
  // 検品履歴: inspections テーブルの内容表示
  //  modelNumber + 生産量(quantity) + 合格数(totalPassed) + 各 product 行
  // ----------------------------------------------------------
  Widget _buildInspectionList(InspectorProductDetail detail) {
    final inspections = detail.inspections;
    if (inspections.isEmpty) {
      return const Padding(
        padding: EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        child: Text('検品履歴はまだありません。'),
      );
    }

    // ★ 生産量 = レコード数
    final int quantity = inspections.length;

    // ★ 合格数 = inspectionResult == 'passed' の件数
    final int totalPassed = inspections
        .where((r) => r.inspectionResult == 'passed')
        .length;

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              '検品履歴',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            // ★ 上部サマリー: modelNumber / 生産量 / 合格数
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('modelNumber: ${detail.modelNumber}'),
                  Text('生産量: $quantity'),
                  Text('合格数: $totalPassed'),
                ],
              ),
            ),
            const Divider(height: 1),
            const SizedBox(height: 8),
            ListView.separated(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              itemCount: inspections.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (context, index) {
                final item = inspections[index];
                final resultLabel = formatResult(item.inspectionResult);
                final modelNumber = item.modelNumber ?? '';

                return ListTile(
                  dense: true,
                  contentPadding: EdgeInsets.zero,
                  title: Text(
                    modelNumber.isNotEmpty
                        ? 'productId: ${item.productId} / modelNumber: $modelNumber'
                        : 'productId: ${item.productId}',
                    style: const TextStyle(fontSize: 13),
                  ),
                  subtitle: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('検査結果: $resultLabel'),
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

  // ----------------------------------------------------------
  // 合否ボタン + 検品履歴 + 続ける / 完了ボタン
  // ----------------------------------------------------------
  Widget _buildActionSection(InspectorProductDetail detail) {
    final nowStatus = detail.inspectionResult;
    final nowStatusLabel = nowStatus.isEmpty ? '未検査' : formatResult(nowStatus);

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (nowStatus.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Text(
                '現在の検品ステータス: $nowStatusLabel',
                style: const TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.bold,
                ),
              ),
            ),

          // 上段: 合格 / 不合格
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

          // 中段: 検品履歴（inspections 一覧）
          const SizedBox(height: 16),
          _buildInspectionList(detail),
          const SizedBox(height: 16),

          // 下段: 検品を続ける / 完了する
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

          return RefreshIndicator(
            onRefresh: _reload,
            child: ListView(
              children: [
                _buildModelCard(detail),
                _buildProductBlueprintCard(detail),
                _buildActionSection(detail),
                const SizedBox(height: 16),
              ],
            ),
          );
        },
      ),
    );
  }
}
