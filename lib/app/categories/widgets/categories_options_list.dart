import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class CategoriesOptionsList extends StatelessWidget {
  final controller = Get.find<CategoriesController>();

  CategoriesOptionsList({
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    // categoriesFiltered
    return Obx(() {
      return ListView.separated(
        reverse: true,
        itemCount: controller.categoriesFiltered.length,
        itemBuilder: (context, index) {
          final category = controller.categoriesFiltered[index];

          return ListTile(
            title: Text(category.name),
            textColor: AppStyles.whiteColor,
            onTap: () {
              controller.setCategorySelected(category);
            },
          );
        },
        separatorBuilder: (context, index) => Divider(
          color: AppStyles.whiteColor.withAlpha(100),
        ),
      );
    });
  }
}
