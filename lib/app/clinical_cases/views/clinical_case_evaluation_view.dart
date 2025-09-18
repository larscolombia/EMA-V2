import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/common/layouts/app_layout.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/outline_ai_button.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/show_error_widget.dart';
import 'package:ema_educacion_medica_avanzada/config/routes/app_pages.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/widgets/full_feedback_animated.dart';

class ClinicalCaseEvaluationView extends StatefulWidget {
  const ClinicalCaseEvaluationView({super.key});

  @override
  State<ClinicalCaseEvaluationView> createState() =>
      _ClinicalCaseEvaluationViewState();
}

class _ClinicalCaseEvaluationViewState
    extends State<ClinicalCaseEvaluationView> {
  final controller = Get.find<ClinicalCaseController>();
  bool showUserTurns = false;

  ChatMessageModel? _evaluationMessage() {
    final msgs = controller.messages.where((m) => m.aiMessage).toList();
    if (msgs.isEmpty) return null;
    final caseType = controller.currentCase.value?.type;
    if (caseType == ClinicalCaseType.analytical) {
      for (final m in msgs.reversed) {
        final lower = m.text.toLowerCase();
        if (lower.contains('puntuación') && lower.contains('referencias')) {
          return m;
        }
      }
      return msgs.last;
    } else {
      // Interactivo: priorizar mensaje con 'Resumen Final:' si existe; evitar el que inicie con 'Evaluación final interactiva' (duplicado)
      ChatMessageModel? resumen;
      for (final m in msgs.reversed) {
        final lower = m.text.toLowerCase();
        if (resumen == null && lower.startsWith('resumen final:')) {
          resumen = m; // preferido
        }
        if (lower.startsWith('evaluación final interactiva')) {
          // ignorar duplicado generado por capa legacy
          continue;
        }
        if (resumen != null) break;
      }
      return resumen ?? msgs.last;
    }
  }

  @override
  void initState() {
    super.initState();
    if (!controller.evaluationGenerated.value &&
        controller.currentCase.value?.type == ClinicalCaseType.analytical) {
      controller.generateFinalEvaluation();
    }
  }

  // (Se removieron heurísticas de métricas de resultados; el diseño ahora depende del feedback generado)

  @override
  Widget build(BuildContext context) {
    return AppLayout(
      backRoute: Routes.home.name,
      body: Obx(() {
        final caseModel = controller.currentCase.value;
        if (caseModel == null) {
          return const StateMessageWidget(
            message: 'Caso no disponible',
            type: StateMessageType.noSearchResults,
          );
        }
        final isInteractive = caseModel.type == ClinicalCaseType.interactive;
        final hasHidden = controller.hasHiddenInteractiveSummary;
        // Evaluación mostrada solo si ya se reveló (interactiveEvaluationGenerated) o es analítico
        final canShowEval =
            caseModel.type == ClinicalCaseType.analytical ||
            controller.interactiveEvaluationGenerated.value;
        if (!canShowEval && isInteractive && controller.isComplete.value) {
          // Mostrar pantalla con botón para revelar
          return _buildRevealButton(context, hasHidden);
        }
        final evalMsg = _evaluationMessage();
        if (evalMsg == null) {
          return const StateMessageWidget(
            message: 'No se encontró la evaluación generada.',
            type: StateMessageType.noSearchResults,
          );
        }
        return SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 32),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                caseModel.type == ClinicalCaseType.interactive
                    ? 'Evaluación final interactiva'
                    : 'Evaluación final analítica',
                style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                  color: Theme.of(context).primaryColor,
                ),
              ),
              const SizedBox(height: 12),
              Text(
                caseModel.anamnesis,
                style: Theme.of(context).textTheme.bodySmall,
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
              ),
              const Divider(height: 32),
              // Feedback con mismo formato y orden que cuestionarios
              FullFeedbackAnimated(
                fitGlobal: evalMsg.text,
                questions: controller.questions.toList(),
                animate: false,
                renderMarkdown: true,
              ),
              const SizedBox(height: 24),
              const SizedBox(height: 16),
              Row(
                children: [
                  Expanded(
                    child: OutlineAiButton(
                      text: 'Ver intervenciones',
                      onPressed: () => _showUserTurnsBottomSheet(context),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: OutlineAiButton(
                      text: 'Inicio',
                      onPressed: () => Get.offAllNamed(Routes.home.name),
                    ),
                  ),
                ],
              ),
            ],
          ),
        );
      }),
    );
  }

  Widget _buildRevealButton(BuildContext context, bool hasHidden) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.lock_outline, size: 52),
            const SizedBox(height: 24),
            Text(
              'Evaluación final disponible',
              style: Theme.of(context).textTheme.titleLarge,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 12),
            Text(
              'Pulsa el botón para ver tu evaluación final del caso interactivo.',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
            const SizedBox(height: 32),
            OutlineAiButton(
              text: 'Ver evaluación final',
              onPressed:
                  hasHidden
                      ? () async {
                        await controller.showInteractiveSummaryIfAvailable();
                        setState(() {});
                      }
                      : null,
              isLoading: false,
            ),
            const SizedBox(height: 16),
            OutlineAiButton(
              text: 'Inicio',
              onPressed: () => Get.offAllNamed(Routes.home.name),
            ),
          ],
        ),
      ),
    );
  }

  void _showUserTurnsBottomSheet(BuildContext context) {
    final caseId = controller.currentCase.value?.uid;
    final turns =
        controller.messages
            .where((m) => !m.aiMessage && m.chatId == caseId)
            .toList();
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) {
        return SafeArea(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        'Intervenciones del usuario',
                        style: Theme.of(ctx).textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.bold,
                          color: Theme.of(ctx).primaryColor,
                        ),
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.close),
                      onPressed: () => Navigator.of(ctx).pop(),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                if (turns.isEmpty)
                  Text(
                    'Sin intervenciones registradas',
                    style: Theme.of(ctx).textTheme.bodySmall,
                  )
                else
                  Flexible(
                    child: ListView.separated(
                      shrinkWrap: true,
                      itemCount: turns.length,
                      separatorBuilder: (_, __) => const Divider(height: 16),
                      itemBuilder:
                          (_, i) => Text(
                            '${i + 1}. ${turns[i].text}',
                            textAlign: TextAlign.justify,
                            style: Theme.of(ctx).textTheme.bodyMedium,
                          ),
                    ),
                  ),
              ],
            ),
          ),
        );
      },
    );
  }
}
