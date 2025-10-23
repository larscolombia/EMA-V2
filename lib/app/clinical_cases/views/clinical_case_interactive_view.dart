import 'package:ema_educacion_medica_avanzada/app/chat/widgets/animations/chat_typing_indicator.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/widgets/chat_message_ai.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/widgets/chat_message_user.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/widgets/clinical_question_inputs.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/widgets/clinical_question_message_single.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/question_type.dart';
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
                    // IDs de preguntas respondidas (para no duplicar sus respuestas)
                    final answeredQuestionIds =
                        controller.questions
                            .where((q) => q.isAnswered && q.options.isNotEmpty)
                            .map((q) => q.id)
                            .toSet();

                    List<Widget> items = [];

                    for (var message in controller.messages) {
                      if (message.aiMessage) {
                        // Buscar si este mensaje corresponde a una pregunta
                        final matchingQuestion =
                            controller.questions
                                .where((q) => q.id == message.uid)
                                .firstOrNull;

                        if (matchingQuestion != null &&
                            matchingQuestion.options.isNotEmpty) {
                          // Es una pregunta con opciones, usar widget especializado
                          if (matchingQuestion.type ==
                              QuestionType.singleChoice) {
                            items.add(
                              ClinicalQuestionMessageSingle(
                                question: matchingQuestion,
                              ),
                            );
                            continue;
                          }
                        }
                        // Es un mensaje normal de IA (feedback)
                        items.add(ChatMessageAi(message: message));
                      } else {
                        // Es mensaje de usuario
                        // Verificar si es una respuesta a una pregunta que ya se muestra en el widget
                        // Las respuestas a preguntas tienen el texto que coincide con answerdString
                        bool isQuestionResponse = answeredQuestionIds.any((
                          qId,
                        ) {
                          final q =
                              controller.questions
                                  .where((quest) => quest.id == qId)
                                  .firstOrNull;
                          return q != null && q.answerdString == message.text;
                        });

                        // Solo mostrar si NO es una respuesta de pregunta (ya incluida en el widget)
                        if (!isQuestionResponse) {
                          items.add(ChatMessageUser(message: message));
                        }
                      }
                    }

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
