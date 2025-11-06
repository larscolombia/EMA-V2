import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'animations/slide_in_left.dart';
import 'chat_markdown_wrapper.dart';

class ChatMessageAi extends StatefulWidget {
  final ChatMessageModel message;

  const ChatMessageAi({super.key, required this.message});

  @override
  State<ChatMessageAi> createState() => _ChatMessageAiState();
}

class _ChatMessageAiState extends State<ChatMessageAi>
    with SingleTickerProviderStateMixin {
  late AnimationController _controller;
  late Animation<double> _fadeAnimation;
  bool _isSourcesExpanded = false; // Estado del desplegable de fuentes

  // Formatea una cadena en estilo Título (conservando preposiciones comunes en minúscula)
  String _toTitleCase(String input) {
    if (input.trim().isEmpty) return input;
    final lower = input.trim().toLowerCase();
    const small = {
      'de',
      'del',
      'la',
      'las',
      'los',
      'y',
      'o',
      'u',
      'en',
      'para',
      'por',
      'con',
      'sin',
      'el',
      'al',
      'a',
      'un',
      'una',
      'unos',
      'unas',
      'the',
      'of',
      'and',
      'or',
      'in',
    };
    final parts = lower.split(RegExp(r"[\s_\-]+"));
    final buf = <String>[];
    for (var i = 0; i < parts.length; i++) {
      var w = parts[i];
      if (w.isEmpty) continue;
      if (i > 0 && small.contains(w)) {
        buf.add(w);
        continue;
      }
      // Capitalizar primera letra
      buf.add(w[0].toUpperCase() + (w.length > 1 ? w.substring(1) : ''));
    }
    return buf.join(' ');
  }

  // Determina el icono apropiado según el tipo de fuente
  IconData _getSourceIcon(String source) {
    final s = source.toLowerCase();
    if (s.contains('pmid')) {
      return Icons.science_outlined; // Artículo científico
    } else if (s.contains('.pdf') || s.contains('[pdf]')) {
      return Icons.picture_as_pdf_outlined; // PDF
    } else if (s.contains('libro') ||
        s.contains('manual') ||
        s.contains('schwartz') ||
        s.contains('harrison') ||
        s.contains('maingot')) {
      return Icons.menu_book_outlined; // Libro
    } else if (s.contains('http')) {
      return Icons.link_outlined; // URL
    } else {
      return Icons.article_outlined; // Genérico
    }
  }

  // Convierte una fuente genérica a un formato APA aproximado, sin inventar autores/revistas.
  // Reglas:
  // - "Título (PMID: 123456, 2023)" -> "Título. (2023). PubMed. https://pubmed.ncbi.nlm.nih.gov/123456/"
  // - "... .pdf" -> "Título del archivo. (s.f.). [PDF]."
  // - URL -> "Título (derivado del dominio). (s.f.). URL"
  // - Fallback: retorna la fuente original tal cual
  String _toApa(String source) {
    final s = source.trim();
    if (s.isEmpty) return s;

    // Patrón PMID con posible año
    final pmidRe = RegExp(
      r'^(.*?)\s*\(?PMID:?\s*(\d+)(?:,\s*(\d{4}))?\)?',
      caseSensitive: false,
    );
    final pm = pmidRe.firstMatch(s);
    if (pm != null) {
      final rawTitle = (pm.group(1) ?? '').trim().replaceAll(
        RegExp(r'["\]\[]'),
        '',
      );
      final title = rawTitle.isEmpty ? 'Artículo' : rawTitle;
      final pmid = pm.group(2)!;
      final year = (pm.group(3) ?? 's.f.');
      return "$title. ($year). PubMed. https://pubmed.ncbi.nlm.nih.gov/$pmid/";
    }

    // PDF filename
    if (s.toLowerCase().contains('.pdf')) {
      // Extraer nombre base del archivo
      final m = RegExp(r'([^/\\]+)\.pdf', caseSensitive: false).firstMatch(s);
      final base = (m != null ? m.group(1) : s).toString();
      final cleaned = base.replaceAll(RegExp(r'[_\-]+'), ' ');
      final title = _toTitleCase(cleaned);
      return "$title. (s.f.). [PDF].";
    }

    // URL genérica
    if (s.startsWith('http://') || s.startsWith('https://')) {
      return "(s.f.). $s"; // Sin título fiable, dejamos URL directa
    }

    // Heurística para Libros/Manuales médicos comunes
    // Detectar patrones de títulos de libros médicos conocidos
    final bookPatterns = {
      r"harrison'?s?\s+principles?\s+of\s+internal\s+medicine":
          "Harrison's Principles of Internal Medicine",
      r"braunwald'?s?\s+heart\s+disease": "Braunwald's Heart Disease",
      r"robbins\s+and\s+cotran\s+pathologic\s+basis":
          "Robbins and Cotran Pathologic Basis of Disease",
      r"nelson\s+textbook\s+of\s+pediatrics": "Nelson Textbook of Pediatrics",
      r"williams\s+obstetrics": "Williams Obstetrics",
      r"gray'?s?\s+anatomy": "Gray's Anatomy",
      r"guyton\s+and\s+hall\s+textbook":
          "Guyton and Hall Textbook of Medical Physiology",
      r"cecil\s+textbook\s+of\s+medicine": "Cecil Textbook of Medicine",
      r"sabiston\s+textbook\s+of\s+surgery": "Sabiston Textbook of Surgery",
      r"principles?\s+of\s+internal\s+medicine":
          "Principles of Internal Medicine",
      r"principles?\s+of\s+surgery": "Principles of Surgery",
      r"tratado\s+de\s+cardiolog[íi]a": "Tratado de Cardiología",
      r"manual\s+de\s+medicina\s+interna": "Manual de Medicina Interna",
    };

    final lowerS = s.toLowerCase();
    for (final pattern in bookPatterns.entries) {
      if (RegExp(pattern.key, caseSensitive: false).hasMatch(lowerS)) {
        // Formato APA para libro: Título. (s.f.). Editorial desconocida.
        return "${pattern.value}. (s.f.). [Libro de texto médico].";
      }
    }

    // Heurística genérica: tratar como Libro/Manual si no es PMID/URL/PDF
    // Condiciones: contiene al menos un espacio (2+ palabras), no contiene dígitos ni 'doi' ni 'pmid'
    final isLikelyBook =
        s.contains(' ') &&
        !RegExp(r'\d').hasMatch(s) &&
        !s.toLowerCase().contains('doi') &&
        !s.toLowerCase().contains('pmid') &&
        !s.toLowerCase().contains('http');
    if (isLikelyBook) {
      // Normalizar título: capitalizar correctamente
      final title = _toTitleCase(s.replaceAll(RegExp(r'\s+'), ' ').trim());
      // Formato APA para libro sin más metadatos: Título. (s.f.). [Tipo de documento].
      return "$title. (s.f.). [Libro/Manual médico].";
    }

    // Fallback: si no podemos transformar, agregamos punto final si falta
    if (!s.endsWith('.')) return "$s.";
    return s;
  }

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      duration: const Duration(milliseconds: 800), // Duración de la animación
      vsync: this,
    );

    _fadeAnimation = CurvedAnimation(
      parent: _controller,
      curve: Curves.easeInOut,
    );

    Future.delayed(const Duration(milliseconds: 200), () {
      if (mounted) _controller.forward();
    });
  }

  List<String> _extractSources(String text) {
    final List<String> sources = [];

    // 1. Buscar sección con ## Fuentes (con cualquier texto adicional como OBLIGATORIO, AL FINAL, etc.)
    // Captura TODO desde ## Fuentes hasta el final del texto completo
    final newFormatRegex = RegExp(
      r'##\s*Fuentes?[^#\n]*\n+([\s\S]*?)$',
      caseSensitive: false,
    );
    final newFormatMatch = newFormatRegex.firstMatch(text);

    String? sourcesSection;

    if (newFormatMatch != null) {
      sourcesSection = newFormatMatch.group(1) ?? '';
      print('📖 [_extractSources] Encontrado header "## Fuentes"');
    } else {
      print(
        '📖 [_extractSources] NO encontrado header "## Fuentes", buscando subsecciones...',
      );

      // 1.B. Caso SIN header "## Fuentes" pero CON subsecciones directas (### 📚 Libros, ### 🔬 PubMed)
      // Captura desde la primera subsección de bibliografía hasta el final
      final directSubsectionsRegex = RegExp(
        r'###\s*[📚🔬📖🔍]\s*(?:Libros?|PubMed|Literatura|Artículos?)',
        caseSensitive: false,
      );

      print(
        '📖 [_extractSources] Buscando patrón: ###\\s*[📚🔬📖🔍]\\s*(?:Libros?|PubMed|...)',
      );

      final directMatch = directSubsectionsRegex.firstMatch(text);

      if (directMatch != null) {
        // Capturar desde el inicio de la primera subsección hasta el final
        final matchStart = directMatch.start;
        sourcesSection = text.substring(matchStart);
        print(
          '📖 [_extractSources] ✅ Encontradas subsecciones directas sin header "Fuentes"',
        );
        print(
          '📖 [_extractSources] matchStart: $matchStart, totalLength: ${text.length}',
        );
      } else {
        print('📖 [_extractSources] ❌ NO encontradas subsecciones directas');

        // 1.C. NUEVO: Detectar fuentes ya formateadas (sin subsecciones ###)
        // Buscar líneas con ** al inicio seguidas de contenido que parece fuente (PMID, Ed., s.f., etc.)
        // Estas líneas aparecen después del contenido principal
        final formattedSourcesRegex = RegExp(
          r'\n\*\*[^*]+\*\*[^\n]*(?:PMID|Ed\.|s\.f\.|—)',
          multiLine: true,
        );

        final formattedMatches =
            formattedSourcesRegex.allMatches(text).toList();

        if (formattedMatches.isNotEmpty) {
          print(
            '📖 [_extractSources] ✅ Encontradas ${formattedMatches.length} fuentes ya formateadas',
          );
          // Capturar desde la primera fuente formateada hasta el final
          final firstMatchStart = formattedMatches.first.start;
          sourcesSection = text.substring(firstMatchStart);
          print(
            '📖 [_extractSources] matchStart: $firstMatchStart, totalLength: ${text.length}',
          );
        } else {
          // Debug: mostrar últimos 200 chars para ver qué hay
          final preview = text.substring(
            text.length > 200 ? text.length - 200 : 0,
          );
          print('📖 [_extractSources] Últimos 200 chars del texto:');
          print(preview.replaceAll('\n', '⏎\n'));
        }
      }
    }

    if (sourcesSection != null) {
      // Debug: mostrar la sección completa capturada
      print(
        '📖 [_extractSources] Sección capturada (${sourcesSection.length} chars)',
      );
      print('📖 Contenido: ${sourcesSection.replaceAll('\n', '⏎\n')}');

      // Extraer líneas que empiezan con - o * o ** (Markdown bullet/bold)
      // El patrón captura: - texto, * texto, ** texto (negritas usadas como bullets)
      final listRegex = RegExp(
        r'^\s*(?:[\-\*]+)\s*\*?\s*(.+?)\.?\s*$',
        multiLine: true,
      );
      final allMatches = listRegex.allMatches(sourcesSection).toList();

      print('📖 Total de líneas con - o * o **: ${allMatches.length}');

      for (var i = 0; i < allMatches.length; i++) {
        var content = allMatches[i].group(1)!.trim();

        // Limpiar asteriscos finales de negritas (**texto**)
        content = content.replaceAll(RegExp(r'\*+$'), '').trim();

        print('📖 Línea $i: "$content"');

        // Solo excluir si la línea es EXACTAMENTE un encabezado de subsección
        // Ejemplos a excluir: "📚 Libros", "🔬 PubMed", "📚 Literatura"
        final isSubsectionHeader = RegExp(
          r'^#{0,3}\s*[📚🔬📖🔍]?\s*(?:Libros?|PubMed|Literatura|Artículos?)\s*$',
          caseSensitive: false,
        ).hasMatch(content);

        if (content.isNotEmpty && !isSubsectionHeader && content.length > 5) {
          sources.add(content);
          print('✅ Añadida: "$content"');
        } else if (isSubsectionHeader) {
          print('⏭️ Ignorada (encabezado): "$content"');
        } else {
          print('⏭️ Ignorada (muy corta o vacía): "$content"');
        }
      }
    }

    // 2. Fallback: Buscar "Fuentes" o "Fuente" con cualquier sufijo/prefijo (: OBLIGATORIO, AL FINAL, etc.)
    if (sources.isEmpty) {
      final sourcesRegex = RegExp(
        r'##\s*Fuentes?[^#\n]*\n(.*?)(?=\n\n|\n## |$)',
        dotAll: true,
        caseSensitive: false,
      );
      final sourcesMatch = sourcesRegex.firstMatch(text);

      if (sourcesMatch != null) {
        final sourcesText = sourcesMatch.group(1) ?? '';
        final listRegex = RegExp(r'^[\-\*]\s*(.+)$', multiLine: true);
        sources.addAll(
          listRegex
              .allMatches(sourcesText)
              .map((match) => match.group(1)!.trim())
              .where((s) => s.isNotEmpty),
        );
      }
    }

    // Fallback: buscar patrón "Fuentes" sin ## pero con cualquier sufijo
    if (sources.isEmpty) {
      final altSourceRegex1 = RegExp(
        r'Fuentes?[^\n]*\n(.*?)(?=\n\n|\n## |$)',
        dotAll: true,
        caseSensitive: false,
      );
      final altMatch = altSourceRegex1.firstMatch(text);
      if (altMatch != null) {
        final sourcesText = altMatch.group(1) ?? '';
        final listRegex = RegExp(r'^[\-\*]\s*(.+)$', multiLine: true);
        sources.addAll(
          listRegex
              .allMatches(sourcesText)
              .map((match) => match.group(1)!.trim())
              .where((s) => s.isNotEmpty),
        );
      }
    }

    // Fallback 2: Buscar "Fuente:" o "Fuentes:" en línea única con cualquier sufijo
    if (sources.isEmpty) {
      final altSourceRegex2 = RegExp(
        r'Fuentes?[^\n]*:\s*(.+?)(?=\n|$)',
        multiLine: true,
        caseSensitive: false,
      );
      sources.addAll(
        altSourceRegex2
            .allMatches(text)
            .map((match) => match.group(1)!.trim())
            .where((s) => s.isNotEmpty),
      );
    }

    return sources;
  }

  String _getMainContent(String text, {bool shouldRemoveSources = true}) {
    if (text.trim().isEmpty) return text;

    String content = text;

    // Solo remover fuentes si fueron detectadas exitosamente
    if (shouldRemoveSources) {
      // 1. Remover sección "## Fuentes" con cualquier sufijo (OBLIGATORIO, AL FINAL, etc.)
      // Este patrón captura desde ## Fuentes hasta el final (incluye subsecciones con emojis)
      content = content.replaceAll(
        RegExp(r'\n*##\s*Fuentes?[^#\n]*\n+[\s\S]*$', caseSensitive: false),
        '',
      );

      // 1.B. Remover subsecciones directas sin header "Fuentes" (### 📚 Libros, ### 🔬 PubMed)
      content = content.replaceAll(
        RegExp(
          r'\n*###\s*[📚🔬📖🔍]\s*(?:Libros?|PubMed|Literatura|Artículos?)\s*\n+[\s\S]*$',
          caseSensitive: false,
        ),
        '',
      );

      // 1.C. Remover fuentes ya formateadas (líneas con ** seguidas de PMID, Ed., s.f., etc.)
      // Buscar la primera ocurrencia y remover desde ahí hasta el final
      final formattedSourcesRegex = RegExp(
        r'\n+\*\*[^*]+\*\*[^\n]*(?:PMID|Ed\.|s\.f\.|—)',
        multiLine: true,
      );
      final firstMatch = formattedSourcesRegex.firstMatch(content);
      if (firstMatch != null) {
        content = content.substring(0, firstMatch.start);
      }

      // 2. Fallback: Remover "Fuentes" sin ## (casos legacy)
      content = content.replaceAll(
        RegExp(r'\n*Fuentes?[^\n]*\n[\s\S]*$', caseSensitive: false),
        '',
      );
    }

    // Limpiar espacios en blanco excesivos al final (siempre)
    content = content.replaceAll(RegExp(r'\s+$'), '');

    return content.trim();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final String timeStr = DateFormat('HH:mm').format(widget.message.createdAt);
    final isLongMessage = widget.message.text.length > 800;

    // Si no hay texto útil, no renderizar la burbuja
    if (widget.message.text.trim().isEmpty) {
      return const SizedBox.shrink();
    }

    // 🔍 DEBUG: Ver el texto RAW antes de procesar
    print('🔍 ==================== DEBUG FUENTES ====================');
    print('🔍 Texto RAW (primeros 800 chars):');
    print(
      widget.message.text.substring(
        0,
        widget.message.text.length > 800 ? 800 : widget.message.text.length,
      ),
    );
    print('🔍 ');
    print('🔍 Texto RAW (últimos 800 chars):');
    final startIdx =
        widget.message.text.length > 800 ? widget.message.text.length - 800 : 0;
    print(widget.message.text.substring(startIdx));
    print('🔍 ');
    print('🔍 Contiene "\\n" literal: ${widget.message.text.contains(r'\n')}');
    print(
      '🔍 Contiene "Fuentes": ${widget.message.text.toLowerCase().contains('fuente')}',
    );
    print('🔍 ======================================================');

    // Extraer fuentes una sola vez para coordinar con _getMainContent
    final extractedSources = _extractSources(widget.message.text);
    final hasValidSources = extractedSources.isNotEmpty;

    print('🔍 Fuentes extraídas: ${extractedSources.length}');
    if (extractedSources.isNotEmpty) {
      for (var i = 0; i < extractedSources.length; i++) {
        print('🔍   [$i]: ${extractedSources[i]}');
      }
    }
    print('🔍 ======================================================');

    // DEBUG: Advertir si el texto contiene "Fuentes" pero no se detectaron
    if (!hasValidSources &&
        widget.message.text.toLowerCase().contains('fuente')) {
      print(
        '⚠️ [ChatMessageAi] Texto contiene "Fuente" pero no se detectaron fuentes',
      );
      print(
        '📄 Preview (primeros 500 chars): ${widget.message.text.substring(0, widget.message.text.length > 500 ? 500 : widget.message.text.length)}',
      );
    }

    return Align(
      alignment: Alignment.centerLeft,
      child: SlideInLeft(
        child: FadeTransition(
          opacity: _fadeAnimation,
          child: Container(
            margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            width: double.infinity,
            decoration: BoxDecoration(
              // 🎨 Cambiar a blanco para mensajes largos
              color:
                  isLongMessage
                      ? Colors.white
                      : const Color.fromRGBO(58, 12, 140, 0.9),
              borderRadius: BorderRadius.circular(16),
              // 🎨 Añadir sombra sutil para mensajes largos
              boxShadow:
                  isLongMessage
                      ? [
                        BoxShadow(
                          color: Colors.black.withOpacity(0.08),
                          blurRadius: 12,
                          offset: const Offset(0, 4),
                        ),
                      ]
                      : null,
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // 📋 Header para mensajes largos
                if (isLongMessage)
                  Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 20,
                      vertical: 14,
                    ),
                    decoration: BoxDecoration(
                      color: const Color.fromRGBO(58, 12, 140, 0.05),
                      borderRadius: const BorderRadius.only(
                        topLeft: Radius.circular(16),
                        topRight: Radius.circular(16),
                      ),
                    ),
                    child: Row(
                      children: [
                        Icon(
                          Icons.article_outlined,
                          color: const Color.fromRGBO(58, 12, 140, 1),
                          size: 20,
                        ),
                        const SizedBox(width: 10),
                        Text(
                          'Respuesta Detallada',
                          style: TextStyle(
                            color: const Color.fromRGBO(58, 12, 140, 1),
                            fontSize: 14,
                            fontWeight: FontWeight.w600,
                            letterSpacing: 0.3,
                          ),
                        ),
                      ],
                    ),
                  ),

                // Contenido principal
                Padding(
                  padding: EdgeInsets.all(isLongMessage ? 20 : 16),
                  child: ChatMarkdownWrapper(
                    text: _getMainContent(
                      widget.message.text,
                      shouldRemoveSources: hasValidSources,
                    ),
                    style: TextStyle(
                      fontSize: isLongMessage ? 15.5 : 15,
                      // 🎨 Texto negro para fondo blanco, blanco para morado
                      color: isLongMessage ? Colors.black87 : Colors.white,
                      height: 1.7,
                      letterSpacing: 0.2,
                    ),
                  ),
                ),

                // 📚 Mostrar fuentes si están disponibles (desplegable)
                if (hasValidSources)
                  Container(
                    margin: EdgeInsets.only(
                      left: isLongMessage ? 16 : 12,
                      right: isLongMessage ? 16 : 12,
                      bottom: 16,
                      top: 8,
                    ),
                    decoration: BoxDecoration(
                      // 🎨 Fondo adaptable: gris claro sobre blanco, blanco transparente sobre morado
                      color:
                          isLongMessage
                              ? const Color.fromRGBO(58, 12, 140, 0.04)
                              : Colors.white.withOpacity(0.08),
                      borderRadius: BorderRadius.circular(12),
                      border: Border.all(
                        color:
                            isLongMessage
                                ? const Color.fromRGBO(58, 12, 140, 0.15)
                                : Colors.white.withOpacity(0.15),
                        width: 1,
                      ),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        // Header clickeable para expandir/contraer
                        InkWell(
                          onTap: () {
                            setState(() {
                              _isSourcesExpanded = !_isSourcesExpanded;
                            });
                          },
                          borderRadius: BorderRadius.circular(12),
                          child: Padding(
                            padding: const EdgeInsets.all(16),
                            child: Row(
                              children: [
                                Container(
                                  padding: const EdgeInsets.all(6),
                                  decoration: BoxDecoration(
                                    color:
                                        isLongMessage
                                            ? const Color.fromRGBO(
                                              58,
                                              12,
                                              140,
                                              0.1,
                                            )
                                            : Colors.white.withOpacity(0.1),
                                    borderRadius: BorderRadius.circular(6),
                                  ),
                                  child: Icon(
                                    Icons.menu_book_rounded,
                                    color:
                                        isLongMessage
                                            ? const Color.fromRGBO(
                                              58,
                                              12,
                                              140,
                                              1,
                                            )
                                            : Colors.white,
                                    size: 18,
                                  ),
                                ),
                                const SizedBox(width: 10),
                                Text(
                                  'Bibliografía',
                                  style: TextStyle(
                                    color:
                                        isLongMessage
                                            ? const Color.fromRGBO(
                                              58,
                                              12,
                                              140,
                                              1,
                                            )
                                            : Colors.white,
                                    fontSize: 15,
                                    fontWeight: FontWeight.bold,
                                    letterSpacing: 0.3,
                                  ),
                                ),
                                const SizedBox(width: 8),
                                // Badge con contador
                                Container(
                                  padding: const EdgeInsets.symmetric(
                                    horizontal: 8,
                                    vertical: 4,
                                  ),
                                  decoration: BoxDecoration(
                                    color:
                                        isLongMessage
                                            ? const Color.fromRGBO(
                                              58,
                                              12,
                                              140,
                                              1,
                                            )
                                            : Colors.white,
                                    borderRadius: BorderRadius.circular(12),
                                  ),
                                  child: Text(
                                    '${extractedSources.length}',
                                    style: TextStyle(
                                      color:
                                          isLongMessage
                                              ? Colors.white
                                              : const Color.fromRGBO(
                                                58,
                                                12,
                                                140,
                                                1,
                                              ),
                                      fontSize: 12,
                                      fontWeight: FontWeight.bold,
                                    ),
                                  ),
                                ),
                                const Spacer(),
                                // Icono de expansión
                                Icon(
                                  _isSourcesExpanded
                                      ? Icons.keyboard_arrow_up
                                      : Icons.keyboard_arrow_down,
                                  color:
                                      isLongMessage
                                          ? const Color.fromRGBO(
                                            58,
                                            12,
                                            140,
                                            0.6,
                                          )
                                          : Colors.white60,
                                  size: 24,
                                ),
                              ],
                            ),
                          ),
                        ),
                        // Contenido desplegable
                        if (_isSourcesExpanded) ...[
                          // Separador
                          Padding(
                            padding: const EdgeInsets.symmetric(horizontal: 16),
                            child: Container(
                              height: 1,
                              color:
                                  isLongMessage
                                      ? const Color.fromRGBO(58, 12, 140, 0.1)
                                      : Colors.white.withOpacity(0.1),
                            ),
                          ),
                          const SizedBox(height: 12),
                          // Lista de fuentes
                          Padding(
                            padding: const EdgeInsets.only(
                              left: 16,
                              right: 16,
                              bottom: 16,
                            ),
                            child: Column(
                              children:
                                  extractedSources
                                      .map(
                                        (source) => Container(
                                          margin: const EdgeInsets.only(
                                            bottom: 10,
                                          ),
                                          padding: const EdgeInsets.all(12),
                                          decoration: BoxDecoration(
                                            color:
                                                isLongMessage
                                                    ? Colors.black.withOpacity(
                                                      0.03,
                                                    )
                                                    : Colors.black.withOpacity(
                                                      0.15,
                                                    ),
                                            borderRadius: BorderRadius.circular(
                                              8,
                                            ),
                                            border: Border.all(
                                              color:
                                                  isLongMessage
                                                      ? const Color.fromRGBO(
                                                        58,
                                                        12,
                                                        140,
                                                        0.08,
                                                      )
                                                      : Colors.white
                                                          .withOpacity(0.08),
                                              width: 0.5,
                                            ),
                                          ),
                                          child: Row(
                                            crossAxisAlignment:
                                                CrossAxisAlignment.start,
                                            children: [
                                              // Icono según tipo de fuente
                                              Padding(
                                                padding: const EdgeInsets.only(
                                                  top: 2,
                                                  right: 10,
                                                ),
                                                child: Icon(
                                                  _getSourceIcon(source),
                                                  color:
                                                      isLongMessage
                                                          ? const Color.fromRGBO(
                                                            58,
                                                            12,
                                                            140,
                                                            0.6,
                                                          )
                                                          : Colors.white60,
                                                  size: 16,
                                                ),
                                              ),
                                              // Contenido de la fuente
                                              Expanded(
                                                child: Text(
                                                  _toApa(source),
                                                  style: TextStyle(
                                                    color:
                                                        isLongMessage
                                                            ? Colors.black87
                                                            : Colors.white
                                                                .withOpacity(
                                                                  0.85,
                                                                ),
                                                    fontSize: 12,
                                                    height: 1.5,
                                                    letterSpacing: 0.2,
                                                  ),
                                                ),
                                              ),
                                            ],
                                          ),
                                        ),
                                      )
                                      .toList(),
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),

                // ⏰ Muestra la hora del mensaje
                Padding(
                  padding: EdgeInsets.only(
                    right: isLongMessage ? 20 : 8,
                    bottom: isLongMessage ? 8 : 4,
                  ),
                  child: Align(
                    alignment: Alignment.bottomRight,
                    child: Text(
                      timeStr,
                      style: TextStyle(
                        color:
                            isLongMessage
                                ? Colors.black45
                                : const Color.fromRGBO(255, 255, 255, 0.7),
                        fontSize: 10,
                      ),
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
}
