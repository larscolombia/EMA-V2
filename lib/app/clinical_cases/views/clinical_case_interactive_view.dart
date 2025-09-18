import 'package:ema_educacion_medica_avanzada/app/chat/widgets/animations/chat_typing_indicator.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/widgets/chat_message_ai.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/widgets/chat_message_user.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/widgets/clinical_question_inputs.dart';
import 'package:ema_educacion_medica_avanzada/common/layouts/app_layout.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/show_error_widget.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

class ClinicalCaseInteractiveView extends StatefulWidget {
  const ClinicalCaseInteractiveView({super.key});

  @override
  State<ClinicalCaseInteractiveView> createState() =>
      _ClinicalCaseInteractiveViewState();
}

class _ClinicalCaseInteractiveViewState
    extends State<ClinicalCaseInteractiveView> {
  final GlobalKey<ScaffoldState> _scaffoldKey = GlobalKey<ScaffoldState>();
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

  @override
  Widget build(BuildContext context) {
    controller.setScrollController(scrollController);

    return AppLayout(
      key: _scaffoldKey,
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
                  controller: controller.scrollController,
                  padding: const EdgeInsets.only(left: 8, right: 8, top: 16),
                  child: Obx(() {
                    List<Widget> items =
                        controller.messages.map((message) {
                          return message.aiMessage
                              ? ChatMessageAi(message: message)
                              : ChatMessageUser(message: message);
                        }).toList();

                    return Column(
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
                                    crossAxisAlignment:
                                        CrossAxisAlignment.start,
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
                                            color:
                                                Theme.of(context).primaryColor,
                                          ),
                                        ],
                                      ),
                                      const SizedBox(height: 8),
                                      Text(
                                        isHeaderExpanded
                                            ? clinicalCase.anamnesis
                                            : _getTruncatedText(
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

                        ...items,

                        if (controller.isTyping.value)
                          Padding(
                            padding: EdgeInsets.only(top: 8.0),
                            child: Obx(() {
                              final stage = controller.currentStage.value;
                              final captions =
                                  <String>[
                                    'Procesando…',
                                    if (stage == 'rag_search' || stage.isEmpty)
                                      'Analizando vector…',
                                    if (stage == 'doc_only')
                                      'Usando documentos adjuntos…',
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
                    );
                  }),
                ),
              ),

              ClinicalQuestionInputs(controller: controller),
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
