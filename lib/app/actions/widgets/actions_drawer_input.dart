import 'package:ema_educacion_medica_avanzada/app/actions/controllers/actions_drawer_list_controller.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/app_icons.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class ActionsDrawerSearchInput extends StatelessWidget {
  final _actionsController = Get.find<ActionsDrawerListController>();
  
  final String title;

  ActionsDrawerSearchInput({
    super.key,
    this.title = 'Buscar',
  });

  final _outlineEnableBorder = OutlineInputBorder(
    borderRadius: BorderRadius.circular(8),
    borderSide: BorderSide(
      color: Colors.transparent,
    ),
  );

  final _outlineFocusBorder = OutlineInputBorder(
    borderRadius: BorderRadius.circular(8),
    borderSide: BorderSide(
      color: AppStyles.primary900,
    ),
  );

  @override
  Widget build(BuildContext context) {
    final textController = TextEditingController();
    final focusNode = FocusNode();

    final closeButton = IconButton(
      onPressed: () {
        _actionsController.cleanFilters();
        textController.clear();
        focusNode.requestFocus();
      },
      icon: AppIcons.cancel(
        height: 36,
        width: 36,
        color: AppStyles.tertiaryColor,
      ),
    );

    final searchButton = IconButton(
      onPressed: () {
        _actionsController.setTitleFilter(textController.value.text);
      },
      icon: AppIcons.search(
        height: 36,
        width: 36,
        color: AppStyles.tertiaryColor,
      ),
    );

    final button = Obx(() {
      return _actionsController.titleFilter.value.isEmpty
        ? searchButton
        : closeButton;
    });

    final inputDecoration = InputDecoration(
      contentPadding: EdgeInsets.symmetric(horizontal: 12, vertical: 2),
      label: Text(title),
      enabledBorder: _outlineEnableBorder,
      focusedBorder: _outlineFocusBorder,
      floatingLabelBehavior: FloatingLabelBehavior.never,
      suffixIcon: button,
      filled: true,
    );

    return TextFormField(
      autocorrect: false,
      focusNode: focusNode,
      controller: textController,
    
      decoration: inputDecoration,
      keyboardType: TextInputType.text,
      maxLines: null,

      onChanged: (value) {
        _actionsController.setTitleFilter(value);
      },
    
      onFieldSubmitted: (value) {
        _actionsController.setTitleFilter(value);
      },
    
      onTapOutside: (event) {
        focusNode.unfocus();
      },
    );
  }
}
