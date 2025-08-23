import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/common/layouts/app_layout.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/show_error_widget.dart';
import 'package:ema_educacion_medica_avanzada/config/routes/app_pages.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gpt_markdown/gpt_markdown.dart';

/// Vista de evaluación final para casos clínicos analíticos.
/// Muestra un preloader mientras se genera la evaluación y luego
/// el contenido en formato markdown bonito (similar a feedback de quizzes).
class ClinicalCaseEvaluationView extends StatefulWidget {
  const ClinicalCaseEvaluationView({super.key});

  @override
  State<ClinicalCaseEvaluationView> createState() => _ClinicalCaseEvaluationViewState();
}

class _ClinicalCaseEvaluationViewState extends State<ClinicalCaseEvaluationView> {
  final controller = Get.find<ClinicalCaseController>();
  bool showUserTurns = false;

  ChatMessageModel? _evaluationMessage() {
    // Busca el último mensaje AI después de marcar evaluationGenerated
    final msgs = controller.messages.where((m) => m.aiMessage).toList();
    if (msgs.isEmpty) return null;
    // Heurísticas distintas para cada tipo
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
      // Interactivo: buscar encabezado 'Evaluación final interactiva'
      for (final m in msgs.reversed) {
        final lower = m.text.toLowerCase();
        if (lower.contains('evaluación final interactiva')) {
          return m;
        }
      }
      return msgs.last;
    }
  }

  @override
  void initState() {
    super.initState();
    // Si aún no se generó, forzar generación (por si navegación manual)
    if (!controller.evaluationGenerated.value && controller.currentCase.value?.type == ClinicalCaseType.analytical) {
      controller.generateFinalEvaluation();
    }
  }

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
  final isAnalytical = caseModel.type == ClinicalCaseType.analytical;
  final interactiveDone = controller.interactiveEvaluationGenerated.value;
  final analyticalDone = controller.evaluationGenerated.value;
  final evalReady = isAnalytical ? analyticalDone : interactiveDone;
  if (!evalReady) {
          return const StateMessageWidget(
            message: 'Generando evaluación...',
            type: StateMessageType.download,
            showLoading: true,
          );
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
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                controller.currentCase.value?.type == ClinicalCaseType.interactive
                    ? 'Evaluación final interactiva'
                    : 'Evaluación final analítica',
                style: Theme.of(context).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 12),
              Text(caseModel.anamnesis, style: Theme.of(context).textTheme.bodySmall, maxLines: 3, overflow: TextOverflow.ellipsis),
              const Divider(height: 32),
              GptMarkdown(evalMsg.text),
              const SizedBox(height: 32),
              AnimatedCrossFade(
                firstChild: const SizedBox.shrink(),
                secondChild: _buildUserTurns(context),
                crossFadeState: showUserTurns ? CrossFadeState.showSecond : CrossFadeState.showFirst,
                duration: const Duration(milliseconds: 250),
              ),
              const SizedBox(height: 16),
              Row(
                children: [
                  OutlinedButton.icon(
                    onPressed: () => setState(() => showUserTurns = !showUserTurns),
                    icon: Icon(showUserTurns ? Icons.visibility_off : Icons.visibility),
                    label: Text(showUserTurns ? 'Ocultar intervenciones' : 'Ver intervenciones'),
                  ),
                  const SizedBox(width: 12),
                  FilledButton.icon(
                    onPressed: () => Get.offAllNamed(Routes.home.name),
                    icon: const Icon(Icons.home),
                    label: const Text('Inicio'),
                  ),
                ],
              ),
            ],
          ),
        );
      }),
    );
  }

  Widget _buildUserTurns(BuildContext context) {
    final caseId = controller.currentCase.value?.uid;
    final turns = controller.messages.where((m) => !m.aiMessage && m.chatId == caseId).toList();
    if (turns.isEmpty) {
      return Text('Sin intervenciones de usuario registradas', style: Theme.of(context).textTheme.bodySmall);
    }
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceVariant.withOpacity(0.3),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Intervenciones del usuario', style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
          const SizedBox(height: 8),
          ...List.generate(turns.length, (i) {
            final t = turns[i];
            return Padding(
              padding: const EdgeInsets.only(bottom: 6),
              child: Text('${i + 1}. ${t.text}', style: Theme.of(context).textTheme.bodySmall),
            );
          }),
        ],
      ),
    );
  }
}
