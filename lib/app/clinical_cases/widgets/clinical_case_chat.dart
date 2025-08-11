import 'package:ema_educacion_medica_avanzada/app/chat/chat.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class ClinicalCaseChat extends GetView<ClinicalCaseController> {
  const ClinicalCaseChat({
    super.key
  });

  Widget onLoading() {
    return const Center(child: CircularProgressIndicator());
  }

  Widget onEmpty() {
    return const Center(
      child: Text('Caso clínico no tiene mensajes'),
    );
  }

  Widget onError(String? errorMessage) {
    return Center(
      child: Text(errorMessage ?? 'Error al cargar los casos clínicos'),
    );
  }

  @override
  Widget build(BuildContext context) {
    return controller.obx(
      (clinicalCase) {
        if (clinicalCase == null) {
          return const Center(
            child: Text('Caso clínico no encontrado'),
          );
        }

        return Column(
          mainAxisSize: MainAxisSize.max,
          mainAxisAlignment: MainAxisAlignment.start,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ChatSubtitle(name: clinicalCase.title),
            ChatMessagesList(),
          ],
        );
      },
      onLoading: onLoading(),
      onEmpty: onEmpty(),
      onError: onError,
    );
  }
}
