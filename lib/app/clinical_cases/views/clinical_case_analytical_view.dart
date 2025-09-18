import 'package:ema_educacion_medica_avanzada/app/chat/widgets/animations/chat_typing_indicator.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/widgets/chat_message_ai.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/widgets/chat_message_user.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/widgets/clinical_case_chat_field_box.dart';
import 'package:ema_educacion_medica_avanzada/common/layouts/app_layout.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/outline_ai_button.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/show_error_widget.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class ClinicalCaseAnalyticalView extends StatefulWidget {
  const ClinicalCaseAnalyticalView({super.key});

  @override
  State<ClinicalCaseAnalyticalView> createState() =>
      _ClinicalCaseAnalyticalViewState();
}

class _ClinicalCaseAnalyticalViewState
    extends State<ClinicalCaseAnalyticalView> {
  final controller = Get.find<ClinicalCaseController>();
  final scrollController = ScrollController();
  bool isHeaderExpanded = false;

  @override
  void initState() {
    super.initState();
    controller.setScrollController(scrollController);
  }

  @override
  void dispose() {
    controller.setScrollController(null);
    super.dispose();
  }

  void toggleHeaderExpansion() {
    setState(() {
      isHeaderExpanded = !isHeaderExpanded;
    });
  }

  String _getTruncatedText(String text) {
    if (text.isEmpty) return '';
    if (text.length <= 100) return text;
    return '${text.substring(0, 100)}...';
  }

  Widget _getFullCaseWidget(ClinicalCaseModel clinicalCase) {
    List<Widget> sections = [];

    // Anamnesis
    if (clinicalCase.anamnesis.isNotEmpty) {
      sections.addAll([
        Text(
          'ANAMNESIS:',
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.bold,
            color: Theme.of(context).primaryColor,
          ),
        ),
        const SizedBox(height: 4),
        Text(
          clinicalCase.anamnesis,
          style: Theme.of(context).textTheme.bodyMedium,
        ),
        const SizedBox(height: 16),
      ]);
    }

    // Examen físico
    if (clinicalCase.physicalExamination.isNotEmpty) {
      sections.addAll([
        Text(
          'EXAMEN FÍSICO:',
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.bold,
            color: Theme.of(context).primaryColor,
          ),
        ),
        const SizedBox(height: 4),
        Text(
          clinicalCase.physicalExamination,
          style: Theme.of(context).textTheme.bodyMedium,
        ),
        const SizedBox(height: 16),
      ]);
    }

    // Pruebas diagnósticas
    if (clinicalCase.diagnosticTests.isNotEmpty) {
      sections.addAll([
        Text(
          'PRUEBAS DIAGNÓSTICAS:',
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.bold,
            color: Theme.of(context).primaryColor,
          ),
        ),
        const SizedBox(height: 4),
        Text(
          clinicalCase.diagnosticTests,
          style: Theme.of(context).textTheme.bodyMedium,
        ),
        const SizedBox(height: 16),
      ]);
    }

    // Diagnóstico final
    if (clinicalCase.finalDiagnosis.isNotEmpty) {
      sections.addAll([
        Text(
          'DIAGNÓSTICO FINAL:',
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.bold,
            color: Theme.of(context).primaryColor,
          ),
        ),
        const SizedBox(height: 4),
        Text(
          clinicalCase.finalDiagnosis,
          style: Theme.of(context).textTheme.bodyMedium,
        ),
        const SizedBox(height: 16),
      ]);
    }

    // Manejo
    if (clinicalCase.management.isNotEmpty) {
      sections.addAll([
        Text(
          'MANEJO:',
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.bold,
            color: Theme.of(context).primaryColor,
          ),
        ),
        const SizedBox(height: 4),
        Text(
          clinicalCase.management,
          style: Theme.of(context).textTheme.bodyMedium,
        ),
      ]);
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: sections,
    );
  }

  @override
  Widget build(BuildContext context) {
    controller.setScrollController(scrollController);

    return AppLayout(
      backRoute: Routes.home.name,
      body: controller.obx(
        (clinicalCase) {
          if (clinicalCase == null) {
            return const StateMessageWidget(
              message: 'No se encontró un caso clínico disponible',
              type: StateMessageType.noSearchResults,
            );
          }

          return Column(
            children: [
              Expanded(
                child: SingleChildScrollView(
                  controller: scrollController,
                  padding: const EdgeInsets.only(left: 8, right: 8, top: 16),
                  child: Column(
                    children: [
                      // Container para mayor separación visual
                      Container(
                        margin: const EdgeInsets.only(bottom: 12),
                        child: Material(
                          color: Colors.transparent,
                          child: InkWell(
                            onTap: toggleHeaderExpansion,
                            borderRadius: BorderRadius.circular(8.0),
                            child: Ink(
                              decoration: BoxDecoration(
                                borderRadius: BorderRadius.circular(8.0),
                                color: Colors.grey.withValues(alpha: 0.08),
                              ),
                              child: Padding(
                                padding: const EdgeInsets.all(12.0),
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Row(
                                      children: [
                                        Expanded(
                                          child: Text(
                                            clinicalCase.type.description,
                                            style: TextStyle(
                                              fontWeight: FontWeight.bold,
                                              color:
                                                  Theme.of(
                                                    context,
                                                  ).primaryColor,
                                            ),
                                          ),
                                        ),
                                        Icon(
                                          isHeaderExpanded
                                              ? Icons.keyboard_arrow_up
                                              : Icons.keyboard_arrow_down,
                                          color: Theme.of(context).primaryColor,
                                        ),
                                      ],
                                    ),
                                    const SizedBox(height: 8),
                                    isHeaderExpanded
                                        ? _getFullCaseWidget(clinicalCase)
                                        : Text(
                                          _getTruncatedText(
                                            clinicalCase.anamnesis,
                                          ),
                                          style:
                                              Theme.of(
                                                context,
                                              ).textTheme.bodyMedium,
                                        ),
                                  ],
                                ),
                              ),
                            ),
                          ),
                        ),
                      ),

                      // Messages list - keep this in a separate Obx
                      Obx(
                        () => Column(
                          children: [
                            // Trigger para navegación a evaluación cuando el caso se marca completo
                            if (controller.isComplete.value &&
                                !controller.evaluationGenerated.value &&
                                !controller.evaluationInProgress.value)
                              FutureBuilder(
                                future:
                                    (() async {
                                      controller.generateFinalEvaluation();
                                      return true;
                                    })(),
                                builder: (_, __) => const SizedBox.shrink(),
                              ),
                            ...controller.messages.map((message) {
                              return message.aiMessage
                                  ? ChatMessageAi(message: message)
                                  : ChatMessageUser(message: message);
                            }),

                            if (controller.isTyping.value)
                              Padding(
                                padding: EdgeInsets.only(top: 8.0),
                                child: Obx(() {
                                  final stage = controller.currentStage.value;
                                  final captions =
                                      <String>[
                                        'Procesando…',
                                        if (stage == 'rag_search' ||
                                            stage.isEmpty)
                                          'Analizando vector…',
                                        if (stage == 'pubmed_search')
                                          'Buscando en PubMed…',
                                        if (stage == 'rag_found')
                                          'Fuente interna encontrada…',
                                        if (stage == 'pubmed_found')
                                          'Referencia PubMed encontrada…',
                                        'Enviando respuesta…',
                                      ].where((e) => e.isNotEmpty).toList();
                                  return ChatTypingIndicator(
                                    captions: captions,
                                    captionInterval: const Duration(seconds: 2),
                                  );
                                }),
                              ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              ClinicalCaseChatFieldBox(controller: controller),
              // Botón de finalizar caso analítico (antes de cierre automático)
              Obx(() {
                if (controller.shouldOfferAnalyticalFinalize) {
                  return Padding(
                    padding: const EdgeInsets.fromLTRB(20, 4, 20, 16),
                    child: OutlineAiButton(
                      text:
                          controller.isFinalizingCase.value
                              ? 'Finalizando...'
                              : 'Finalizar Caso',
                      onPressed:
                          controller.isFinalizingCase.value
                              ? null
                              : () => controller.finalizeAnalyticalFromUser(),
                      isLoading: controller.isFinalizingCase.value,
                    ),
                  );
                }
                return const SizedBox(height: 8);
              }),
            ],
          );
        },
        onLoading: const StateMessageWidget(
          message: 'Preparando Caso Clínico...',
          type: StateMessageType.download,
          showLoading: true,
        ),
        onEmpty: const StateMessageWidget(
          message: 'No se encontró un caso clínico  disponible',
          type: StateMessageType.noSearchResults,
        ),
        onError: (error) {
          return StateMessageWidget(
            message: error ?? 'Error al cargar el caso clínico',
            showHomeButton: true,
            type: StateMessageType.error,
          );
        },
      ),
    );
  }
}
