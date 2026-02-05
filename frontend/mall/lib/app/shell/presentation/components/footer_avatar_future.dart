//frontend\mall\lib\app\shell\presentation\components\footer_avatar_future.dart
import 'package:flutter/material.dart';

import 'package:mall/features/avatar/infrastructure/avatar_api_client.dart';
import 'package:mall/features/avatar/presentation/model/me_avatar.dart';

/// 「/mall/me/avatar を叩いて MeAvatar を取る」責務だけを持つ。
/// ✅ build毎に Future を作らず、State で1回だけ保持する。
class AvatarProfileFuture extends StatefulWidget {
  const AvatarProfileFuture({super.key, required this.builder});

  final Widget Function(BuildContext context, MeAvatar? avatar) builder;

  @override
  State<AvatarProfileFuture> createState() => _AvatarProfileFutureState();
}

class _AvatarProfileFutureState extends State<AvatarProfileFuture> {
  late final Future<MeAvatar?> _future;

  @override
  void initState() {
    super.initState();
    _future = _fetchOnce();
  }

  Future<MeAvatar?> _fetchOnce() async {
    final api = AvatarApiClient();
    try {
      return await api.fetchMyAvatarProfile(); // => MeAvatar?
    } finally {
      api.dispose();
    }
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<MeAvatar?>(
      future: _future,
      builder: (context, snap) {
        return widget.builder(context, snap.data);
      },
    );
  }
}
