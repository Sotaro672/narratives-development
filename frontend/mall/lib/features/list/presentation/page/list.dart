// frontend\mall\lib\features\list\presentation\page\list.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../infrastructure/list_repository_http.dart';

class HomePage extends StatefulWidget {
  const HomePage({super.key});

  static const String pageName = 'home';

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  late final ListRepositoryHttp _repo;
  late Future<MallListIndexResponse> _future;

  @override
  void initState() {
    super.initState();
    _repo = ListRepositoryHttp();
    _future = _repo.fetchLists(page: 1, perPage: 20);
  }

  @override
  void dispose() {
    _repo.dispose();
    super.dispose();
  }

  Future<void> _reload() async {
    setState(() {
      _future = _repo.fetchLists(page: 1, perPage: 20);
    });
  }

  @override
  Widget build(BuildContext context) {
    // ✅ Scaffold は AppShell 側で持つ前提
    //    ここでは「中身」だけを返す
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // ✅ 見出し行（Mall + refresh）を削除して、ヘッダーっぽい行スペースを消す

        // ✅ 本文（一覧）
        FutureBuilder<MallListIndexResponse>(
          future: _future,
          builder: (context, snap) {
            if (snap.connectionState == ConnectionState.waiting) {
              return const Padding(
                padding: EdgeInsets.only(top: 24),
                child: Center(child: CircularProgressIndicator()),
              );
            }
            if (snap.hasError) {
              return _ErrorView(error: snap.error, onRetry: _reload);
            }

            final data = snap.data!;
            final items = data.items;

            if (items.isEmpty) {
              return const Padding(
                padding: EdgeInsets.only(top: 24),
                child: Center(child: Text('No listings')),
              );
            }

            // ✅ AppShell(AppMain) がスクロールを持つ前提なので、
            //    ここでは shrinkWrap で ListView を使う
            return RefreshIndicator(
              onRefresh: _reload,
              child: ListView.builder(
                shrinkWrap: true,
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.only(bottom: 12),
                itemCount: items.length,
                itemBuilder: (context, i) {
                  return _ListCard(item: items[i]);
                },
              ),
            );
          },
        ),
      ],
    );
  }
}

class _ListCard extends StatelessWidget {
  const _ListCard({required this.item});

  final MallListItem item;

  String _safeUrl(String raw) => Uri.encodeFull(raw.trim());

  String _priceText(List<MallListPriceRow> rows) {
    if (rows.isEmpty) return '';
    final prices = rows.map((e) => e.price).toList()..sort();
    final min = prices.first;
    final max = prices.last;
    if (min == max) return '¥$min';
    return '¥$min 〜 ¥$max';
  }

  @override
  Widget build(BuildContext context) {
    final imageUrl = item.image.trim();
    final hasImage = imageUrl.isNotEmpty;
    final price = _priceText(item.prices);

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: () {
          // ✅ go_router で遷移（ShellRoute(AppShell) が必ず効く）
          context.pushNamed(
            'catalog',
            pathParameters: {'listId': item.id},
            extra: item, // Catalog 側で initialItem として使いたい場合
          );
        },
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            // -------- image --------
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
                        return const Center(child: CircularProgressIndicator());
                      },
                    )
                  : const _ImageFallback(label: 'no image'),
            ),

            // -------- content --------
            Padding(
              padding: const EdgeInsets.all(12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    item.title.isNotEmpty ? item.title : '(no title)',
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  const SizedBox(height: 6),
                  if (item.description.trim().isNotEmpty)
                    Text(
                      item.description.trim(),
                      maxLines: 3,
                      overflow: TextOverflow.ellipsis,
                      style: Theme.of(context).textTheme.bodyMedium,
                    ),
                  const SizedBox(height: 10),
                  Row(
                    children: [
                      if (price.isNotEmpty)
                        Text(
                          price,
                          style: Theme.of(context).textTheme.titleSmall,
                        ),
                      const Spacer(),
                      Text(
                        'items: ${item.prices.length}',
                        style: Theme.of(context).textTheme.labelMedium,
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
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
