import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class QuizzesOptions extends StatelessWidget {
  final categoriesController = Get.find<CategoriesController>();
  final quizController = Get.find<QuizController>();
  final uiObserverService = Get.find<UiObserverService>();
  
  final bool withCategory;
  final Rx<double> _numQuestions = 15.0.obs;
  final Rx<double> _level = 2.0.obs;

  QuizzesOptions({
    super.key,
    this.withCategory = false,
  });

  void generateQuiz() {
    quizController.generate(
      numQuestions: _numQuestions.value.toInt(),
      level: QuizzLevel.fromValue(_level.value),
      category: withCategory
        ? categoriesController.currentCategory.value
        : null,
    );
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Obx(() {
          final int number = _numQuestions.value.toInt();
          final bool hidden = uiObserverService.isKeyboardVisible.value;

          return hidden ? const SizedBox() : Row(
            children: [
              Expanded(flex: 1, child: Text('$number preguntas')),
              Expanded(
                flex: 2,
                child: Slider(
                  value: _numQuestions.value,
                  // min: 10.0,
                  // max: 50.0,
                  // divisions: 4,
                  min: 5.0,
                  max: 25.0,
                  divisions: 4,
                  onChanged: (value) {
                    _numQuestions.value = value.roundToDouble();
                  },
                ),
              ),
            ],
          );
        }),

        Obx(() {
          final bool hidden = uiObserverService.isKeyboardVisible.value;

          return hidden ? const SizedBox() : Row(
            children: [
              Expanded(
                flex: 1,
                child: Text('Complejidad'),
              ),
              Expanded(
                flex: 2,
                child: Slider(
                  value: _level.value,
                  min: 1.0,
                  max: 3.0,
                  divisions: 2,
                  onChanged: (value) {
                    _level.value = value.roundToDouble();
                  },
                ),
              ),
            ],
          );
        }),

        if (withCategory) const SizedBox(height: 8),
        
        if (withCategory) CategoryFieldBox(),

        const SizedBox(height: 16),

        Obx(() {
          final bool hidden = uiObserverService.isKeyboardVisible.value;
          final enabled = !withCategory || (withCategory && categoriesController.currentCategory.value.id > 0);

          return hidden ? const SizedBox() : OutlineAiButton(
            text: 'Genera cuestionario',
            enabled: enabled,
            onPressed: generateQuiz,
          );
        }),
      ],
    );
  }
}
