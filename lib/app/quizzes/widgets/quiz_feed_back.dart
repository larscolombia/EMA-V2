// import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';

import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class QuizFeedBack extends GetView<QuizController> {
  final scrollController = ScrollController();

  final List<Widget> header;

  QuizFeedBack({
    super.key,
    required this.header,
  });

  @override
  Widget build(BuildContext context) {
    controller.setScrollController(scrollController);

    return SingleChildScrollView(
      controller: scrollController,
      padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 32),
      child: Obx(() {

        return Column(
          mainAxisSize: MainAxisSize.max,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ...header,
            Text(controller.state?.feedback ?? ''),
          ],
        );
      }),
    );
  }
}
