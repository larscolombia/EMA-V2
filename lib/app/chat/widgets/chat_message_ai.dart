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

    // 1. Buscar sección nueva con emojis: ## Fuentes seguido de ### 📚 y ### 🔬
    final newFormatRegex = RegExp(
      r'##\s*Fuentes\s*\n+(.*?)(?=\n\n[^#\-\*]|\z)',
      dotAll: true,
      caseSensitive: false,
    );
    final newFormatMatch = newFormatRegex.firstMatch(text);

    if (newFormatMatch != null) {
      final sourcesSection = newFormatMatch.group(1) ?? '';
      // Extraer líneas que empiezan con - (lista de fuentes)
      final listRegex = RegExp(r'^\s*[\-\*]\s*(.+)$', multiLine: true);
      sources.addAll(
        listRegex
            .allMatches(sourcesSection)
            .map((match) => match.group(1)!.trim())
            .where((s) => s.isNotEmpty && !s.contains('###')),
      );
    }

    // 2. Fallback: Buscar formato antiguo ## Fuentes sin subsecciones
    if (sources.isEmpty) {
      final sourcesRegex = RegExp(
        r'##\s*Fuentes\s*:?\s*\n(.*?)(?=\n\n|\n## |$)',
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

    // Fallback: buscar patrón "Fuentes:" sin ##
    if (sources.isEmpty) {
      final altSourceRegex1 = RegExp(
        r'Fuentes?:\s*\n(.*?)(?=\n\n|\n## |$)',
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

    // Fallback 2: Buscar "Fuente:" o "Fuentes:" en línea única
    if (sources.isEmpty) {
      final altSourceRegex2 = RegExp(
        r'Fuentes?:\s*(.+?)(?=\n|$)',
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

  String _getMainContent(String text) {
    if (text.trim().isEmpty) return text;

    String content = text;

    // 1. Remover sección "## Fuentes" completa incluyendo subsecciones con emojis
    // Este patrón captura desde ## Fuentes hasta el final del texto
    content = content.replaceAll(
      RegExp(
        r'\n*##\s*Fuentes\s*:?\s*\n+(?:###\s*[📚🔬].*?\n+)?[\s\S]*$',
        caseSensitive: false,
      ),
      '',
    );

    // 2. Remover sección "Fuentes:" sin ## (fallback para formatos antiguos)
    content = content.replaceAll(
      RegExp(r'\n*Fuentes?\s*:?\s*\n[\s\S]*$', caseSensitive: false),
      '',
    );

    // 3. Remover líneas sueltas que empiezan con "- " y contienen referencias bibliográficas
    // (Manual, PMID, PDF, etc.) que puedan haber quedado
    content = content.replaceAll(
      RegExp(
        r'\n\s*-\s+.*?(?:Manual\s+Schwartz|PMID:\s*\d+|\.pdf|PubMed|Libro de texto|Literatura Científica).*?$',
        caseSensitive: false,
        multiLine: true,
      ),
      '',
    );

    // 4. Remover subsecciones de fuentes sueltas que quedaron
    content = content.replaceAll(
      RegExp(
        r'###\s*[📚🔬].*?(?:Libros|Literatura).*?\n',
        caseSensitive: false,
      ),
      '',
    );

    // 5. Remover símbolos ## mal formados al final
    content = content.replaceAll(RegExp(r'\.?\s*##\s*[A-Z]?\s*$'), '.');

    // 6. Limpiar espacios en blanco excesivos al final
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
                    text: _getMainContent(widget.message.text),
                    style: TextStyle(
                      fontSize: isLongMessage ? 15.5 : 15,
                      // 🎨 Texto negro para fondo blanco, blanco para morado
                      color: isLongMessage ? Colors.black87 : Colors.white,
                      height: 1.7,
                      letterSpacing: 0.2,
                    ),
                  ),
                ),

                // 📚 Mostrar fuentes si están disponibles
                if (_extractSources(widget.message.text).isNotEmpty)
                  Container(
                    margin: EdgeInsets.only(
                      left: isLongMessage ? 16 : 12,
                      right: isLongMessage ? 16 : 12,
                      bottom: 16,
                      top: 8,
                    ),
                    padding: const EdgeInsets.all(16),
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
                        // Header de fuentes
                        Row(
                          children: [
                            Container(
                              padding: const EdgeInsets.all(6),
                              decoration: BoxDecoration(
                                color:
                                    isLongMessage
                                        ? const Color.fromRGBO(58, 12, 140, 0.1)
                                        : Colors.white.withOpacity(0.1),
                                borderRadius: BorderRadius.circular(6),
                              ),
                              child: Icon(
                                Icons.menu_book_rounded,
                                color:
                                    isLongMessage
                                        ? const Color.fromRGBO(58, 12, 140, 1)
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
                                        ? const Color.fromRGBO(58, 12, 140, 1)
                                        : Colors.white,
                                fontSize: 15,
                                fontWeight: FontWeight.bold,
                                letterSpacing: 0.3,
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 12),
                        // Separador
                        Container(
                          height: 1,
                          color:
                              isLongMessage
                                  ? const Color.fromRGBO(58, 12, 140, 0.1)
                                  : Colors.white.withOpacity(0.1),
                        ),
                        const SizedBox(height: 12),
                        // Lista de fuentes
                        ..._extractSources(widget.message.text).map(
                          (source) => Container(
                            margin: const EdgeInsets.only(bottom: 10),
                            padding: const EdgeInsets.all(12),
                            decoration: BoxDecoration(
                              color:
                                  isLongMessage
                                      ? Colors.black.withOpacity(0.03)
                                      : Colors.black.withOpacity(0.15),
                              borderRadius: BorderRadius.circular(8),
                              border: Border.all(
                                color:
                                    isLongMessage
                                        ? const Color.fromRGBO(
                                          58,
                                          12,
                                          140,
                                          0.08,
                                        )
                                        : Colors.white.withOpacity(0.08),
                                width: 0.5,
                              ),
                            ),
                            child: Row(
                              crossAxisAlignment: CrossAxisAlignment.start,
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
                                              : Colors.white.withOpacity(0.85),
                                      fontSize: 12,
                                      height: 1.5,
                                      letterSpacing: 0.2,
                                    ),
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
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
