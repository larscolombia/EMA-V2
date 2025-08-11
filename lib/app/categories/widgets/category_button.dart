import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';


class CategoryButton extends StatelessWidget {
  final CategoryModel category;
  final VoidCallback onPressed;

  const CategoryButton({
    super.key,
    required this.category,
    required this.onPressed,
  });

  @override
  Widget build(BuildContext context) {
    return OutlinedButton(
      onPressed: onPressed,
      style: ButtonStyle(
        side: WidgetStateProperty.all(BorderSide(color: AppStyles.tertiaryColor)),
      ),
      child: Text(
        category.name,
        style: AppStyles.categoryBtn,
      ),
    );
  }
}
