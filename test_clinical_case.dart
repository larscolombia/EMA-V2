import 'package:flutter/material.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/widgets/chat_markdown_wrapper.dart';

void main() {
  const testResponse =
      '''1. Diagnóstico clínico y métodos: El diagnóstico de gastritis aguda usualmente se basa en la anamnesis, los síntomas y los hallazgos endoscópicos. Los síntomas incluyen dolor epigástrico, náuseas, vómitos y, en casos graves, hemorragia digestiva alta. 

2. Diagnóstico diferencial: Es importante diferenciar la gastritis aguda de úlcera péptica, dispepsia funcional y otras causas de dolor epigástrico o sangrado. 

3. Estudios recientes: No existen biomarcadores ampliamente aceptados para el diagnóstico no invasivo; la endoscopía sigue siendo el estándar de oro. Se recomienda endoscopía temprana en pacientes con sangrado activo, factores de riesgo o síntomas graves.

Referencias:
- Kamboj AK, Cotter TG, Oxentenko AS. Acute and Chronic Gastritis. Gastroenterol Clin North Am. 2021 Mar;50(1):99-114. PMID: 33242692.
- Sugano K. Current status and future perspectives of Helicobacter pylori infection-related gastroduodenal diseases. Digestion. 2019;100(1):26-34. PMID: 31397982.

¿Te gustaría información sobre nuevas técnicas diagnósticas o recomendaciones clínicas específicas?

*(Fuente: PubMed)*

**Referencias:**
- - La endoscopía digestiva alta es el método diagnóstico de elección para identificar cambios inflamatorios agudos como eritema, edema, erosiones o sangrado en la mucosa gástrica【PMID: 33242692】.
- - La biopsia gástrica puede confirmar la inflamación aguda y descartar otras causas (infección por Helicobacter pylori, neoplasias, etc.)【PMID: 31397982】.
- - Es importante diferenciar la gastritis aguda de úlcera péptica, dispepsia funcional y otras causas de dolor epigástrico o sangrado【PMID: 33242692】.
- - No existen biomarcadores ampliamente aceptados para el diagnóstico no invasivo; la endoscopía sigue siendo el estándar de oro【PMID: 31397982】.
- - Kamboj AK, Cotter TG, Oxentenko AS. Acute and Chronic Gastritis: Approach to and Histopathology of Inflammatory Disorders of the Stomach. Gastroenterol Clin North Am. 2021 Mar;50(1):99-114. doi: 10.1016/j.gtc.2020.09.007. PMID: 33242692.''';

  runApp(
    MaterialApp(
      home: Scaffold(
        appBar: AppBar(
          title: const Text('Test - Referencias Duplicadas Limpiadas'),
          backgroundColor: Colors.purple,
          foregroundColor: Colors.white,
        ),
        body: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                'ANTES (Referencias duplicadas):',
                style: TextStyle(
                  fontWeight: FontWeight.bold,
                  color: Colors.red,
                ),
              ),
              const SizedBox(height: 8),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  border: Border.all(color: Colors.red),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  testResponse,
                  style: const TextStyle(fontSize: 12, fontFamily: 'monospace'),
                ),
              ),
              const SizedBox(height: 24),
              const Text(
                'DESPUÉS (Referencias consolidadas):',
                style: TextStyle(
                  fontWeight: FontWeight.bold,
                  color: Colors.green,
                ),
              ),
              const SizedBox(height: 8),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  border: Border.all(color: Colors.green),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: ChatMarkdownWrapper(
                  text: testResponse,
                  style: const TextStyle(
                    fontSize: 16,
                    color: Colors.black87,
                    height: 1.6,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    ),
  );
}
