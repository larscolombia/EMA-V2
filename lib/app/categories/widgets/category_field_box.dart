import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class CategoryFieldBox extends GetView<CategoriesController> {
  final String title;

  CategoryFieldBox({
    super.key,
    this.title = 'Categoria',
  });

  final _outlineEnableBorder = OutlineInputBorder(
    borderRadius: BorderRadius.circular(16),
    borderSide: BorderSide(
      color: Colors.transparent,
    ),
  );

  final _outlineFocusBorder = OutlineInputBorder(
    borderRadius: BorderRadius.circular(16),
    borderSide: BorderSide(
      color: AppStyles.primary900,
    ),
  );

  @override
  Widget build(BuildContext context) {
    final textController = TextEditingController();
    final focusNode = FocusNode();

    final buttons = Padding(
      padding: const EdgeInsets.only(right: 6),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            onPressed: () {
              textController.clear();
              controller.setCategoryFilter('');
            },
            padding: EdgeInsets.all(8),
            icon: AppIcons.closeSquare(
              height: 24,
              width: 24,
              color: AppStyles.tertiaryColor,
            ),
          ),
        ],
      ),
    );

    final inputDecoration = InputDecoration(
      label: Text(title),
      enabledBorder: _outlineEnableBorder,
      focusedBorder: _outlineFocusBorder,
      contentPadding: EdgeInsets.symmetric(horizontal: 12, vertical: 0),
      floatingLabelBehavior: FloatingLabelBehavior.never,
      suffixIcon: buttons,
      filled: true,
    );

    final textFormField = TextFormField(
      autocorrect: false,
      focusNode: focusNode,
      controller: textController,

      decoration: inputDecoration,
      keyboardType: TextInputType.text,
      maxLines: 1,

      onFieldSubmitted: (value) {
        controller.setCategoryFilter(value);
        focusNode.unfocus();
      },

      onChanged: (value) {
        controller.setCategoryFilter(value);
      },

      onEditingComplete: () {
        final namePart = textController.text;
        controller.setCategoryFilter(namePart);
      },

      onTapOutside: (event) {
        focusNode.unfocus();
      },
    );

    return Obx(() {
      final text = controller.categorySelectedName.value;

      if (text.isNotEmpty) {
        textController.text = text;
        focusNode.unfocus();
      }
      return textFormField;
    });
  }
}
