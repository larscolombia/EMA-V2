import 'dart:async';
import 'package:flutter/material.dart';

class ChatTypingIndicator extends StatefulWidget {
  const ChatTypingIndicator({
    super.key,
    this.captions,
    this.captionInterval = const Duration(seconds: 2),
  });

  // Optional rotating captions to display next to the dots
  final List<String>? captions;
  // Interval to switch captions
  final Duration captionInterval;

  @override
  State<ChatTypingIndicator> createState() => _ChatTypingIndicatorState();
}

class _ChatTypingIndicatorState extends State<ChatTypingIndicator>
    with SingleTickerProviderStateMixin {
  late AnimationController _controller;
  Timer? _captionTimer;
  int _captionIndex = 0;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1000),
    )..repeat();

    _maybeStartCaptionTimer();
  }

  @override
  void dispose() {
    _captionTimer?.cancel();
    _controller.dispose();
    super.dispose();
  }

  @override
  void didUpdateWidget(covariant ChatTypingIndicator oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.captions != widget.captions ||
        oldWidget.captionInterval != widget.captionInterval) {
      _captionTimer?.cancel();
      _captionIndex = 0;
      _maybeStartCaptionTimer();
    }
  }

  void _maybeStartCaptionTimer() {
    if (widget.captions == null || widget.captions!.isEmpty) return;
    _captionTimer = Timer.periodic(widget.captionInterval, (_) {
      if (!mounted) return;
      setState(() {
        _captionIndex = (_captionIndex + 1) % widget.captions!.length;
      });
    });
  }

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: Alignment.centerLeft,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
        margin: const EdgeInsets.only(right: 24, left: 12),
        decoration: BoxDecoration(
          color: const Color.fromRGBO(58, 12, 140, 0.9),
          borderRadius: BorderRadius.circular(12),
        ),
        child: IntrinsicWidth(
          child: Row(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              _DotWidget(
                controller: _controller,
                interval: const Interval(0.0, 0.3),
              ),
              const SizedBox(width: 3),
              _DotWidget(
                controller: _controller,
                interval: const Interval(0.3, 0.6),
              ),
              const SizedBox(width: 3),
              _DotWidget(
                controller: _controller,
                interval: const Interval(0.6, 0.9),
              ),
              if (widget.captions != null && widget.captions!.isNotEmpty) ...[
                const SizedBox(width: 8),
                AnimatedSwitcher(
                  duration: const Duration(milliseconds: 250),
                  switchInCurve: Curves.easeOut,
                  switchOutCurve: Curves.easeIn,
                  child: Text(
                    widget.captions![_captionIndex],
                    key: ValueKey(_captionIndex),
                    style: Theme.of(
                      context,
                    ).textTheme.bodySmall?.copyWith(color: Colors.white),
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _DotWidget extends StatelessWidget {
  final AnimationController controller;
  final Interval interval;

  const _DotWidget({required this.controller, required this.interval});

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: controller,
      builder: (context, child) {
        final animation = CurvedAnimation(parent: controller, curve: interval);
        return Transform.translate(
          offset: Offset(0, -2 * animation.value),
          child: Container(
            width: 6,
            height: 6,
            decoration: const BoxDecoration(
              color: Colors.white,
              shape: BoxShape.circle,
            ),
          ),
        );
      },
    );
  }
}
