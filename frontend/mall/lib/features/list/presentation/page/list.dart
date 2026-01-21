// frontend\mall\lib\features\list\presentation\page\list.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/routing/routes.dart';
import '../../../../app/shell/presentation/components/app_fixed_extent_grid.dart';
import '../../infrastructure/list_repository_http.dart';

class HomePage extends StatefulWidget {
  const HomePage({super.key});

  static const String pageName = 'list';

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  late final ListRepositoryHttp _repo;
  late Future<MallListIndexResponse> _future;

  // ✅ route change hook: listen to routeInformationProvider (ChangeNotifier)
  GoRouteInformationProvider? _rip;
  String? _lastPath;
  bool _routeReloadGuard = false;

  @override
  void initState() {
    super.initState();
    _repo = ListRepositoryHttp();
    _future = _repo.fetchLists(page: 1, perPage: 20);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();

    final router = GoRouter.of(context);
    final rip = router.routeInformationProvider;

    if (_rip != rip) {
      _rip?.removeListener(_onRouteChanged);
      _rip = rip;
      _lastPath = rip.value.uri.path;
      _rip!.addListener(_onRouteChanged);
    }
  }

  void _onRouteChanged() {
    final rip = _rip;
    if (rip == null) {
      return;
    }

    final path = rip.value.uri.path;
    final prev = _lastPath;
    _lastPath = path;

    final isHome = path == AppRoutePath.home;
    final wasHome = prev == AppRoutePath.home;

    if (isHome && !wasHome) {
      if (_routeReloadGuard) {
        return;
      }
      _routeReloadGuard = true;

      Future.microtask(() async {
        if (!mounted) {
          return;
        }
        try {
          await _reload();
        } finally {
          _routeReloadGuard = false;
        }
      });
    }
  }

  @override
  void dispose() {
    _rip?.removeListener(_onRouteChanged);
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
    // ✅ Scaffold は AppShell 側で持つ前提（ここでは中身だけ）
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
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

            // ✅ 期待値: list item を「3個1列（3列グリッド）」で表示
            return RefreshIndicator(
              onRefresh: _reload,
              child: AppFixedExtentGrid(
                crossAxisCount: 3,
                spacing: 10,
                childAspectRatio: 0.82, // ✅ token と同じ（スマホ幅前提）
                extraTextLines: 1, // ✅ token と同じ（最低限の増分）
                shrinkWrap: true,
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.only(bottom: 12),
                itemCount: items.length,
                itemBuilder: (context, i) {
                  return _ListGridCard(item: items[i]);
                },
              ),
            );
          },
        ),
      ],
    );
  }
}

class _ListGridCard extends StatelessWidget {
  const _ListGridCard({required this.item});

  final MallListItem item;

  String _safeUrl(String raw) {
    final s = raw.trim();
    final uri = Uri.tryParse(s);
    return (uri ?? Uri()).toString();
  }

  String _priceText(List<MallListPriceRow> rows) {
    if (rows.isEmpty) {
      return '';
    }
    final prices = rows.map((e) => e.price).toList()..sort();
    final min = prices.first;
    final max = prices.last;
    if (min == max) {
      return '¥$min';
    }
    return '¥$min 〜 ¥$max';
  }

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    final imageUrl = item.image.trim();
    final hasImage = imageUrl.isNotEmpty;

    final title = item.title.isNotEmpty ? item.title : '(no title)';
    final price = _priceText(item.prices);

    // ✅ token_card と同様に「高さ固定グリッド」内で崩れないよう、
    //    LayoutBuilder で残り高さを画像に割り当てる
    return Card(
      elevation: 0,
      color: cs.surfaceContainerHighest,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      clipBehavior: Clip.antiAlias,
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: () {
            context.pushNamed(
              AppRouteName.catalog,
              pathParameters: {'listId': item.id},
              extra: item,
            );
          },
          child: Padding(
            padding: const EdgeInsets.all(10),
            child: LayoutBuilder(
              builder: (context, constraints) {
                final titleStyle = Theme.of(context).textTheme.titleSmall
                    ?.copyWith(
                      color: cs.onSurface,
                      fontWeight: FontWeight.w700,
                      fontSize: 12.5,
                      height: 1.1,
                    );

                final subStyle = Theme.of(context).textTheme.bodySmall
                    ?.copyWith(
                      color: cs.onSurfaceVariant,
                      fontWeight: FontWeight.w700,
                      fontSize: 11.5,
                      height: 1.1,
                    );

                final titleLineH =
                    ((titleStyle?.fontSize ?? 12.5) *
                    (titleStyle?.height ?? 1.1));
                final subLineH =
                    ((subStyle?.fontSize ?? 11.5) * (subStyle?.height ?? 1.1));

                const gap1 = 6.0;
                const gap2 = 6.0;

                // title(1行) + gap + price(1行) + gap
                final reservedTextHeight = titleLineH + gap1 + subLineH + gap2;

                final imageH = (constraints.maxHeight - reservedTextHeight)
                    .clamp(56.0, 9999.0);

                return Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    SizedBox(
                      height: imageH,
                      child: ClipRRect(
                        borderRadius: BorderRadius.circular(10),
                        child: Container(
                          color: cs.surface,
                          child: hasImage
                              ? Image.network(
                                  _safeUrl(imageUrl),
                                  fit: BoxFit.cover,
                                  loadingBuilder:
                                      (context, child, loadingProgress) {
                                        if (loadingProgress == null) {
                                          return child;
                                        }
                                        return const Center(
                                          child: SizedBox(
                                            width: 20,
                                            height: 20,
                                            child: CircularProgressIndicator(
                                              strokeWidth: 2,
                                            ),
                                          ),
                                        );
                                      },
                                  errorBuilder: (context, error, stackTrace) {
                                    return Center(
                                      child: Text(
                                        'no image',
                                        style: Theme.of(context)
                                            .textTheme
                                            .bodySmall
                                            ?.copyWith(
                                              color: cs.onSurfaceVariant,
                                              fontWeight: FontWeight.w600,
                                            ),
                                      ),
                                    );
                                  },
                                )
                              : Center(
                                  child: Text(
                                    'no image',
                                    style: Theme.of(context).textTheme.bodySmall
                                        ?.copyWith(
                                          color: cs.onSurfaceVariant,
                                          fontWeight: FontWeight.w600,
                                        ),
                                  ),
                                ),
                        ),
                      ),
                    ),
                    const SizedBox(height: gap2),
                    Text(
                      title,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: titleStyle,
                    ),
                    const SizedBox(height: gap1),
                    Text(
                      price.isEmpty ? '—' : price,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: subStyle,
                    ),
                  ],
                );
              },
            ),
          ),
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
