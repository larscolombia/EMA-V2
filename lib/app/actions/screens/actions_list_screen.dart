// ignore_for_file: public_member_api_docs, sort_constructors_first
import 'package:ema_educacion_medica_avanzada/app/actions/widgets/actions_list_widget.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/widgets/actions_search_input.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/app_icons.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class ActionsListScreen extends StatelessWidget {
  
  const ActionsListScreen({
    super.key,
  });

  @override
  Widget build(BuildContext context) {

    final appBar = AppBar(
      leading: IconButton(
        color: Colors.black,
        icon: AppIcons.arrowLeftSquare(
          height: 34,
          width: 34,
        ),
        onPressed: () {
          Get.back(closeOverlays: true);
        },
      ),
      title: ActionsSearchInput(),
    );

    return Scaffold(
      appBar: appBar,
      body: ActionsListWidget(),
    );
  }
}
