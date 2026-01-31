// frontend/inspector/lib/screens/inspection_detail/inspection_detail_screen.dart
import 'package:flutter/material.dart';

import '../../models/inspector_product_detail.dart';
import 'inspection_detail_actions.dart';
import 'widgets/action_section.dart';
import 'widgets/model_card.dart';
import 'widgets/product_blueprint_card.dart';

class InspectionDetailScreen extends StatefulWidget {
  final String productId;

  const InspectionDetailScreen({super.key, required this.productId});

  @override
  State<InspectionDetailScreen> createState() => _InspectionDetailScreenState();
}

class _InspectionDetailScreenState extends State<InspectionDetailScreen> {
  final _actions = InspectionDetailActions();

  late Future<InspectorProductDetail> _futureDetail;
  bool _submitting = false;

  @override
  void initState() {
    super.initState();
    _futureDetail = _actions.fetchDetail(widget.productId);
  }

  Future<void> _reload() async {
    setState(() {
      _futureDetail = _actions.fetchDetail(widget.productId);
    });
  }

  Future<void> _onSubmitResult(InspectorProductDetail detail, String result) {
    return _actions.submitResult(
      context: context,
      setSubmittingTrue: () => setState(() => _submitting = true),
      setSubmittingFalse: () => setState(() => _submitting = false),
      reload: _reload,
      detail: detail,
      result: result,
      submitting: _submitting,
    );
  }

  Future<void> _onComplete(String productionId) {
    return _actions.completeInspection(
      context: context,
      setSubmittingTrue: () => setState(() => _submitting = true),
      setSubmittingFalse: () => setState(() => _submitting = false),
      reload: _reload,
      productionId: productionId,
      submitting: _submitting,
    );
  }

  void _continueInspection() {
    Navigator.of(context).pop();
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
                ModelCard(detail: detail),
                ProductBlueprintCard(detail: detail),
                ActionSection(
                  detail: detail,
                  submitting: _submitting,
                  onContinue: _continueInspection,
                  onSubmitResult: _onSubmitResult,
                  onComplete: _onComplete,
                ),
                const SizedBox(height: 16),
              ],
            ),
          );
        },
      ),
    );
  }
}
