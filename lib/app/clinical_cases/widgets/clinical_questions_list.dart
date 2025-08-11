// import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';

import 'package:ema_educacion_medica_avanzada/app/clinical_cases/controllers/clinical_case_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class ClinicalQuestionsList extends StatefulWidget {
  final ClinicalCaseController controller;
  final Widget header;

  const ClinicalQuestionsList({
    super.key,
    required this.controller,
    required this.header,
  });

  @override
  State<ClinicalQuestionsList> createState() => _ClinicalQuestionsListState();
}

class _ClinicalQuestionsListState extends State<ClinicalQuestionsList> {
  final scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    widget.controller.setScrollController(scrollController);
  }

  @override
  void dispose() {
    widget.controller.setScrollController(null);
    super.dispose();
  }

  Widget _buildQuestion(QuestionResponseModel question) {
    if (question.type == QuestionType.open) {
      return QuestionMessageOpen(question: question);
    }

    if (question.type == QuestionType.singleChoice) {
      return QuestionMessageSingle(question: question);
    }

    if (question.type == QuestionType.trueFalse) {
      return QuestionMessageTrueFalse(question: question);
    }

    return SizedBox();
  }

  @override
  Widget build(BuildContext context) {
    widget.controller.setScrollController(scrollController);

    return SingleChildScrollView(
      controller: scrollController,
      padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 32),
      child: Obx(() {
        final widgets = <Widget>[widget.header];

        // Build questions based on the current state
        for (var question in widget.controller.questions) {
          widgets.add(_buildQuestion(question));
        }

        return Column(
          mainAxisSize: MainAxisSize.max,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: widgets,
        );
      }),
    );
  }
}
