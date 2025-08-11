// ignore_for_file: public_member_api_docs, sort_constructors_first

import 'package:ema_educacion_medica_avanzada/app/actions/helpers/actions_helper.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/models/action_model.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/models/action_type.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/services/actions_service.dart';
import 'package:ema_educacion_medica_avanzada/config/routes/app_pages.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_service.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class ActionsListController extends GetxService {
  final _actionsList = <ActionModel>[].obs;
  final _actionsService = Get.find<ActionsService>();
  final _categoryFilter = RxInt(0);
  final _changeCounter = 0.obs;
  final _page = RxInt(1);
  final _titleFilter = RxString('');
  final _typeFilter = RxString('');
  final _userService = Get.find<UserService>();

  final typeTitle = ''.obs;
  final loading = false.obs;
  final bool useTypeFilter;

  List<ActionModel> get actions => _actionsList;
  int get page => _page.value;
  RxString get titleFilter => _titleFilter;

  ActionsListController({
    this.useTypeFilter = true,
  });

  @override
  void onInit() {
    super.onInit();

    loadActions(0);

    debounce(
      _changeCounter,
      loadActions,
      time: const Duration(milliseconds: 350),
    );
  }

  Future<void> loadActions(int _) async {
    if (loading.value) return;
    loading.value = true;

    final userId = _userService.currentUser.value.id;

    final whereArgs = ActionsHelper.getWhereArgs(
      userId: userId,
      categoryId: _categoryFilter.value != 0 ? _categoryFilter.value : null,
      title: _titleFilter.value,
      type: _typeFilter.value,
      page: _page.value,
    );

    final where = ActionsHelper.getWhere(
      userId: userId,
      categoryId: _categoryFilter.value != 0 ? _categoryFilter.value : null,
      title: _titleFilter.value,
      type: _typeFilter.value,
      page: _page.value,
    );

    final actions = await _actionsService.getActions(where: where, whereArgs: whereArgs, page: _page.value);

    _actionsList.assignAll(actions);

    loading.value = false;
  }

  void setCategoryFilter(int categoryId) {
    _categoryFilter.value = categoryId;
    _page.value = 1; // Resetear la página al cambiar el filtro

    _changeCounter.value++;
  }

  void setTitleFilter(String title) {
    _titleFilter.value = title;
    _page.value = 1;
    
    _changeCounter.value++;
  }

  void nextPage() async {
    if (loading.value) return;
    loading.value = true;

    _page.value++;

    final userId = _userService.currentUser.value.id;

    final whereArgs = ActionsHelper.getWhereArgs(
      userId: userId,
      categoryId: _categoryFilter.value != 0 ? _categoryFilter.value : null,
      title: _titleFilter.value,
      type: _typeFilter.value,
      page: _page.value,
    );

    final where = ActionsHelper.getWhere(
      userId: userId,
      categoryId: _categoryFilter.value != 0 ? _categoryFilter.value : null,
      title: _titleFilter.value,
      type: _typeFilter.value,
      page: _page.value,
    );

    final actions = await _actionsService.getActions(where: where, whereArgs: whereArgs, page: _page.value);

    _actionsList.addAll(actions);

    loading.value = false;    
  }

  void cleanFilters() {
    _categoryFilter.value = 0;
    _titleFilter.value = '';
    _page.value = 1;

    _changeCounter.value++;
  }

  void showActionsList(ActionType? type) {
    _setTypeFilter(type);

    WidgetsBinding.instance.addPostFrameCallback((_) {
      Get.toNamed(Routes.actionsList.name);
    });
  }

  void _setTypeFilter(ActionType? type) {
    typeTitle.value = type?.title ?? '';
    _typeFilter.value = type?.toString() ?? '';

    _page.value = 1;
    loadActions(0);
  }
}
