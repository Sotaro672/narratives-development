//frontend\mall\lib\features\avatar\presentation\page\avatar_edit.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/routing/navigation.dart';
import '../../../../app/routing/routes.dart';
import '../../../../app/shell/presentation/components/header.dart';

import '../component/avatar_form.dart';
import '../vm/avatar_form_vm.dart';

class AvatarEditPage extends StatefulWidget {
  const AvatarEditPage({super.key});

  @override
  State<AvatarEditPage> createState() => _AvatarEditPageState();
}

class _AvatarEditPageState extends State<AvatarEditPage> {
  late final AvatarFormVm _vm;
  bool _loaded = false;

  @override
  void initState() {
    super.initState();
    _vm = AvatarFormVm(mode: AvatarFormMode.edit);
    _vm.addListener(_onVmChanged);
    _load();
  }

  Future<void> _load() async {
    await _vm.loadInitialIfNeeded();
    if (!mounted) return;
    setState(() => _loaded = true);
  }

  void _onVmChanged() {
    if (!mounted) return;
    setState(() {});
  }

  @override
  void dispose() {
    _vm.removeListener(_onVmChanged);
    _vm.dispose();
    super.dispose();
  }

  String _consumeBackTo() {
    try {
      // consumeReturnTo() が non-nullable の場合に合わせる
      final v = NavStore.I.consumeReturnTo();
      final s = v.toString().trim();
      if (s.isNotEmpty) return s;
    } catch (_) {}
    return AppRoutePath.avatar;
  }

  Future<void> _onSave() async {
    final ok = await _vm.save();
    if (!mounted) return;

    if (ok) {
      final backTo = _consumeBackTo();
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
              title: 'Edit Avatar',
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
                    child: !_loaded
                        ? const Center(child: CircularProgressIndicator())
                        : AvatarForm(vm: _vm, onSave: _onSave),
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
