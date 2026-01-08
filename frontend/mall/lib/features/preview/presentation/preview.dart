// frontend\mall\lib\features\preview\presentation\preview.dart
import 'package:flutter/material.dart';

/// Preview page (buyer-facing).
///
/// 現時点では「ルート追加のための受け皿」として、
/// avatarId / from を表示する最小実装にしています。
/// preview_query の表示ロジックは、このページに後から足していけます。
class PreviewPage extends StatefulWidget {
  const PreviewPage({super.key, required this.avatarId, this.from});

  final String avatarId;
  final String? from;

  @override
  State<PreviewPage> createState() => _PreviewPageState();
}

class _PreviewPageState extends State<PreviewPage> {
  String get _avatarId => widget.avatarId.trim();

  @override
  Widget build(BuildContext context) {
    final avatarId = _avatarId;
    final from = (widget.from ?? '').trim();

    if (avatarId.isEmpty) {
      return const Center(child: Text('avatarId is required'));
    }

    final t = Theme.of(context).textTheme;

    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(14),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Preview', style: t.titleMedium),
                  const SizedBox(height: 8),
                  Text('avatarId: $avatarId', style: t.bodySmall),
                  const SizedBox(height: 4),
                  Text(
                    'from: ${from.isEmpty ? '-' : from}',
                    style: t.bodySmall,
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 12),
          const Card(
            child: Padding(
              padding: EdgeInsets.all(14),
              child: Text('ここに preview_query の結果を表示します。'),
            ),
          ),
        ],
      ),
    );
  }
}
