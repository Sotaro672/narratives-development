// frontend\mall\lib\features\auth\presentation\page\avatar_create.dart
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/routing/navigation.dart';
import '../../../../app/routing/routes.dart';
import '../../../../app/shell/presentation/components/header.dart';

import '../../../avatar/presentation/component/avatar_form.dart';
import '../../../avatar/presentation/vm/avatar_form_vm.dart';
import '../../../avatar/infrastructure/avatar_api_client.dart';

class AvatarCreatePage extends StatefulWidget {
  const AvatarCreatePage({super.key});

  @override
  State<AvatarCreatePage> createState() => _AvatarCreatePageState();
}

class _AvatarCreatePageState extends State<AvatarCreatePage> {
  late final AvatarApiClient _api;
  late final AvatarFormVm _vm;

  // ✅ dev log helper (Chrome console に出ます)
  void _log(String msg) {
    if (!kDebugMode) return;
    debugPrint('[AvatarCreatePage] $msg');
  }

  @override
  void initState() {
    super.initState();

    _api = AvatarApiClient();

    // ✅ create は "submitCreate" を注入する（AvatarFormVm.save は create で submitCreate を要求する）
    _vm = AvatarFormVm(
      mode: AvatarFormMode.create,
      apiClient: _api,
      submitCreate: (body) async {
        final payload = body.toJson();
        _log('submitCreate start payload=$payload');

        // POST /mall/avatars
        await _api.createAvatar(payload);

        _log('submitCreate done');
      },
    );

    _vm.addListener(_onVmChanged);

    _log('initState done (create mode)');
  }

  void _onVmChanged() {
    if (!mounted) return;
    setState(() {});
  }

  @override
  void dispose() {
    _vm.removeListener(_onVmChanged);
    _vm.dispose();
    _api.dispose();
    super.dispose();
  }

  String _consumeBackTo() {
    try {
      final v = NavStore.I.consumeReturnTo();
      final s = v.toString().trim();
      if (s.isNotEmpty) return s;
    } catch (_) {}
    return AppRoutePath.avatar;
  }

  Future<void> _onSave() async {
    _log('onSave tapped');

    // ✅ submitCreate は VM に注入済み
    final ok = await _vm.save();
    _log('onSave result ok=$ok msg=${_vm.msg}');

    if (!mounted) return;

    if (ok) {
      final backTo = _consumeBackTo();
      _log('navigate to backTo=$backTo');
      context.go(backTo);
    }
  }

  @override
  Widget build(BuildContext context) {
    final backTo = _consumeBackTo();

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            AppHeader(
              title: 'アバター作成',
              showBack: true,
              backTo: backTo,
              actions: const [],
              onTapTitle: () => context.go(AppRoutePath.home),
            ),
            Expanded(
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 560),
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(16),
                    child: AvatarForm(vm: _vm, onSave: _onSave),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
