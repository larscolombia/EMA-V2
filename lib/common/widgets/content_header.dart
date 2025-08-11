import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class ContentHeader extends StatelessWidget {
  final String breadcrumb;
  final String subtitle;
  final double bottomPadding;
  final RxBool collapsed = true.obs;

  ContentHeader({
    super.key,
    required this.breadcrumb,
    required this.subtitle,
    this.bottomPadding = 20.0,
  });

  static List<Widget> headerItems({required String subtitle, required String breadcrumb, double bottomPadding = 20}) {
    return [
      Text(
        breadcrumb,
        style: AppStyles.breadCrumb,
      ),

      SizedBox(height: 12),

      Text(subtitle, maxLines: null, style: AppStyles.subtitle),

      Divider(),

      SizedBox(height: bottomPadding),
    ];
  }

  @override
  Widget build(BuildContext context) {
    List<Widget> rxHeaderItems({required String subtitle, required String breadcrumb, double bottomPadding = 20}) {
      return [
        Text(
          breadcrumb,
          style: AppStyles.breadCrumb,
        ),

        SizedBox(height: 12),

        Obx(() {
          return collapsed.value == true
            ? Text(subtitle, maxLines: 3, overflow: TextOverflow.ellipsis, style: AppStyles.subtitle)
            : Text(subtitle, maxLines: null, style: AppStyles.subtitle);
        }),

        Divider(),

        SizedBox(height: bottomPadding),
      ];
    }

    return GestureDetector(
      onTap: () {
        collapsed.value = !collapsed.value;
      },
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: rxHeaderItems(
          subtitle: subtitle,
          breadcrumb: breadcrumb,
          bottomPadding: bottomPadding,
        ),
      ),
    );
  }
}
